package dice

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRoll verifies that NewRoll creates a valid roll in range 1-20
func TestNewRoll(t *testing.T) {
	t.Parallel()

	for range 100 {
		roll := NewRoll()
		require.NotNil(t, roll)
		assert.GreaterOrEqual(t, roll.Result, 1)
		assert.LessOrEqual(t, roll.Result, 20)
	}
}

// TestNewState verifies State initialization
func TestNewState(t *testing.T) {
	t.Parallel()

	state := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})
	require.NotNil(t, state)

	// Verify we can roll immediately
	roll := state.Roll()
	require.NotNil(t, roll.Result)
	assert.GreaterOrEqual(t, roll.Result, 1)
	assert.LessOrEqual(t, roll.Result, 20)
}

// TestRoll_AllValues verifies that the dice eventually produces all values 1-20
func TestRoll_AllValues(t *testing.T) {
	t.Parallel()
	diceState := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})
	seen := make(map[int]bool)

	assert.Eventually(t, func() bool {
		result := diceState.Roll()
		assert.GreaterOrEqual(t, result.Result, 1)
		assert.LessOrEqual(t, result.Result, 20)

		seen[result.Result] = true
		return len(seen) == 20
	}, 5*time.Second, 10*time.Millisecond,
		"Expected to see all 20 dice values")
}

// TestRoll_Distribution verifies dice rolls have uniform distribution across 1-20 range
func TestRoll_Distribution(t *testing.T) {
	t.Parallel()
	diceState := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})
	counts := make(map[int]int)
	totalRolls := 10000

	for range totalRolls {
		result := diceState.Roll()
		counts[result.Result]++
	}

	// Each number should appear roughly 500 times (10000/20)
	// Allow variance of 50% in either direction
	expectedFreq := totalRolls / 20
	minAcceptable := expectedFreq / 2 // At least 250
	maxAcceptable := expectedFreq * 2 // At most 1000

	for i := 1; i <= 20; i++ {
		count := counts[i]
		assert.GreaterOrEqual(t, count, minAcceptable,
			"Value %d appeared %d times (expected ~%d)", i, count, expectedFreq)
		assert.LessOrEqual(t, count, maxAcceptable,
			"Value %d appeared %d times (expected ~%d)", i, count, expectedFreq)
	}
}

// TestRoll_Concurrent verifies thread-safe roll operations
func TestRoll_Concurrent(t *testing.T) {
	t.Parallel()
	state := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})
	const goroutines = 100
	const rollsPerGoroutine = 100

	var wg sync.WaitGroup

	// Launch concurrent rollers
	for range goroutines {
		wg.Go(func() {
			for range rollsPerGoroutine {
				roll := state.Roll()
				require.NotNil(t, roll.Result)
				assert.GreaterOrEqual(t, roll.Result, 1)
				assert.LessOrEqual(t, roll.Result, 20)
			}
		})
	}

	wg.Wait()

	// Verify final roll still works
	roll := state.Roll()
	require.NotNil(t, roll.Result)
	assert.GreaterOrEqual(t, roll.Result, 1)
	assert.LessOrEqual(t, roll.Result, 20)
}

// TestDecideSkillCheckDifficulty_Probability verifies skill checks occur at the configured frequency
func TestDecideSkillCheckDifficulty_Probability(t *testing.T) {
	t.Parallel()
	diceState := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})
	checksRequired := 0
	totalCalls := 10000

	for i := range totalCalls {
		result := diceState.DecideNextSkillCheckDifficulty(i)
		if result > 0 {
			checksRequired++
		}
	}

	// Should be approximately 25% (2500 out of 10000)
	// Allow variance of +/- 5% (2000-3000)
	assert.GreaterOrEqual(t, checksRequired, 2000, "Should require skill checks roughly 25%% of the time")
	assert.LessOrEqual(t, checksRequired, 3000, "Should require skill checks roughly 25%% of the time")
}

// TestDecideSkillCheckDifficulty_ScalingDifficulty verifies difficulty scales with turn progression
func TestDecideSkillCheckDifficulty_ScalingDifficulty(t *testing.T) {
	t.Parallel()
	diceState := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 1.0}) // Always require check

	// Early turns (0-1): difficulty should be 5
	for i := 0; i <= 1; i++ {
		result := diceState.DecideNextSkillCheckDifficulty(i)
		assert.Equal(t, 5, result, "Turn %d should have difficulty 5", i)
	}

	// Turns 2-3: difficulty should be 6
	for i := 2; i <= 3; i++ {
		result := diceState.DecideNextSkillCheckDifficulty(i)
		assert.Equal(t, 6, result, "Turn %d should have difficulty 6", i)
	}

	// Mid turns (~10): difficulty should be ~10
	result := diceState.DecideNextSkillCheckDifficulty(10)
	assert.Equal(t, 10, result, "Turn 10 should have difficulty 10")

	// Turn 20: difficulty should be 15
	result = diceState.DecideNextSkillCheckDifficulty(20)
	assert.Equal(t, 15, result, "Turn 20 should have difficulty 15")

	// Late turns (30+): difficulty should be capped at 18
	for i := 30; i <= 50; i++ {
		result := diceState.DecideNextSkillCheckDifficulty(i)
		assert.Equal(t, 18, result, "Turn %d should be capped at difficulty 18", i)
	}

	// Test with frequency < 1.0 to verify 0s are still returned
	diceState = NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})
	foundZero := false
	foundDifficulty := false

	for i := range 100 {
		result := diceState.DecideNextSkillCheckDifficulty(i)
		if result == 0 {
			foundZero = true
		} else {
			foundDifficulty = true
		}
	}

	assert.True(t, foundZero, "Should sometimes return 0 (no check needed)")
	assert.True(t, foundDifficulty, "Should sometimes return difficulty check")
}

// TestStateRoll verifies State.Roll() produces valid rolls
func TestStateRoll(t *testing.T) {
	t.Parallel()
	state := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})

	// Perform multiple rolls
	for range 10 {
		roll := state.Roll()
		assert.GreaterOrEqual(t, roll.Result, 1)
		assert.LessOrEqual(t, roll.Result, 20)
	}
}

// TestRollTurn_Idempotent verifies that multiple calls to RollTurn with the same turn return the same roll
func TestRollTurn_Idempotent(t *testing.T) {
	t.Parallel()
	state := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})

	turnNumber := 5

	// Roll for turn 5 multiple times
	roll1 := state.RollTurn(turnNumber)
	roll2 := state.RollTurn(turnNumber)
	roll3 := state.RollTurn(turnNumber)

	// All should return the exact same result
	assert.Equal(t, roll1.Result, roll2.Result, "Second roll should match first roll")
	assert.Equal(t, roll1.Result, roll3.Result, "Third roll should match first roll")

	// Verify rolls are valid
	assert.GreaterOrEqual(t, roll1.Result, 1)
	assert.LessOrEqual(t, roll1.Result, 20)
}

// TestRollTurn_DifferentTurns verifies that different turns get different rolls
func TestRollTurn_DifferentTurns(t *testing.T) {
	t.Parallel()
	state := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})

	// Roll for multiple turns
	rolls := make(map[int]int)
	for turn := 0; turn < 10; turn++ {
		roll := state.RollTurn(turn)
		rolls[turn] = roll.Result
		assert.GreaterOrEqual(t, roll.Result, 1)
		assert.LessOrEqual(t, roll.Result, 20)
	}

	// Verify rolls are tracked per turn
	assert.Len(t, rolls, 10, "Should have 10 different turn rolls")

	// Verify idempotency - re-rolling same turns returns same values
	for turn := 0; turn < 10; turn++ {
		roll := state.RollTurn(turn)
		assert.Equal(t, rolls[turn], roll.Result, "Re-rolling turn %d should return same value", turn)
	}
}

// TestRollTurn_Concurrent verifies thread-safe turn-based rolling
func TestRollTurn_Concurrent(t *testing.T) {
	t.Parallel()
	state := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})

	const goroutines = 100
	turnNumber := 42

	var wg sync.WaitGroup
	results := make([]int, goroutines)

	// Launch concurrent rollers for the same turn
	for i := range goroutines {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			roll := state.RollTurn(turnNumber)
			results[idx] = roll.Result
		}(i)
	}

	wg.Wait()

	// All goroutines should get the same result (idempotent)
	firstResult := results[0]
	for i, result := range results {
		assert.Equal(t, firstResult, result, "Goroutine %d got different result", i)
	}

	// Verify result is valid
	assert.GreaterOrEqual(t, firstResult, 1)
	assert.LessOrEqual(t, firstResult, 20)
}
