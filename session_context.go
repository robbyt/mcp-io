package mcpio

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/capabilities"
)

// sessionContextKey is the private key for storing Session in context
type sessionContextKey struct{}

// GetSession extracts the Session from the context.
// Returns nil if no session is available (e.g., in tests or non-session contexts).
//
// Example:
//
//	func myTool(ctx context.Context, input Input) (Output, error) {
//	    session := mcpio.GetSession(ctx)
//	    if session != nil {
//	        session.Logger().Info("Processing request", "sessionID", session.SessionID())
//	    }
//	    return output, nil
//	}
func GetSession(ctx context.Context) *capabilities.Session {
	val := ctx.Value(sessionContextKey{})
	if val == nil {
		return nil
	}
	if session, ok := val.(*capabilities.Session); ok {
		return session
	}
	return nil
}

// injectSession adds a session to the context.
// This is called internally by the handler before invoking user functions.
func injectSession(ctx context.Context, session *capabilities.Session) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, session)
}

// injectMCPSession creates a Session from an MCP ServerSession and injects it into the context.
// This helper combines Session creation and context injection in one call.
func injectMCPSession(ctx context.Context, mcpSession *mcp.ServerSession) context.Context {
	session := capabilities.NewSessionCapability(mcpSession)
	return injectSession(ctx, session)
}

// CreateMessage asks the client's LLM to generate a response to the provided messages.
// This enables servers to use the client's LLM for analysis, suggestions, or processing.
// Returns an error if the session is not available or the client doesn't support sampling.
//
// Example:
//
//	result, err := mcpio.CreateMessage(ctx, []*capabilities.Message{{
//	    Role:    "user",
//	    Content: "Analyze this code",
//	}}, 2000)
//	if err != nil {
//	    return nil, err
//	}
//	analysis := result.Content.Text
func CreateMessage(ctx context.Context, messages []*capabilities.Message, maxTokens int) (*capabilities.MessageResult, error) {
	session := GetSession(ctx)
	if session == nil {
		return nil, ErrNoSession
	}
	if !session.SupportsSampling() {
		return nil, ErrSamplingNotSupported
	}
	return session.CreateMessage(ctx, messages, maxTokens)
}

// CreateMessageRaw provides direct access to the MCP CreateMessage API with full control over parameters.
// Use this when you need to specify model preferences, temperature, system prompts, or other advanced options.
// Returns an error if the session is not available or doesn't support sampling.
//
// Example:
//
//	params := &mcp.CreateMessageParams{
//	    Messages: []*mcp.SamplingMessage{{
//	        Role: "user",
//	        Content: &mcp.TextContent{Text: "Analyze this code"},
//	    }},
//	    MaxTokens: 1000,
//	    ModelPreferences: &mcp.ModelPreferences{
//	        Hints: []*mcp.ModelHint{{Name: "llama3.1"}},
//	    },
//	}
//	result, err := mcpio.CreateMessageRaw(ctx, params)
func CreateMessageRaw(ctx context.Context, params *mcp.CreateMessageParams) (*mcp.CreateMessageResult, error) {
	session := GetSession(ctx)
	if session == nil {
		return nil, ErrNoSession
	}
	if !session.SupportsSampling() {
		return nil, ErrSamplingNotSupported
	}
	return session.CreateMessageRaw(ctx, params)
}

// ListRoots queries the client's workspace roots (directories, files, etc.).
// Returns an error if the session is not available.
//
// Example:
//
//	roots, err := mcpio.ListRoots(ctx)
//	if err != nil {
//	    return nil, err
//	}
func ListRoots(ctx context.Context) ([]*capabilities.Root, error) {
	session := GetSession(ctx)
	if session == nil {
		return nil, ErrNoSession
	}
	return session.ListRoots(ctx)
}

// NotifyProgress sends a progress update for long-running operations.
// Progress should be between 0.0 and total, where total represents completion.
// Returns an error if the session is not available.
//
// Example:
//
//	mcpio.NotifyProgress(ctx, 0.0, 1.0)  // Starting
//	// ... do work ...
//	mcpio.NotifyProgress(ctx, 0.5, 1.0)  // Halfway
//	// ... do more work ...
//	mcpio.NotifyProgress(ctx, 1.0, 1.0)  // Complete
func NotifyProgress(ctx context.Context, progress, total float64) error {
	session := GetSession(ctx)
	if session == nil {
		return ErrNoSession
	}
	return session.NotifyProgress(ctx, progress, total)
}

// LogInfo sends an info-level log message to the client.
// Returns an error if the session is not available.
//
// Example:
//
//	mcpio.LogInfo(ctx, "Processing started", map[string]any{"file": filename})
func LogInfo(ctx context.Context, message string, data map[string]any) error {
	session := GetSession(ctx)
	if session == nil {
		return ErrNoSession
	}
	return session.Log(ctx, capabilities.LogLevelInfo, message, data)
}

// LogWarn sends a warning-level log message to the client.
// Returns an error if the session is not available.
func LogWarn(ctx context.Context, message string, data map[string]any) error {
	session := GetSession(ctx)
	if session == nil {
		return ErrNoSession
	}
	return session.Log(ctx, capabilities.LogLevelWarning, message, data)
}

// LogError sends an error-level log message to the client.
// Returns an error if the session is not available.
func LogError(ctx context.Context, message string, data map[string]any) error {
	session := GetSession(ctx)
	if session == nil {
		return ErrNoSession
	}
	return session.Log(ctx, capabilities.LogLevelError, message, data)
}

// LogDebug sends a debug-level log message to the client.
// Returns an error if the session is not available.
func LogDebug(ctx context.Context, message string, data map[string]any) error {
	session := GetSession(ctx)
	if session == nil {
		return ErrNoSession
	}
	return session.Log(ctx, capabilities.LogLevelDebug, message, data)
}

// GetLogger returns a slog.Logger that sends logs to the client.
// Returns nil if the session is not available.
//
// Example:
//
//	logger := mcpio.GetLogger(ctx)
//	if logger != nil {
//	    logger.Info("Processing request", "file", filename, "size", fileSize)
//	}
func GetLogger(ctx context.Context) *slog.Logger {
	session := GetSession(ctx)
	if session == nil {
		return nil
	}
	return session.Logger()
}

// GetSessionID returns the session ID, or empty string if no session is available.
func GetSessionID(ctx context.Context) string {
	session := GetSession(ctx)
	if session == nil {
		return ""
	}
	return session.SessionID()
}

// InjectSessionForTesting injects a Session into the context for testing purposes.
// This allows tests to provide sessions without going through the full MCP handler.
//
// Example:
//
//	// In tests, inject a session:
//	ctx := mcpio.InjectSessionForTesting(context.Background(), session)
//	result, err := myTool(ctx, input)
func InjectSessionForTesting(ctx context.Context, session *capabilities.Session) context.Context {
	return injectSession(ctx, session)
}
