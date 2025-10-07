package narrative

import (
	"fmt"
	"strings"
	"sync"
)

// Turn represents a single turn in the adventure
type Turn struct {
	Action      string // Player's action
	Narrative   string // Generated narrative
	SkillCheck  int    // Skill check set for NEXT turn (0 = none, 1-20 = required)
	RollContext string // Roll result context (if any)
}

// State manages narrative history and skill check state
type State struct {
	mu sync.RWMutex

	turns             []*Turn // Turn history
	summary           string  // Condensed older history
	pendingSkillCheck int     // Pending skill check from previous turn (0 = none, 1-20 = required roll)
	lastEncryptedRoll string  // Encrypted roll from last roll (empty = no roll yet, prevents re-rolling)
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
	Action        string `json:"action,omitempty"        jsonschema:"New action to take. Omit when providing encryptedData to continue a pending action."`
	EncryptedData string `json:"encryptedData,omitempty" jsonschema:"Continuation token for pending skill check. When provided, completes the previous action without needing to restate it."`
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

// CompleteTurn clears both pending skill check and encrypted roll state
// Call this after successfully completing a turn that had a skill check
func (s *State) CompleteTurn() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingSkillCheck = 0
	s.lastEncryptedRoll = ""
}

// AddTurn records a completed turn in the history
func (s *State) AddTurn(action, narrative string, skillCheck int, rollContext string) *Turn {
	s.mu.Lock()
	defer s.mu.Unlock()

	turn := &Turn{
		Action:      action,
		Narrative:   narrative,
		SkillCheck:  skillCheck,
		RollContext: rollContext,
	}

	s.turns = append(s.turns, turn)

	// Store skill check requirement for validation in NEXT turn
	if skillCheck > 0 {
		s.pendingSkillCheck = skillCheck
	}

	return turn
}

// BuildTurnPrompt creates a context-aware prompt for the LLM including game history
func (s *State) BuildTurnPrompt(action string, nextSkillCheck int, rollContext string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var prompt strings.Builder

	// Add roll context if present (from previous turn's skill check)
	if rollContext != "" {
		prompt.WriteString(rollContext)
		prompt.WriteString("\n")
	}

	prompt.WriteString("You are a dungeon master narrating a weird and wild adventure. Generate small story beats based on player actions.\n\n")

	prompt.WriteString("Validate consistency:\n")
	prompt.WriteString("- If the player's action conflicts with previous narrative, tell them they can't do that.\n")
	prompt.WriteString("- If they try to use items they haven't found, explain they don't have them.\n")
	prompt.WriteString("- If they try to interact with NPCs not yet introduced, explain the NPC isn't present.\n\n")

	// Add skill check requirement if needed
	if nextSkillCheck > 0 {
		prompt.WriteString("This action is challenging and requires a skill check.\n")
		prompt.WriteString("Describe what they're attempting in dramatic terms. The response will include the skill check requirement.\n")
		prompt.WriteString("Focus on the narrative drama - don't mention dice rolling (the tool handles that).\n\n")
	}

	// Include older summary if available
	if s.summary != "" {
		prompt.WriteString("Previous events:\n")
		prompt.WriteString("<summary>\n")
		prompt.WriteString(s.summary)
		prompt.WriteString("</summary>\n")
	}

	// Include recent turn history
	if len(s.turns) > 0 {
		prompt.WriteString("Recent events:\n")
		for _, turn := range s.turns {
			prompt.WriteString("<event>\n")
			prompt.WriteString(fmt.Sprintf("Player Action: %s\nNarrative Response: %s", turn.Action, turn.Narrative))
			prompt.WriteString("</event>\n")
			prompt.WriteString("\n")
		}
		prompt.WriteString("\n")
	}

	// Add current action
	prompt.WriteString("Current action:\n")
	prompt.WriteString("<action>")
	prompt.WriteString(action)
	prompt.WriteString("</action>\n\n")
	prompt.WriteString("Check if this action conflicts with the established narrative, then narrate what happens next in 2 sentences. Be dramatic!\n")

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
		entries = append(entries, fmt.Sprintf("Action: %s\nNarrative: %s", turn.Action, turn.Narrative))
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
		entries[i] = fmt.Sprintf("Action: %s\nNarrative: %s", turn.Action, turn.Narrative)
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

// GetLastEncryptedRoll returns the encrypted roll from the last roll (empty = no roll yet)
func (s *State) GetLastEncryptedRoll() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastEncryptedRoll
}

// SetLastEncryptedRoll stores the encrypted roll (prevents re-rolling)
func (s *State) SetLastEncryptedRoll(encrypted string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastEncryptedRoll = encrypted
}

// ClearLastEncryptedRoll resets the encrypted roll after consumption
func (s *State) ClearLastEncryptedRoll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastEncryptedRoll = ""
}
