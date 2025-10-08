package narrative

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function for tests to add turns with old signature
func addTestTurn(state *State, action, narrative string, nextSkillCheck int) *Turn {
	config := PromptConfig{
		InputAction:    action,
		NextSkillCheck: nextSkillCheck,
	}
	return state.AddTurn(narrative, config)
}

// TestNewState verifies State initialization
func TestNewState(t *testing.T) {
	t.Parallel()

	state := NewState()
	require.NotNil(t, state)
	assert.Equal(t, 0, state.GetPendingSkillCheck(), "New state should have no pending skill check")
	assert.Empty(t, state.GetTurns(), "New state should have no turns")
	assert.Empty(t, state.GetSummary(), "New state should have no summary")
	assert.Empty(t, state.GetEntries(), "New state should have no entries")
}

// TestClearPendingSkillCheck verifies resetting pending skill check
func TestClearPendingSkillCheck(t *testing.T) {
	t.Parallel()

	state := NewState()

	// Add a turn with a skill check to set pendingSkillCheck
	addTestTurn(state, "test action", "test narrative", 15)
	assert.Equal(t, 15, state.GetPendingSkillCheck(), "Should have pending skill check after AddTurn")

	// Clear it
	state.ClearPendingSkillCheck()
	assert.Equal(t, 0, state.GetPendingSkillCheck(), "Should have no pending skill check after clear")
}

// TestGetPendingSkillCheck verifies reading pending skill check value
func TestGetPendingSkillCheck(t *testing.T) {
	t.Parallel()

	state := NewState()
	assert.Equal(t, 0, state.GetPendingSkillCheck(), "Should start at 0")

	// Set via AddTurn
	addTestTurn(state, "action", "narrative", 12)
	assert.Equal(t, 12, state.GetPendingSkillCheck(), "Should return 12")

	// Clear it
	state.ClearPendingSkillCheck()
	assert.Equal(t, 0, state.GetPendingSkillCheck(), "Should return 0 after clear")
}

// TestAddTurn verifies adding turns and skill check persistence
func TestAddTurn(t *testing.T) {
	t.Parallel()

	state := NewState()

	// Add turn without skill check
	turn1 := addTestTurn(state, "walk forward", "you walk forward", 0)
	require.NotNil(t, turn1)
	assert.Equal(t, "walk forward", turn1.Config.InputAction)
	assert.Equal(t, "you walk forward", turn1.Narrative)
	assert.Equal(t, 0, turn1.Config.NextSkillCheck)
	assert.Equal(t, 0, state.GetPendingSkillCheck(), "No pending check when skill check is 0")

	// Add turn with skill check
	turn2 := addTestTurn(state, "climb wall", "you attempt to climb", 14)
	require.NotNil(t, turn2)
	assert.Equal(t, "climb wall", turn2.Config.InputAction)
	assert.Equal(t, "you attempt to climb", turn2.Narrative)
	assert.Equal(t, 14, turn2.Config.NextSkillCheck)
	assert.Equal(t, 14, state.GetPendingSkillCheck(), "Should set pending check for next turn")

	// Verify both turns are stored
	turns := state.GetTurns()
	assert.Len(t, turns, 2)
	assert.Equal(t, turn1, turns[0])
	assert.Equal(t, turn2, turns[1])
}

// TestGetTurns verifies retrieving turn history
func TestGetTurns(t *testing.T) {
	t.Parallel()

	state := NewState()
	assert.Empty(t, state.GetTurns(), "Should start empty")

	// Add multiple turns
	turn1 := addTestTurn(state, "action1", "narrative1", 0)
	turn2 := addTestTurn(state, "action2", "narrative2", 10)
	turn3 := addTestTurn(state, "action3", "narrative3", 0)

	turns := state.GetTurns()
	assert.Len(t, turns, 3)
	assert.Equal(t, turn1, turns[0])
	assert.Equal(t, turn2, turns[1])
	assert.Equal(t, turn3, turns[2])
}

// TestGetEntries verifies formatted string entries
func TestGetEntries(t *testing.T) {
	t.Parallel()

	state := NewState()
	assert.Empty(t, state.GetEntries(), "Should start empty")

	// Add turns
	addTestTurn(state, "look around", "you see a door", 0)
	addTestTurn(state, "open door", "the door opens", 0)

	entries := state.GetEntries()
	assert.Len(t, entries, 2)
	assert.Equal(t, "Action: look around\nNarrative: you see a door", entries[0])
	assert.Equal(t, "Action: open door\nNarrative: the door opens", entries[1])
}

// TestShouldSummarize verifies threshold logic
func TestShouldSummarize(t *testing.T) {
	t.Parallel()

	state := NewState()
	threshold := 3

	// Below threshold
	assert.False(t, state.ShouldSummarize(threshold), "Should not summarize with 0 turns")

	addTestTurn(state, "action1", "narrative1", 0)
	assert.False(t, state.ShouldSummarize(threshold), "Should not summarize with 1 turn")

	addTestTurn(state, "action2", "narrative2", 0)
	addTestTurn(state, "action3", "narrative3", 0)
	assert.False(t, state.ShouldSummarize(threshold), "Should not summarize at threshold")

	// Above threshold
	addTestTurn(state, "action4", "narrative4", 0)
	assert.True(t, state.ShouldSummarize(threshold), "Should summarize above threshold")
}

// TestBuildSummaryPrompt verifies prompt generation with turns
func TestBuildSummaryPrompt(t *testing.T) {
	t.Parallel()

	state := NewState()

	// Add multiple turns
	addTestTurn(state, "find key", "you find a rusty key", 0)
	addTestTurn(state, "unlock door", "you unlock the door", 0)
	addTestTurn(state, "enter room", "you enter a dark room", 0)
	addTestTurn(state, "light torch", "you light a torch", 0)

	prompt := state.BuildSummaryPrompt()

	// Should include instructions
	assert.Contains(t, prompt, "Summarize these adventure events")

	// Should only include first half of turns (2 out of 4)
	assert.Contains(t, prompt, "Action: find key")
	assert.Contains(t, prompt, "Action: unlock door")
	assert.NotContains(t, prompt, "Action: enter room")
	assert.NotContains(t, prompt, "Action: light torch")
}

// TestBuildSummaryPrompt_OddNumberOfTurns verifies handling of odd turn counts
func TestBuildSummaryPrompt_OddNumberOfTurns(t *testing.T) {
	t.Parallel()

	state := NewState()

	// Add 5 turns (splitPoint will be 2)
	addTestTurn(state, "action1", "narrative1", 0)
	addTestTurn(state, "action2", "narrative2", 0)
	addTestTurn(state, "action3", "narrative3", 0)
	addTestTurn(state, "action4", "narrative4", 0)
	addTestTurn(state, "action5", "narrative5", 0)

	prompt := state.BuildSummaryPrompt()

	// First 2 should be included
	assert.Contains(t, prompt, "action1")
	assert.Contains(t, prompt, "action2")

	// Last 3 should not be included
	assert.NotContains(t, prompt, "action3")
	assert.NotContains(t, prompt, "action4")
	assert.NotContains(t, prompt, "action5")
}

// TestRecordSummary verifies summary appending and turn trimming
func TestRecordSummary(t *testing.T) {
	t.Parallel()

	state := NewState()

	// Add turns
	addTestTurn(state, "action1", "narrative1", 0)
	addTestTurn(state, "action2", "narrative2", 0)
	addTestTurn(state, "action3", "narrative3", 0)
	addTestTurn(state, "action4", "narrative4", 0)

	// Record first summary
	state.RecordSummary("First part of adventure")
	assert.Equal(t, "First part of adventure", state.GetSummary())

	// Should keep only second half of turns (2 out of 4)
	turns := state.GetTurns()
	assert.Len(t, turns, 2)
	assert.Equal(t, "action3", turns[0].Config.InputAction)
	assert.Equal(t, "action4", turns[1].Config.InputAction)

	// Add more turns
	addTestTurn(state, "action5", "narrative5", 0)
	addTestTurn(state, "action6", "narrative6", 0)

	// Record second summary - should append
	state.RecordSummary("Second part of adventure")
	assert.Equal(t, "First part of adventure Second part of adventure", state.GetSummary())

	// Should again keep only second half (2 from previous + 2 new = 4 total, keep 2)
	turns = state.GetTurns()
	assert.Len(t, turns, 2)
	assert.Equal(t, "action5", turns[0].Config.InputAction)
	assert.Equal(t, "action6", turns[1].Config.InputAction)
}

// TestGetSummary verifies reading summary value
func TestGetSummary(t *testing.T) {
	t.Parallel()

	state := NewState()
	assert.Empty(t, state.GetSummary(), "Should start empty")

	addTestTurn(state, "action", "narrative", 0)
	state.RecordSummary("Test summary")
	assert.Equal(t, "Test summary", state.GetSummary())
}

// TestBuildTurnPrompt_BasicScenario verifies prompt with minimal state
func TestBuildTurnPrompt_BasicScenario(t *testing.T) {
	t.Parallel()

	state := NewState()
	config := &PromptConfig{
		InputAction: "look around",
	}
	prompt := state.BuildTurnPrompt(*config)

	// Should include the action
	assert.Contains(t, prompt, "look around")

	// Should include narration instruction
	assert.Contains(t, prompt, "Narrate")

	// Should not include skill check instructions
	assert.NotContains(t, prompt, "skill check")

	// Should not include history sections
	assert.NotContains(t, prompt, "Previous events")
	assert.NotContains(t, prompt, "Recent")
}

// TestBuildTurnPrompt_WithRollContext verifies roll context inclusion
func TestBuildTurnPrompt_WithRollContext(t *testing.T) {
	t.Parallel()

	state := NewState()
	config := &PromptConfig{
		InputAction:        "continue forward",
		PendingCheck:       true,
		PendingCheckResult: 15,
		PassedCheck:        true,
	}
	prompt := state.BuildTurnPrompt(*config)

	// Should contain roll result
	assert.Contains(t, prompt, "rolled 15")
	assert.Contains(t, prompt, "SUCCEEDED")
}

// TestBuildTurnPrompt_WithSkillCheck verifies skill check requirement
func TestBuildTurnPrompt_WithSkillCheck(t *testing.T) {
	t.Parallel()

	state := NewState()
	config := &PromptConfig{
		InputAction:    "climb the wall",
		NextSkillCheck: 14,
	}
	prompt := state.BuildTurnPrompt(*config)

	// Should include skill check instructions
	assert.Contains(t, prompt, "skill check")
	assert.Contains(t, prompt, "dramatic")
}

// TestBuildTurnPrompt_WithSummary verifies summary inclusion
func TestBuildTurnPrompt_WithSummary(t *testing.T) {
	t.Parallel()

	state := NewState()
	addTestTurn(state, "action1", "narrative1", 0)
	state.RecordSummary("You explored the dungeon and found treasures")

	config := &PromptConfig{
		InputAction: "continue exploring",
	}
	prompt := state.BuildTurnPrompt(*config)

	// Should include summary section
	assert.Contains(t, prompt, "Canonical summary of recent events")
	assert.Contains(t, prompt, "You explored the dungeon and found treasures")
}

// TestBuildTurnPrompt_WithHistory verifies turn history inclusion
func TestBuildTurnPrompt_WithHistory(t *testing.T) {
	t.Parallel()

	state := NewState()
	addTestTurn(state, "open door", "you open the door", 0)
	addTestTurn(state, "enter room", "you enter a dark room", 0)

	config := &PromptConfig{
		InputAction: "light torch",
	}
	prompt := state.BuildTurnPrompt(*config)

	// Should include recent events
	assert.Contains(t, prompt, "Recent")
	assert.Contains(t, prompt, "open door")
	assert.Contains(t, prompt, "you open the door")
	assert.Contains(t, prompt, "enter room")
	assert.Contains(t, prompt, "you enter a dark room")
}

// TestBuildTurnPrompt_Complete verifies all elements together
func TestBuildTurnPrompt_Complete(t *testing.T) {
	t.Parallel()

	state := NewState()
	addTestTurn(state, "find key", "you find a key", 0)
	state.RecordSummary("You started your adventure")
	addTestTurn(state, "unlock door", "you unlock the door", 12)

	config := &PromptConfig{
		InputAction:        "enter room",
		PendingCheck:       true,
		PendingCheckResult: 15,
		PassedCheck:        true,
		NextSkillCheck:     14,
	}
	prompt := state.BuildTurnPrompt(*config)

	// Should have all components
	assert.Contains(t, prompt, "rolled 15")
	assert.Contains(t, prompt, "SUCCEEDED")
	assert.Contains(t, prompt, "skill check")
	assert.Contains(t, prompt, "Canonical summary of recent events")
	assert.Contains(t, prompt, "You started your adventure")
	assert.Contains(t, prompt, "Recent turns:")
	assert.Contains(t, prompt, "unlock door")
	assert.Contains(t, prompt, "enter room")
}

// TestConcurrentAddTurn verifies thread-safe turn addition
func TestConcurrentAddTurn(t *testing.T) {
	t.Parallel()

	state := NewState()
	const goroutines = 100

	var wg sync.WaitGroup

	// Launch concurrent turn additions
	for i := range goroutines {
		wg.Go(func() {
			addTestTurn(state, "action", "narrative", i%20)
		})
	}

	wg.Wait()

	// Verify all turns were added
	turns := state.GetTurns()
	assert.Len(t, turns, goroutines, "All turns should be recorded")
}

// TestConcurrentRead verifies thread-safe reading operations
func TestConcurrentRead(t *testing.T) {
	t.Parallel()

	state := NewState()

	// Populate state
	for range 10 {
		addTestTurn(state, "action", "narrative", 0)
	}

	// RecordSummary trims turns to half, so we'll have 5 turns left
	state.RecordSummary("Test summary")

	var wg sync.WaitGroup
	const goroutines = 50

	// Launch concurrent reads
	for range goroutines {
		wg.Go(func() {
			// Multiple read operations
			_ = state.GetTurns()
			_ = state.GetEntries()
			_ = state.GetSummary()
			_ = state.GetPendingSkillCheck()
			_ = state.ShouldSummarize(5)
			config := &PromptConfig{InputAction: "action"}
			_ = state.BuildTurnPrompt(*config)
			_ = state.BuildSummaryPrompt()
		})
	}

	wg.Wait()

	// Verify state remains consistent (RecordSummary keeps second half, so 5 turns)
	assert.Len(t, state.GetTurns(), 5)
	assert.Equal(t, "Test summary", state.GetSummary())
}

// TestConcurrentMixed verifies thread-safe mixed read/write operations
func TestConcurrentMixed(t *testing.T) {
	t.Parallel()

	state := NewState()
	var wg sync.WaitGroup
	const goroutines = 50

	// Mix of readers and writers
	for i := range goroutines {
		wg.Go(func() {
			if i%2 == 0 {
				// Write operations
				addTestTurn(state, "action", "narrative", i%20)
			} else {
				// Read operations
				_ = state.GetTurns()
				config := &PromptConfig{InputAction: "action"}
				_ = state.BuildTurnPrompt(*config)
			}
		})
	}

	wg.Wait()

	// Verify state is consistent
	turns := state.GetTurns()
	assert.LessOrEqual(t, len(turns), goroutines, "Should have recorded some turns")
}
