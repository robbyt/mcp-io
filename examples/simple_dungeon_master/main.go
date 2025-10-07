package main

import (
	"context"
	"flag"
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
	modelFlag := flag.String("model", "gpt-oss", "LLM model to use for narrative generation")
	narrativeMaxTokens := flag.Int("narrative-max-tokens", 200, "Max tokens for narrative generation")
	summarizeMaxTokens := flag.Int("summarize-max-tokens", 1000, "Max tokens for internal summarization")
	summarizeThreshold := flag.Int("summarize-threshold", 10, "Number of turns before running the summarization")
	gracePeriodMin := flag.Int("grace-period-min", 1, "Min turns before skill checks begin (random value chosen between min-max)")
	gracePeriodMax := flag.Int("grace-period-max", 3, "Max turns before skill checks begin")
	skillCheckFrequency := flag.Float64("skill-check-frequency", 0.25, "Probability of skill check per turn (0.0-1.0)")
	flag.Parse()

	// Initialize cryptographic state for encrypted data validation
	cryptState, err := crypt.NewState()
	if err != nil {
		log.Fatalf("Failed to initialize crypto state: %v", err)
	}

	gameState := &GameState{
		narrative: narrative.NewState(),
		dice: dice.NewState(&dice.Config{
			GracePeriodMin:      *gracePeriodMin,
			GracePeriodMax:      *gracePeriodMax,
			SkillCheckFrequency: *skillCheckFrequency,
		}),
		crypt:   cryptState,
		llmFunc: createLLMMessage,
		config: &Config{
			narrativeMaxTokens:     *narrativeMaxTokens,
			summarizeMaxTokens:     *summarizeMaxTokens,
			summarizationThreshold: *summarizeThreshold,
			preferredModel:         *modelFlag,
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
