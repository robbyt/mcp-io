package main

import (
	"context"
	"fmt"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/examples/simple_dungeon_master/dice"
	"github.com/robbyt/mcp-io/examples/simple_dungeon_master/narrative"
)

// RollDiceTool simulates rolling a 20-sided dice and stores it in history
// Uses turn-based roll (idempotent - multiple calls return the same roll for this turn)
func (state *GameState) RollDiceTool(ctx context.Context, input dice.RollInput) (dice.Roll, error) {
	mcpio.LogDebug(ctx, "RollDiceTool called", map[string]any{"input": input}) //nolint:errcheck
	pendingCheck := state.narrative.GetPendingSkillCheck()

	// Perform turn-based roll (idempotent for this turn)
	roll := state.dice.RollTurn(state.turnCounter)

	// Encrypt result if skill check was pending
	if pendingCheck > 0 {
		rollData := map[string]int{"result": roll.Result}
		encrypted, err := state.crypt.Encrypt(rollData, state.turnCounter)
		if err != nil {
			return dice.Roll{}, fmt.Errorf("failed to encrypt roll: %w", err)
		}

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
	var pendingCheckResult int
	var passedSkillCheck bool
	var nextSkillCheck int

	if pendingCheck > 0 {
		// Get encrypted roll from input
		encryptedRoll := input.EncryptedData

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
				ErrorMessage: fmt.Sprintf("Unable to decrypt encryptedData: %v", err),
			}, err
		}

		pendingCheckResult = rollData["result"]
		if pendingCheckResult < 1 || pendingCheckResult > 20 {
			return narrative.Response{
				TurnNumber:   state.turnCounter,
				ErrorMessage: "Invalid roll result: must be 1-20",
			}, fmt.Errorf("invalid roll result: %d", pendingCheckResult)
		}

		// Determine pass/fail
		passedSkillCheck = pendingCheckResult >= pendingCheck

		// Clear the pending check after successful validation
		state.narrative.ClearPendingSkillCheck()

		//nolint:errcheck
		mcpio.LogDebug(ctx, "Skill check validated", map[string]any{
			"result": pendingCheckResult,
			"passed": passedSkillCheck,
		})
	}

	// Determine if a new skill check is needed for next turn (regardless of whether there was a pending check)
	if nextSkillCheck == 0 {
		nextSkillCheck = state.dice.DecideNextSkillCheckDifficulty(state.turnCounter)
	}

	// PHASE 2: Build prompt config and generate draft narrative
	promptConfig := narrative.PromptConfig{
		TurnNumber:             state.turnCounter,
		InputAction:            input.Action,
		InputPreviousNarrative: input.PreviousNarrative,
		PendingCheck:           pendingCheck > 0,
		PendingCheckResult:     pendingCheckResult,
		PassedCheck:            passedSkillCheck,
		NextSkillCheck:         nextSkillCheck,
	}

	// Generate draft narrative
	draftPrompt := state.narrative.BuildTurnPrompt(promptConfig)
	draftNarrative, err := state.llmFunc(ctx, draftPrompt, state.config.narrativeMaxTokens, state.config.preferredModel)
	if err != nil {
		mcpio.LogError(ctx, "LLM error generating draft", map[string]any{"error": err.Error()}) //nolint:errcheck
		return narrative.Response{
			TurnNumber:   state.turnCounter,
			ErrorMessage: fmt.Sprintf("LLM error: %v", err),
		}, err
	}

	// Review and validate draft narrative
	var narrativeText string
	if state.turnCounter == 0 {
		narrativeText = draftNarrative
		mcpio.LogDebug(ctx, "Skipping narrative review for turn 0", nil) //nolint:errcheck
	} else {
		reviewPrompt := state.narrative.BuildReviewPrompt(promptConfig, draftNarrative)
		narrativeText, err = state.llmFunc(ctx, reviewPrompt, state.config.narrativeMaxTokens, state.config.preferredModel)
		if err != nil {
			mcpio.LogError(ctx, "LLM error during review", map[string]any{"error": err.Error()}) //nolint:errcheck
			// Fall back to draft if review fails
			narrativeText = draftNarrative
		}
	}

	// Record the turn and increment counter
	state.narrative.AddTurn(narrativeText, promptConfig)
	state.turnCounter++

	// Log if skill check is required for next turn
	if nextSkillCheck > 0 {
		mcpio.LogDebug(ctx, "New skill check set for next turn", map[string]any{"nextSkillCheck": nextSkillCheck}) //nolint:errcheck
	}

	// PHASE 3: Summarize if history is growing too long
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
