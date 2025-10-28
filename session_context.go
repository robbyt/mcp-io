package mcpio

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/capabilities"
)

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
var GetSession = capabilities.GetSession

// withMCPSession creates a Session from an MCP ServerSession and adds it to the context.
// This helper combines Session creation and context injection in one call.
func withMCPSession(ctx context.Context, mcpSession *mcp.ServerSession) context.Context {
	session := capabilities.NewSession(mcpSession)
	return capabilities.WithSession(ctx, session)
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

// GetSessionID returns the session ID for logging, debugging, and request correlation.
// Returns empty string if no session is available.
//
// The session ID is set by the MCP client and uniquely identifies the connection.
// Use this for audit trails, correlation IDs, or debugging client-specific behavior.
//
// Example:
//
//	sessionID := mcpio.GetSessionID(ctx)
//	if sessionID != "" {
//	    logger.Info("Processing request", "sessionID", sessionID, "operation", "analyze")
//	}
func GetSessionID(ctx context.Context) string {
	session := GetSession(ctx)
	if session == nil {
		return ""
	}
	return session.SessionID()
}
