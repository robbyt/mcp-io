package main

import (
	"context"
	"sync"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/examples/simple_dungeon_master/crypt"
	"github.com/robbyt/mcp-io/examples/simple_dungeon_master/dice"
	"github.com/robbyt/mcp-io/examples/simple_dungeon_master/narrative"
)

// GameState coordinates game components with sub-states for narrative, dice, and cryptography.
type GameState struct {
	narrative *narrative.State                                                                 // Narrative engine with history and skill check state
	dice      *dice.State                                                                      // Dice roller with roll history
	crypt     *crypt.State                                                                     // Cryptographic operations for encrypted data validation
	config    *Config                                                                          // Runtime configuration options
	llmFunc   func(context.Context, mcpio.RequestContext, string, int, string) (string, error) // LLM function for narrative generation (injectable for testing)

	// turnCounterMutex protects turnCounter
	turnCounterMutex sync.RWMutex
	turnCounter      int // protected by turnCounterMutex; used for encryption nonce and game state progression
}

type Config struct {
	narrativeMaxTokens     int    // Max tokens for narrative generation
	summarizationThreshold int    // Number of recent entries before summarization
	summarizeMaxTokens     int    // Max tokens for summarization
	preferredModel         string // The LLM model the client should use when receiving sampling hint responses from this server (e.g., "llama3.2")
}

// GameStateResponse is the response type for the get_state tool (without mutex)
type GameStateResponse struct {
	CurrentTurnNumber  int               `json:"currentTurnNumber"  jsonschema:"Current turn number in the game"`
	TurnHistory        []*narrative.Turn `json:"turnHistory"        jsonschema:"Complete history of past turns with narrative and skill check outcomes"`
	Summary            string            `json:"summary"            jsonschema:"Summary of older narrative history"`
	SkillCheckRequired int               `json:"skillCheckRequired" jsonschema:"Pending skill check requirement (0 = none, 1-20 = required roll)"`
}

// GameStateTool returns the current game state, for use as an MCP tool to peak inside the internal game state
func (state *GameState) GameStateTool(_ context.Context, _ mcpio.RequestContext, _ struct{}) (GameStateResponse, error) {
	return GameStateResponse{
		CurrentTurnNumber:  state.getTurnCounter(),
		Summary:            state.narrative.GetSummary(),
		SkillCheckRequired: state.narrative.GetPendingSkillCheck(),
		TurnHistory:        state.narrative.GetTurns(),
	}, nil
}

// getTurnCounter returns the current turn counter in a thread-safe manner
func (state *GameState) getTurnCounter() int {
	state.turnCounterMutex.RLock()
	defer state.turnCounterMutex.RUnlock()
	return state.turnCounter
}

// incrementTurnCounter increments the turn counter in a thread-safe manner
func (state *GameState) incrementTurnCounter() {
	state.turnCounterMutex.Lock()
	defer state.turnCounterMutex.Unlock()
	state.turnCounter++
}
