package narrative

import (
	"fmt"
	"strings"
	"sync"
)

// Turn represents a single turn in the adventure
type Turn struct {
	Narrative string        // Generated narrative
	Config    *PromptConfig // Configuration used to generate this turn's narrative
}

// State manages narrative history and skill check state
type State struct {
	mu sync.RWMutex

	turns             []*Turn // Turn history
	summary           string  // Condensed older history
	pendingSkillCheck int     // Pending skill check from previous turn (0 = none, 1-20 = required roll)
}

// NewState creates a new narrative State instance
func NewState() *State {
	return &State{
		turns:             []*Turn{},
		summary:           "",
		pendingSkillCheck: 0,
	}
}

// ActionInput represents input describing what the player does during this turn
type ActionInput struct {
	Action            string `json:"action,omitempty"            jsonschema:"Next action to take. When continuing after a skill check, provide the same or continuation action along with encryptedData."`
	PreviousNarrative string `json:"previousNarrative,omitempty" jsonschema:"Optional brief summary (1-2 sentences) of the previous turn. Where is the player now? What just happened?"`
	EncryptedData     string `json:"encryptedData,omitempty"     jsonschema:"Continuation token for pending skill check. Provide along with the action field when completing a skill check."`
}

// Response is the response object returned to the player after calling the dungeon_master tool
type Response struct {
	Narrative          string `json:"narrative"                    jsonschema:"The generated narrative based on the action and game state. Show this to the user."`
	TurnNumber         int    `json:"turnNumber"                   jsonschema:"Current turn number in the game"`
	ErrorMessage       string `json:"errorMessage,omitempty"       jsonschema:"Error message if any error occurred"`
	SkillCheckRequired int    `json:"skillCheckRequired,omitempty" jsonschema:"If a skill check is required, the minimum roll needed (1-20). When this is > 0, call roll_d20 before your next action."`
}

// ClearPendingSkillCheck resets the pending skill check after validation
func (s *State) ClearPendingSkillCheck() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingSkillCheck = 0
}

// CompleteTurn clears both pending skill check state
// Call this after successfully completing a turn that had a skill check
func (s *State) CompleteTurn() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingSkillCheck = 0
}

// AddTurn records a completed turn in the history
func (s *State) AddTurn(narrative string, config PromptConfig) *Turn {
	s.mu.Lock()
	defer s.mu.Unlock()

	turn := &Turn{
		Narrative: narrative,
		Config:    &config,
	}

	s.turns = append(s.turns, turn)

	// Store skill check requirement for validation in NEXT turn
	if config.NextSkillCheck > 0 {
		s.pendingSkillCheck = config.NextSkillCheck
	}

	return turn
}

// BuildTurnPrompt creates a simple draft narrative prompt based on the current game state.
// This generates an initial narrative without heavy validation - use BuildReviewPrompt for consistency checks.
func (s *State) BuildTurnPrompt(config PromptConfig) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var prompt strings.Builder

	// Add previous context if provided (client-side summary for continuity)
	// Note: This is supplementary - the canonical server history below takes precedence
	if config.InputPreviousNarrative != "" {
		prompt.WriteString("Client's brief summary of what they displayed to the player last turn:")
		prompt.WriteString("\n<clientSummary>\n")
		prompt.WriteString(config.InputPreviousNarrative)
		prompt.WriteString("\n</clientSummary>\n\n")
	} else {
		prompt.WriteString("You are a dungeon master narrating a weird and wild adventure. Generate small story beats based on player actions.\n\n")
	}

	// Include server's canonical game history for authoritative context
	// This is the source of truth - the server's record of all validated events
	if s.summary != "" {
		prompt.WriteString("Canonical summary of recent events (server's authoritative history):")
		prompt.WriteString("\n<summary>\n")
		prompt.WriteString(s.summary)
		prompt.WriteString("\n</summary>\n")
	}

	if len(s.turns) > 0 {
		prompt.WriteString("Recent turns:")
		prompt.WriteString("\n<Turns>\n")
		for _, turn := range s.turns {
			prompt.WriteString(fmt.Sprintf("- %s → %s\n", turn.Config.InputAction, turn.Narrative))
		}
		prompt.WriteString("\n</Turns>\n")
	}

	// Add roll result if there was a pending check (outcome of the skill check from previous turn)
	// This tells the LLM whether the player succeeded or failed their dice roll
	if config.PendingCheck {
		prompt.WriteString(fmt.Sprintf("The player rolled %d", config.PendingCheckResult))
		if config.PassedCheck {
			prompt.WriteString(" - SUCCEEDED!\n")
			prompt.WriteString("Describe how the player overcomes the challenge successfully.\n\n")
		} else {
			prompt.WriteString(" - FAILED!\n")
			prompt.WriteString("Describe how the player fails the challenge and what the consequences are.\n\n")
		}
	}

	// Current action
	prompt.WriteString("Player action: ")
	prompt.WriteString(config.InputAction)
	prompt.WriteString("\n\n")

	// Skill check notice
	if config.NextSkillCheck > 0 {
		prompt.WriteString("This action requires a skill check - describe the challenge dramatically.\n\n")
	}

	// Simple instruction
	prompt.WriteString("Narrate what happens to the player next in a few sentences. Be dramatic. Include sensory details, surroundings, and any NPCs or items involved. Keep it concise and engaging.\n")

	// Anti-echo guidance: explicitly request the LLM avoid repeating prior narrative text.
	prompt.WriteString("Do NOT repeat previous narrative passages verbatim; instead, continue the scene with new details and consequences of the player's action.\n")

	return prompt.String()
}

// BuildReviewPrompt creates a reflection/validation prompt to review draft narrative for consistency.
// Takes the draft narrative and checks it against game history and rules.
func (s *State) BuildReviewPrompt(config PromptConfig, draftNarrative string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var prompt strings.Builder

	prompt.WriteString("Review this draft narrative for consistency:\n\n")
	prompt.WriteString("\n<draftNarrative>\n")
	prompt.WriteString(draftNarrative)
	prompt.WriteString("\n</draftNarrative>\n")

	// Include history context
	if s.summary != "" {
		prompt.WriteString("Summary of the story so far: ")
		prompt.WriteString("\n<summary>\n")
		prompt.WriteString(s.summary)
		prompt.WriteString("\n</summary>\n")
	}

	if len(s.turns) > 0 {
		prompt.WriteString("Recent turns:")
		prompt.WriteString("\n<Turns>\n")
		for _, turn := range s.turns {
			prompt.WriteString(fmt.Sprintf("- %s → %s\n", turn.Config.InputAction, turn.Narrative))
		}
		prompt.WriteString("\n</Turns>\n")
	}

	prompt.WriteString("Validation rules:\n")
	prompt.WriteString("- The narrative must follow the established story.\n")
	prompt.WriteString("- The player must only use items they have, or have interacted with previously.\n")
	prompt.WriteString("- The player can only interact with NPCs that have been introduced.\n")
	prompt.WriteString("- If a skill check was required, the outcome must reflect whether the player succeeded or failed based on the provided roll result.\n")
	prompt.WriteString("- Keep the narrative concise and focused on the player's action and its immediate consequences.\n\n")
	prompt.WriteString("Your task: Validate the draft against the rules above. If the draft has errors, fix them. If the draft is correct, enhance it with one additional vivid sensory detail or consequence.\n\n")
	prompt.WriteString("IMPORTANT: Do NOT return the draft verbatim - always improve or enhance it slightly.\n\n")
	prompt.WriteString("Output format: Return only the final narrative text - no prefixes, no explanations, no meta-commentary.\n\n")
	prompt.WriteString("Example input and output, with the location adjusted to fit the summary:\n")
	prompt.WriteString("Summary:\n")
	prompt.WriteString("The player is in a chamber, holding a golden key.\n")
	prompt.WriteString("Input draft:\n")
	prompt.WriteString("You look around the cave. The walls are smooth.\n")
	prompt.WriteString("Enhanced output:\n")
	prompt.WriteString("You look around the chamber, golden key in hand. The walls are smooth to the touch.\n")

	return prompt.String()
}

// ShouldSummarize checks if the turn history exceeds the threshold
func (s *State) ShouldSummarize(threshold int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.turns) > threshold
}

// BuildSummaryPrompt creates a prompt for summarizing turn history
func (s *State) BuildSummaryPrompt() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Summarize first half, keep second half
	splitPoint := len(s.turns) / 2
	toSummarize := s.turns[:splitPoint]

	var entries []string
	for _, turn := range toSummarize {
		entries = append(entries, fmt.Sprintf("Action: %s\nNarrative: %s", turn.Config.InputAction, turn.Narrative))
	}

	prompt := "Summarize these adventure events in a few sentences, focus on important events such as collecting inventory, entering new locations, interacting with NPCs or battling enemies:\n\n"
	prompt += strings.Join(entries, "\n")

	return prompt
}

// RecordSummary updates the summary and trims the turn history
func (s *State) RecordSummary(summaryText string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update summary (append to existing if present)
	if s.summary != "" {
		s.summary = s.summary + " " + summaryText
	} else {
		s.summary = summaryText
	}

	// Keep only second half of turns
	splitPoint := len(s.turns) / 2
	s.turns = s.turns[splitPoint:]
}

// GetTurns returns the turn history
func (s *State) GetTurns() []*Turn {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.turns
}

// GetEntries returns the turn history as formatted strings (for backwards compatibility)
func (s *State) GetEntries() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]string, len(s.turns))
	for i, turn := range s.turns {
		entries[i] = fmt.Sprintf("Action: %s\nNarrative: %s", turn.Config.InputAction, turn.Narrative)
	}
	return entries
}

// GetSummary returns the current narrative summary
func (s *State) GetSummary() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.summary
}

// GetPendingSkillCheck returns the current pending skill check difficulty (0 = none)
func (s *State) GetPendingSkillCheck() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pendingSkillCheck
}
