package dice

import "math/rand/v2"

// Config holds configuration for dice state initialization
type Config struct {
	GracePeriodMin      int     // Minimum turns before skill checks can occur
	GracePeriodMax      int     // Maximum turns before skill checks can occur
	SkillCheckFrequency float64 // Probability of skill check per action (0.0-1.0)
}

// randomGracePeriod returns a random turn number between min and max (inclusive)
func randomGracePeriod(min, max int) int {
	return rand.IntN(max-min+1) + min
}
