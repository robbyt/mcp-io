package dice

import (
	"errors"
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

// State represents a dice manager with turn-based roll tracking
type State struct {
	mu sync.RWMutex

	rollsByTurn map[int]*Roll // Rolls indexed by turn number (idempotent per turn)

	// Skill check configuration
	skillCheckFrequency float64 // Probability of skill check per action (0.0-1.0)
	gracePeriodMin      int     // Minimum turns before skill checks can occur
	gracePeriodMax      int     // Maximum turns before skill checks can occur
	gracePeriodUntil    int     // Randomly chosen turn when skill checks activate
}

// NewState creates a new Dice instance with skill check configuration
func NewState(c *Config) *State {
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
		rollsByTurn:         make(map[int]*Roll),
		skillCheckFrequency: skillCheckFrequency,
		gracePeriodMin:      gracePeriodMin,
		gracePeriodMax:      gracePeriodMax,
		gracePeriodUntil:    randomGracePeriod(gracePeriodMin, gracePeriodMax),
	}
}

// Roll performs a dice roll
func (d *State) Roll() Roll {
	d.mu.Lock()
	defer d.mu.Unlock()

	roll := NewRoll()
	return *roll
}

// RollTurn performs a dice roll for a specific turn number
// If a roll already exists for this turn, returns it (idempotent)
func (d *State) RollTurn(turn int) Roll {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Return existing roll for this turn if available (idempotent behavior)
	if roll, exists := d.rollsByTurn[turn]; exists {
		return *roll
	}

	// Generate new roll for this turn
	roll := NewRoll()
	d.rollsByTurn[turn] = roll

	return *roll
}

// DecideNextSkillCheckDifficulty determines if a skill check is required and returns the minimum roll needed
// Returns 0 if no skill check is needed
// Returns scaling difficulty (5-18) that increases with turn progression
func (d *State) DecideNextSkillCheckDifficulty(turnCounter int) int {
	// Check if still in grace period
	if turnCounter < d.gracePeriodUntil {
		return 0
	}

	// Check against configured frequency
	if rand.Float64() >= d.skillCheckFrequency {
		return 0
	}

	// Calculate scaling difficulty based on turns after grace period
	const (
		baseDifficulty = 5  // Starting difficulty after grace period
		maxDifficulty  = 18 // Cap difficulty (leave room for criticals)
		turnsPerLevel  = 2  // +1 difficulty every 2 turns
	)

	turnsAfterGrace := turnCounter - d.gracePeriodUntil
	difficulty := baseDifficulty + (turnsAfterGrace / turnsPerLevel)

	return min(difficulty, maxDifficulty)
}
