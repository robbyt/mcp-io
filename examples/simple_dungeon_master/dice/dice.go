package dice

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
)

const ToolName = "roll_d20"

// ErrSkillCheckRequired is returned when a skill check is required but no roll is available
var ErrSkillCheckRequired = errors.New("skill check dice roll is required")

// Roll represents a D20 dice roll result
type Roll struct {
	Result        int    `json:"result"        jsonschema:"The dice roll result (1-20)"`
	EncryptedData string `json:"encryptedData" jsonschema:"Continuation token containing the encrypted roll. Pass this to dungeon_master (without an action) to continue the pending turn."`
}

// RollInput represents input for the roll_d20 tool
type RollInput struct{}

func NewRoll() *Roll {
	roll := rand.IntN(20) + 1 // Random number 1-20
	return &Roll{Result: roll}
}

// State represents a dice, including a history of rolls
type State struct {
	mu sync.RWMutex

	history         []*Roll
	maxRollsStorage int
	lastRoll        *Roll // Last unconsumed roll (nil if consumed or no roll yet)

	// Skill check configuration
	skillCheckFrequency float64 // Probability of skill check per action (0.0-1.0)
	gracePeriodMin      int     // Minimum turns before skill checks can occur
	gracePeriodMax      int     // Maximum turns before skill checks can occur
	gracePeriodUntil    int     // Randomly chosen turn when skill checks activate
}

// NewState creates a new Dice instance with skill check configuration
func NewState(c *Config) *State {
	rollHistorySize := 1000
	diceHist := make([]*Roll, 0, rollHistorySize)

	// Validate and clamp grace period values
	gracePeriodMin := max(0, c.GracePeriodMin)
	gracePeriodMax := c.GracePeriodMax

	// Special case: if both are 0, keep them both 0 (no grace period)
	// Otherwise ensure max is at least min+1
	if gracePeriodMin != 0 || gracePeriodMax != 0 {
		gracePeriodMax = max(gracePeriodMin+1, gracePeriodMax)
	}

	// Validate and clamp skill check frequency to 0.0-1.0 range
	skillCheckFrequency := max(0.0, min(1.0, c.SkillCheckFrequency))

	return &State{
		history:             diceHist,
		maxRollsStorage:     rollHistorySize,
		skillCheckFrequency: skillCheckFrequency,
		gracePeriodMin:      gracePeriodMin,
		gracePeriodMax:      gracePeriodMax,
		gracePeriodUntil:    randomGracePeriod(gracePeriodMin, gracePeriodMax),
	}
}

// Roll performs a dice roll and stores it in history
func (d *State) Roll() Roll {
	d.mu.Lock()
	defer d.mu.Unlock()

	roll := NewRoll()
	d.history = append(d.history, roll)
	d.lastRoll = roll // Track for skill check consumption

	if len(d.history) > d.maxRollsStorage {
		d.history = d.history[len(d.history)-d.maxRollsStorage:] // Keep last N rolls
	}

	return *roll
}

// GetLastRollValue returns the value of the unconsumed roll, or 0 if no roll is available
func (d *State) GetLastRollValue() int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.lastRoll == nil {
		return 0
	}
	return d.lastRoll.Result
}

// DecideSkillCheckDifficulty determines if a skill check is required and returns the minimum roll needed
// Returns 0 if no skill check is needed
// Returns 10-15 for the difficulty of the skill check
func (d *State) DecideSkillCheckDifficulty(action string, turnCounter int) int {
	// Check if still in grace period
	if turnCounter < d.gracePeriodUntil {
		return 0
	}

	// Check against configured frequency
	if rand.Float64() >= d.skillCheckFrequency {
		return 0
	}

	// Random difficulty between 10-15
	return rand.IntN(6) + 10
}

// HandlePendingSkillCheck validates and processes a pending skill check
// Returns rollContext for LLM prompt, or error if roll is missing
// Consumes the roll to prevent reuse (anti-cheat)
func (d *State) HandlePendingSkillCheck(difficulty int) (rollContext string, err error) {
	if difficulty == 0 {
		return "", nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if user has rolled
	if d.lastRoll == nil {
		return "", fmt.Errorf("%w: need %d or higher - use the %s tool to roll", ErrSkillCheckRequired, difficulty, ToolName)
	}

	// Get roll value and consume it immediately
	rollValue := d.lastRoll.Result
	d.lastRoll = nil // Consume roll to prevent reuse

	// Determine pass/fail
	passed := rollValue >= difficulty

	// return prompt context for the LLM
	if passed {
		return fmt.Sprintf("\nThe player rolled %d (required %d or higher) - they SUCCEEDED!\n", rollValue, difficulty), nil
	}

	return fmt.Sprintf("\nThe player rolled %d (required %d or higher) - they FAILED!\n", rollValue, difficulty), nil
}
