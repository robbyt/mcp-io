package capabilities

import (
	"context"
	"log/slog"
	"maps"

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

	// ListRoots queries the client's workspace roots (directories, files, etc.).
	// Servers can use this to discover what the client has access to.
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

	// ClientCapabilities returns the capabilities the client declared during initialization.
	ClientCapabilities() *ClientCapabilities

	// SupportsElicitation returns true if the client supports elicitation (user input requests).
	SupportsElicitation() bool

	// SupportsSampling returns true if the client supports LLM sampling (CreateMessage).
	SupportsSampling() bool

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

func (s *sessionCapability) SupportsElicitation() bool {
	return s.session.InitializeParams().Capabilities.Elicitation != nil
}

func (s *sessionCapability) Elicit(ctx context.Context, message string, requestedSchema any) (*mcp.ElicitResult, error) {
	params := &mcp.ElicitParams{
		Message:         message,
		RequestedSchema: requestedSchema,
	}
	return s.session.Elicit(ctx, params)
}

func (s *sessionCapability) SupportsSampling() bool {
	return s.session.InitializeParams().Capabilities.Sampling != nil
}

func (s *sessionCapability) CreateMessage(ctx context.Context, messages []*Message, maxTokens int) (*MessageResult, error) {
	// Convert our Message type to mcp.SamplingMessage
	mcpMessages := make([]*mcp.SamplingMessage, len(messages))
	for i, msg := range messages {
		mcpMessages[i] = &mcp.SamplingMessage{
			Role:    mcp.Role(msg.Role),
			Content: &mcp.TextContent{Text: msg.Content},
		}
	}

	params := &mcp.CreateMessageParams{
		Messages:  mcpMessages,
		MaxTokens: int64(maxTokens),
	}

	result, err := s.session.CreateMessage(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert result back to our types
	mcpContent, ok := result.Content.(*mcp.TextContent)
	if !ok {
		return &MessageResult{
			Role:    string(result.Role),
			Content: TextContent{Text: ""},
		}, nil
	}

	return &MessageResult{
		Role:    string(result.Role),
		Content: TextContent{Text: mcpContent.Text},
	}, nil
}

func (s *sessionCapability) CreateMessageRaw(ctx context.Context, params *mcp.CreateMessageParams) (*mcp.CreateMessageResult, error) {
	return s.session.CreateMessage(ctx, params)
}

func (s *sessionCapability) ListRoots(ctx context.Context) ([]*Root, error) {
	params := &mcp.ListRootsParams{}
	result, err := s.session.ListRoots(ctx, params)
	if err != nil {
		return nil, err
	}

	roots := make([]*Root, len(result.Roots))
	for i, r := range result.Roots {
		roots[i] = &Root{
			URI:  r.URI,
			Name: r.Name,
		}
	}
	return roots, nil
}

func (s *sessionCapability) Log(ctx context.Context, level LogLevel, message string, data map[string]any) error {
	logData := make(map[string]any)
	if data != nil {
		maps.Copy(logData, data)
	}
	logData["message"] = message

	params := &mcp.LoggingMessageParams{
		Level: mcp.LoggingLevel(level),
		Data:  logData,
	}
	return s.session.Log(ctx, params)
}

func (s *sessionCapability) Logger() *slog.Logger {
	return slog.New(mcp.NewLoggingHandler(s.session, nil))
}

func (s *sessionCapability) NotifyProgress(ctx context.Context, progress, total float64) error {
	params := &mcp.ProgressNotificationParams{
		Progress: progress,
		Total:    total,
	}
	return s.session.NotifyProgress(ctx, params)
}

func (s *sessionCapability) SessionID() string {
	return s.session.ID()
}

func (s *sessionCapability) ClientCapabilities() *ClientCapabilities {
	initParams := s.session.InitializeParams()
	caps := &ClientCapabilities{}

	if initParams.Capabilities.Elicitation != nil {
		caps.Elicitation = &ElicitationCapabilities{}
	}
	if initParams.Capabilities.Sampling != nil {
		caps.Sampling = &SamplingCapabilities{}
	}
	// Roots capability - check if ListChanged is supported
	if initParams.Capabilities.Roots.ListChanged {
		caps.Roots = &RootsCapabilities{
			ListChanged: true,
		}
	}

	return caps
}

func (s *sessionCapability) Wait() error {
	return s.session.Wait()
}

func (s *sessionCapability) Close() error {
	return s.session.Close()
}
