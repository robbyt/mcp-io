package mcpio

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/capabilities"
)

// mcpContextKey is the context key for storing requestContext instances.
type mcpContextKey struct{}

// requestContext holds all MCP request metadata.
// This allows single-allocation context injection - we store this once instead of
// storing Session and request metadata separately.
type requestContext struct {
	session    *capabilities.Session // MCP session wrapper
	extra      *mcp.RequestExtra     // OAuth tokens, HTTP headers
	identifier string                // Tool name, prompt name, or resource URI
}

// withMCPContext creates a requestContext and injects it into the context.
// This is the single allocation point that replaces the previous dual-allocation pattern.
func withMCPContext(
	ctx context.Context,
	identifier string,
	mcpSession *mcp.ServerSession,
	extra *mcp.RequestExtra,
) context.Context {
	reqCtx := &requestContext{
		session:    capabilities.NewSession(mcpSession),
		extra:      extra,
		identifier: identifier,
	}
	return context.WithValue(ctx, mcpContextKey{}, reqCtx)
}

// getMCPContext extracts the requestContext from the context.
// Returns nil if no request context is available.
func getMCPContext(ctx context.Context) *requestContext {
	if reqCtx, ok := ctx.Value(mcpContextKey{}).(*requestContext); ok {
		return reqCtx
	}
	return nil
}

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
	reqCtx := getMCPContext(ctx)
	if reqCtx == nil {
		// Fallback for tests that inject sessions using capabilities.WithSession directly
		return capabilities.GetSession(ctx)
	}
	return reqCtx.session
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

// GetIdentifier returns the identifier for the current request.
// For tools, this is the tool name. For prompts, the prompt name. For resources, the URI.
// Returns empty string if no request context is available.
//
// Example:
//
//	toolName := mcpio.GetIdentifier(ctx)
//	logger.Info("Tool invoked", "name", toolName)
func GetIdentifier(ctx context.Context) string {
	reqCtx := getMCPContext(ctx)
	if reqCtx == nil {
		return ""
	}
	return reqCtx.identifier
}

// GetTokenInfo returns OAuth token information from the request if available.
// Returns nil if no token was provided or request context is unavailable.
//
// Example:
//
//	tokenInfo := mcpio.GetTokenInfo(ctx)
//	if tokenInfo != nil {
//	    userID := tokenInfo.Subject
//	    scopes := tokenInfo.Scopes
//	}
func GetTokenInfo(ctx context.Context) *auth.TokenInfo {
	reqCtx := getMCPContext(ctx)
	if reqCtx == nil || reqCtx.extra == nil {
		return nil
	}
	return reqCtx.extra.TokenInfo
}

// GetRequestHeader returns a specific HTTP header value from the request.
// Returns empty string if the header is not present or request context is unavailable.
//
// Example:
//
//	userAgent := mcpio.GetRequestHeader(ctx, "User-Agent")
//	if userAgent != "" {
//	    logger.Info("Client info", "userAgent", userAgent)
//	}
func GetRequestHeader(ctx context.Context, key string) string {
	reqCtx := getMCPContext(ctx)
	if reqCtx == nil || reqCtx.extra == nil || reqCtx.extra.Header == nil {
		return ""
	}
	return reqCtx.extra.Header.Get(key)
}

// GetRequestHeaders returns all HTTP headers from the request.
// Returns nil if no headers are available or request context is unavailable.
//
// Example:
//
//	headers := mcpio.GetRequestHeaders(ctx)
//	if headers != nil {
//	    for key, values := range headers {
//	        logger.Debug("Header", "key", key, "values", values)
//	    }
//	}
func GetRequestHeaders(ctx context.Context) http.Header {
	reqCtx := getMCPContext(ctx)
	if reqCtx == nil || reqCtx.extra == nil {
		return nil
	}
	return reqCtx.extra.Header
}
