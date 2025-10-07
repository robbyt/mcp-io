package dice

import (
	"fmt"
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
	assert.Equal(t, 0, state.GetLastRollValue(), "New state should have no last roll")

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

// TestRoll_HistoryOverflow verifies roll history doesn't grow unbounded
func TestRoll_HistoryOverflow(t *testing.T) {
	t.Parallel()
	state := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})

	// Roll more than maxRollsStorage (1000)
	for range 1500 {
		state.Roll()
	}

	// Verify state is still functional
	roll := state.Roll()
	require.NotNil(t, roll.Result)
	assert.GreaterOrEqual(t, roll.Result, 1)
	assert.LessOrEqual(t, roll.Result, 20)
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
		result := diceState.DecideSkillCheckDifficulty("test action", i)
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
		result := diceState.DecideSkillCheckDifficulty("action", i)
		assert.Equal(t, 5, result, "Turn %d should have difficulty 5", i)
	}

	// Turns 2-3: difficulty should be 6
	for i := 2; i <= 3; i++ {
		result := diceState.DecideSkillCheckDifficulty("action", i)
		assert.Equal(t, 6, result, "Turn %d should have difficulty 6", i)
	}

	// Mid turns (~10): difficulty should be ~10
	result := diceState.DecideSkillCheckDifficulty("action", 10)
	assert.Equal(t, 10, result, "Turn 10 should have difficulty 10")

	// Turn 20: difficulty should be 15
	result = diceState.DecideSkillCheckDifficulty("action", 20)
	assert.Equal(t, 15, result, "Turn 20 should have difficulty 15")

	// Late turns (30+): difficulty should be capped at 18
	for i := 30; i <= 50; i++ {
		result := diceState.DecideSkillCheckDifficulty("action", i)
		assert.Equal(t, 18, result, "Turn %d should be capped at difficulty 18", i)
	}

	// Test with frequency < 1.0 to verify 0s are still returned
	diceState = NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})
	foundZero := false
	foundDifficulty := false

	for i := range 100 {
		result := diceState.DecideSkillCheckDifficulty("action", i)
		if result == 0 {
			foundZero = true
		} else {
			foundDifficulty = true
		}
	}

	assert.True(t, foundZero, "Should sometimes return 0 (no check needed)")
	assert.True(t, foundDifficulty, "Should sometimes return difficulty check")
}

func TestHandlePendingSkillCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		difficulty        int
		lastRoll          int
		expectError       bool
		expectRollContext bool
		expectSuccess     bool // if rollContext expected, was it success?
	}{
		{
			name:              "no skill check required",
			difficulty:        0,
			lastRoll:          10,
			expectError:       false,
			expectRollContext: false,
		},
		{
			name:              "skill check required but no roll",
			difficulty:        12,
			lastRoll:          0,
			expectError:       true,
			expectRollContext: false,
		},
		{
			name:              "roll succeeds exact match",
			difficulty:        10,
			lastRoll:          10,
			expectError:       false,
			expectRollContext: true,
			expectSuccess:     true,
		},
		{
			name:              "roll succeeds above requirement",
			difficulty:        10,
			lastRoll:          15,
			expectError:       false,
			expectRollContext: true,
			expectSuccess:     true,
		},
		{
			name:              "roll fails below requirement",
			difficulty:        15,
			lastRoll:          10,
			expectError:       false,
			expectRollContext: true,
			expectSuccess:     false,
		},
		{
			name:              "maximum values success",
			difficulty:        20,
			lastRoll:          20,
			expectError:       false,
			expectRollContext: true,
			expectSuccess:     true,
		},
		{
			name:              "minimum values success",
			difficulty:        1,
			lastRoll:          1,
			expectError:       false,
			expectRollContext: true,
			expectSuccess:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diceState := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})

			// Set up lastRoll in DiceState if needed
			if tt.lastRoll > 0 {
				// Manually create and set a roll
				diceState.mu.Lock()
				diceState.lastRoll = &Roll{Result: tt.lastRoll}
				diceState.mu.Unlock()
			}

			rollContext, err := diceState.HandlePendingSkillCheck(tt.difficulty)

			if tt.expectError {
				require.Error(t, err)
				assert.Empty(t, rollContext)
				assert.ErrorIs(t, err, ErrSkillCheckRequired)
			} else {
				assert.NoError(t, err)

				if tt.expectRollContext {
					assert.NotEmpty(t, rollContext)
					assert.Contains(t, rollContext, fmt.Sprintf("rolled %d", tt.lastRoll))
					assert.Contains(t, rollContext, fmt.Sprintf("required %d", tt.difficulty))

					if tt.expectSuccess {
						assert.Contains(t, rollContext, "SUCCEEDED")
					} else {
						assert.Contains(t, rollContext, "FAILED")
					}

					// Verify roll was consumed
					assert.Equal(t, 0, diceState.GetLastRollValue(), "Roll should be consumed after use")
				} else {
					assert.Empty(t, rollContext)
				}
			}
		})
	}
}

// TestHandlePendingSkillCheck_NoConsumptionWhenZero verifies roll is not consumed when no check required
func TestHandlePendingSkillCheck_NoConsumptionWhenZero(t *testing.T) {
	t.Parallel()
	state := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})

	// Create a roll
	roll := state.Roll()
	require.NotNil(t, roll.Result)
	rollValue := roll.Result

	// Verify roll is tracked
	assert.Equal(t, rollValue, state.GetLastRollValue())

	// Call with difficulty=0 (no check)
	context, err := state.HandlePendingSkillCheck(0)
	require.NoError(t, err)
	assert.Empty(t, context)

	// Verify roll was NOT consumed
	assert.Equal(t, rollValue, state.GetLastRollValue(), "Roll should not be consumed when no check required")

	// Now consume it with an actual check
	context, err = state.HandlePendingSkillCheck(1)
	require.NoError(t, err)
	assert.NotEmpty(t, context)

	// Verify roll was consumed
	assert.Equal(t, 0, state.GetLastRollValue(), "Roll should be consumed after actual check")
}

// TestHandlePendingSkillCheck_Concurrent verifies thread-safe skill check handling
func TestHandlePendingSkillCheck_Concurrent(t *testing.T) {
	t.Parallel()
	state := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})

	// Create a single roll
	roll := state.Roll()
	require.NotNil(t, roll.Result)

	var wg sync.WaitGroup
	successCount := 0
	errorCount := 0
	var mu sync.Mutex

	// Launch multiple goroutines trying to consume the same roll
	for range 10 {
		wg.Go(func() {
			_, err := state.HandlePendingSkillCheck(10)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errorCount++
			} else {
				successCount++
			}
		})
	}

	wg.Wait()

	// Only one should succeed, rest should fail
	assert.Equal(t, 1, successCount, "Exactly one goroutine should successfully consume the roll")
	assert.Equal(t, 9, errorCount, "Other goroutines should get an error")

	// Verify roll was consumed
	assert.Equal(t, 0, state.GetLastRollValue(), "Roll should be consumed")

	// Create new roll and verify it works
	newRoll := state.Roll()
	context, err := state.HandlePendingSkillCheck(newRoll.Result)
	require.NoError(t, err)
	assert.NotEmpty(t, context)
}

// TestStateRoll verifies State.Roll() produces valid rolls and updates history
func TestStateRoll(t *testing.T) {
	t.Parallel()
	state := NewState(&Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 0.25})

	// Perform multiple rolls
	for range 10 {
		roll := state.Roll()
		assert.GreaterOrEqual(t, roll.Result, 1)
		assert.LessOrEqual(t, roll.Result, 20)
	}

	// Verify history was updated
	assert.Len(t, state.history, 10)
	assert.NotNil(t, state.lastRoll)
	assert.Equal(t, state.history[9].Result, state.lastRoll.Result)
}
