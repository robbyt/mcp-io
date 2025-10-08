package narrative

type PromptConfig struct {
	TurnNumber             int    // Current turn number in the game
	InputAction            string // Player's action for this turn
	InputPreviousNarrative string // Optional client-provided context from previous turn (supplementary to server's canonical history)

	PendingCheck bool // Whether a skill check was pending for this turn
	// If PendingCheck is true, these fields must be set:
	PendingCheckResult int  // The result of the pending skill check roll (1-20)
	PassedCheck        bool // Whether the pending skill check was passed
	NextSkillCheck     int  // The difficulty of the next skill check (0 = none)
}
