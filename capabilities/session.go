package capabilities

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SessionCapability provides access to all MCP session features.
// It is automatically injected into the context for all tool, prompt, and resource handlers.
// Access it using GetSession(ctx).
//
// Example:
//
//	func myTool(ctx context.Context, input Input) (Output, error) {
//	    session := mcpio.GetSession(ctx)
//	    if session != nil && session.SupportsSampling() {
//	        result, _ := session.CreateMessage(ctx, messages, 1000)
//	    }
//	    return output, nil
//	}
type SessionCapability interface {
	// Elicit sends an elicitation request to the client asking for user input.
	// Returns an ElicitResult containing the user's action and optionally submitted data.
	Elicit(ctx context.Context, message string, requestedSchema any) (*mcp.ElicitResult, error)

	// CreateMessage asks the client's LLM to generate a response to the provided messages.
	// This enables servers to use the client's LLM for analysis, suggestions, or processing.
	// Returns nil error and empty result if the client doesn't support sampling.
	CreateMessage(ctx context.Context, messages []*Message, maxTokens int) (*MessageResult, error)

	// CreateMessageRaw provides direct access to the MCP CreateMessage API with full control over parameters.
	// Use this when you need to specify model preferences, temperature, system prompts, or other advanced options.
	CreateMessageRaw(ctx context.Context, params *mcp.CreateMessageParams) (*mcp.CreateMessageResult, error)

	// ListRoots retrieves the filesystem roots exposed by the client.
	// Roots define the boundaries of where servers can operate within the filesystem.
	ListRoots(ctx context.Context) ([]*Root, error)

	// Log sends a structured log message to the client.
	// The message will only be sent if it meets the client's minimum log level.
	Log(ctx context.Context, level LogLevel, message string, data map[string]any) error

	// Logger returns a slog.Logger that sends logs to the client.
	// This provides standard Go logging integration with MCP client logging.
	Logger() *slog.Logger

	// NotifyProgress sends a progress update for long-running operations.
	// Progress should be between 0.0 and total, where total represents completion.
	NotifyProgress(ctx context.Context, progress, total float64) error

	// SessionID returns the unique identifier for this session.
	SessionID() string

	// SupportsElicitation returns true if the client supports elicitation (user input requests).
	SupportsElicitation() bool

	// SupportsSampling returns true if the client supports LLM sampling (CreateMessage).
	SupportsSampling() bool

	// SupportsRoots returns true if the client supports filesystem roots.
	SupportsRoots() bool

	// Wait blocks until the client disconnects or the session is closed.
	Wait() error

	// Close closes this session and the underlying connection.
	Close() error
}

// sessionCapability implements SessionCapability by wrapping mcp.ServerSession
type sessionCapability struct {
	session *mcp.ServerSession
}

// NewSessionCapability creates a SessionCapability from an mcp.ServerSession
func NewSessionCapability(session *mcp.ServerSession) SessionCapability {
	return &sessionCapability{session: session}
}

// Logger returns a slog.Logger that sends logs to the client.
// This provides standard Go logging integration with MCP client logging.
func (s *sessionCapability) Logger() *slog.Logger {
	return slog.New(mcp.NewLoggingHandler(s.session, nil))
}

// SessionID returns the unique identifier for this session.
func (s *sessionCapability) SessionID() string {
	return s.session.ID()
}

// Wait blocks until the client disconnects or the session is closed.
func (s *sessionCapability) Wait() error {
	return s.session.Wait()
}

// Close closes this session and the underlying connection.
func (s *sessionCapability) Close() error {
	return s.session.Close()
}
