package capabilities

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SupportsElicitation returns true if the client supports elicitation (user input requests).
func (s *Session) SupportsElicitation() bool {
	return s.session.InitializeParams().Capabilities.Elicitation != nil
}

// Elicit sends an elicitation request to the client asking for user input.
// Returns an ElicitResult containing the user's action and optionally submitted data.
func (s *Session) Elicit(ctx context.Context, message string, requestedSchema any) (*mcp.ElicitResult, error) {
	params := &mcp.ElicitParams{
		Message:         message,
		RequestedSchema: requestedSchema,
	}
	return s.session.Elicit(ctx, params)
}
