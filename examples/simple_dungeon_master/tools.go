package main

import (
	"context"
	"fmt"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/examples/simple_dungeon_master/dice"
	"github.com/robbyt/mcp-io/examples/simple_dungeon_master/narrative"
)

// RollDiceTool simulates rolling a 20-sided dice and stores it in history
func (state *GameState) RollDiceTool(ctx context.Context, input dice.RollInput) (dice.Roll, error) {
	mcpio.LogDebug(ctx, "RollDiceTool called", map[string]any{"input": input}) //nolint:errcheck
	pendingCheck := state.narrative.GetPendingSkillCheck()

	// If skill check is pending, check for re-rolling (anti-cheat)
	if pendingCheck > 0 {
		if state.narrative.GetLastEncryptedRoll() != "" {
			return dice.Roll{}, fmt.Errorf("already rolled for this skill check: cannot re-roll")
		}
	}

	// Perform the roll
	roll := state.dice.Roll()

	// Encrypt result if skill check was pending
	if pendingCheck > 0 {
		rollData := map[string]int{"result": roll.Result}
		encrypted, err := state.crypt.Encrypt(rollData, state.turnCounter)
		if err != nil {
			return dice.Roll{}, fmt.Errorf("failed to encrypt roll: %w", err)
		}

		// Store encrypted roll to prevent re-rolling
		state.narrative.SetLastEncryptedRoll(encrypted)
		roll.EncryptedData = encrypted
	}

	mcpio.LogDebug(ctx, "Dice rolled", map[string]any{"result": roll.Result, "encrypted": roll.EncryptedData != ""}) //nolint:errcheck

	return roll, nil
}

// NarrativeActionTool coordinates narrative generation across dice and narrative packages
func (state *GameState) NarrativeActionTool(ctx context.Context, input narrative.ActionInput) (narrative.Response, error) {
	mcpio.LogDebug(ctx, "NarrativeActionTool called", map[string]any{"input": input}) //nolint:errcheck
	// PHASE 1: Validate pending skill check from previous turn (if any)
	pendingCheck := state.narrative.GetPendingSkillCheck()
	var rollContext string

	if pendingCheck > 0 {
		// Get encrypted roll from input or storage (auto-consume)
		encryptedRoll := input.EncryptedData
		if encryptedRoll == "" {
			encryptedRoll = state.narrative.GetLastEncryptedRoll()
		}

		// Validate encrypted roll exists
		if encryptedRoll == "" {
			return narrative.Response{
				TurnNumber:         state.turnCounter,
				ErrorMessage:       fmt.Sprintf("Skill check required! Call the %s tool to get an encrypted roll, then include it in your next action.", dice.ToolName),
				SkillCheckRequired: pendingCheck,
			}, nil
		}

		// Decrypt and validate roll
		var rollData map[string]int
		if err := state.crypt.Decrypt(encryptedRoll, state.turnCounter, &rollData); err != nil {
			return narrative.Response{
				TurnNumber:   state.turnCounter,
				ErrorMessage: fmt.Sprintf("Invalid encrypted roll: %v", err),
			}, err
		}

		result := rollData["result"]
		if result < 1 || result > 20 {
			return narrative.Response{
				TurnNumber:   state.turnCounter,
				ErrorMessage: "Invalid roll result: must be 1-20",
			}, fmt.Errorf("invalid roll result: %d", result)
		}

		// Clear the encrypted roll (consumed)
		state.narrative.ClearLastEncryptedRoll()

		// Determine pass/fail
		passed := result >= pendingCheck
		if passed {
			rollContext = fmt.Sprintf("\nThe player rolled %d (required %d or higher) - they SUCCEEDED!\n", result, pendingCheck)
		} else {
			rollContext = fmt.Sprintf("\nThe player rolled %d (required %d or higher) - they FAILED!\n", result, pendingCheck)
		}

		// Clear the pending check after successful validation
		state.narrative.ClearPendingSkillCheck()

		//nolint:errcheck
		mcpio.LogDebug(ctx, "Skill check validated", map[string]any{
			"result":       result,
			"pendingCheck": pendingCheck,
			"passed":       passed,
		})
	}

	// PHASE 2: Decide if next turn requires a skill check
	nextSkillCheck := state.dice.DecideSkillCheckDifficulty(input.Action, state.turnCounter)

	// PHASE 3: Build prompt and generate narrative
	prompt := state.narrative.BuildTurnPrompt(input.Action, nextSkillCheck, rollContext)
	narrativeText, err := state.llmFunc(ctx, prompt, state.config.narrativeMaxTokens, state.config.preferredModel)
	if err != nil {
		mcpio.LogError(ctx, "LLM error", map[string]any{"error": err.Error()}) //nolint:errcheck
		return narrative.Response{
			TurnNumber:   state.turnCounter,
			ErrorMessage: fmt.Sprintf("LLM error: %v", err),
		}, err
	}

	// Record the turn and increment counter
	state.narrative.AddTurn(input.Action, narrativeText, nextSkillCheck, rollContext)
	state.turnCounter++

	// Clear encrypted roll for next turn if skill check is required
	if nextSkillCheck > 0 {
		state.narrative.ClearLastEncryptedRoll()
		mcpio.LogDebug(ctx, "New skill check set for next turn", map[string]any{"nextSkillCheck": nextSkillCheck}) //nolint:errcheck
	}

	// PHASE 4: Summarize if history is growing too long
	if state.narrative.ShouldSummarize(state.config.summarizationThreshold) {
		summaryPrompt := state.narrative.BuildSummaryPrompt()
		summaryText, err := state.llmFunc(ctx, summaryPrompt, state.config.summarizeMaxTokens, state.config.preferredModel)
		if err != nil {
			mcpio.LogError(ctx, "Summarization failed", map[string]any{"error": err.Error()}) //nolint:errcheck
		} else {
			state.narrative.RecordSummary(summaryText)
			mcpio.LogDebug(ctx, "History summarized", map[string]any{"turns": len(state.narrative.GetTurns())}) //nolint:errcheck
		}
	}

	return narrative.Response{
		Narrative:          narrativeText,
		TurnNumber:         state.turnCounter,
		SkillCheckRequired: nextSkillCheck,
	}, nil
}
