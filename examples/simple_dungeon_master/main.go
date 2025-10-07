package main

import (
	"context"
	"log"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/examples/simple_dungeon_master/crypt"
	"github.com/robbyt/mcp-io/examples/simple_dungeon_master/dice"
	"github.com/robbyt/mcp-io/examples/simple_dungeon_master/narrative"
)

const (
	dungeonMasterToolDesc = `Process player actions and generate narrative.

Usage patterns:
1. New action: {"action": "yell at the troll"} → Returns narrative. If skillCheckRequired > 0, call roll_d20 next.
2. Continue pending action: {"encryptedData": "..."} → Completes the previous action with roll result.

Always show the narrative response to the user.`

	rollD20ToolDesc = `Roll a 20-sided dice (D20) for skill checks.

Call when dungeon_master returns skillCheckRequired > 0.
Returns a continuation token (encryptedData) that you pass to dungeon_master WITHOUT an action to complete the pending turn.`

	debugGameStateToolDesc = `Return the current internal game state for debugging purposes.`
)

func buildHandler(gameState *GameState) (*mcpio.Handler, error) {
	return mcpio.NewToolHandler(
		mcpio.WithName("example-dungeon-master"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("dungeon_master", dungeonMasterToolDesc, gameState.NarrativeActionTool),
		mcpio.WithTool(dice.ToolName, rollD20ToolDesc, gameState.RollDiceTool),
		mcpio.WithTool("debug_gameState", debugGameStateToolDesc, gameState.GameStateTool),
	)
}

func main() {
	// Initialize cryptographic state for encrypted data validation
	cryptState, err := crypt.NewState()
	if err != nil {
		log.Fatalf("Failed to initialize crypto state: %v", err)
	}

	gameState := &GameState{
		narrative: narrative.NewState(),
		dice: dice.NewState(&dice.Config{
			GracePeriodMin:      1,
			GracePeriodMax:      3,
			SkillCheckFrequency: 0.25,
		}),
		crypt:   cryptState,
		llmFunc: createLLMMessage,
		config: &Config{
			narrativeMaxTokens:     200,
			summarizeMaxTokens:     1000,
			summarizationThreshold: 10,
			preferredModel:         "gpt-oss",
		},
	}
	handler, err := buildHandler(gameState)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	if err := handler.ServeStdio(context.Background(), nil, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
