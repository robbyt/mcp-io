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
	Action        string `json:"action"                  jsonschema:"Input from the player describing their desired action for this turn"`
	EncryptedData string `json:"encryptedData,omitempty" jsonschema:"Optional encrypted data from previous tool responses."`
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

	prompt.WriteString("You are a dungeon master. You narrate adventures based on player actions, and generate small story beats to create an weird and wild adventure.\n")
	prompt.WriteString("If the player's action conflicts or contradicts with the previous narrative, they should be stylistically told they can't do that.\n")
	prompt.WriteString("For example, if the tries to open a locked door, but they didn't pick up the key off the ground, they should be told they can't do that because they don't have a key.\n")
	prompt.WriteString("Or if the player tries to attack a friendly NPC, they should be told they can't do that because the NPC is friendly.\n")
	prompt.WriteString("If the player attempts to communicate with an NPC that hasn't yet been introduced, or isn't near, explain their error and do NOT allow them to communicate.\n\n")

	// Add skill check requirement if needed
	if nextSkillCheck > 0 {
		prompt.WriteString("IMPORTANT: This action is challenging and requires a skill check.\n")
		prompt.WriteString("Describe what they're attempting in dramatic terms. The response will automatically include the skill check requirement.\n")
		prompt.WriteString("Do NOT tell them to roll dice - the tool response handles that. Focus on the narrative drama of their attempt.\n\n")
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
	prompt.WriteString("The player performs the following action:\n")
	prompt.WriteString("<action>")
	prompt.WriteString(action)
	prompt.WriteString("</action>")
	prompt.WriteString("Does this action conflict with previous narrative?")
	prompt.WriteString("Is the player attempting to communicate with characters they have not encountered? Does this action use items that have not yet been located? If so, explain why they can't do that.\n\n")
	prompt.WriteString("Narrate what happens next in 2 sentences. Be dramatic!\n")

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
