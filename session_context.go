package mcpio

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/capabilities"
)

// mcpContextKey is the context key for storing RequestContext instances.
type mcpContextKey struct{}

// RequestContext holds all MCP request metadata and implements the ToolContext interface.
// This struct is created once per request and passed directly to tool functions,
// eliminating the need for context storage and retrieval.
//
// Exported to enable test mocking. Tests construct RequestContext instances
// with mock sessions when testing elicitation, logging, and other
// session-dependent functionality.
type RequestContext struct {
	// Session provides access to MCP session capabilities (sampling, elicitation, logging, etc).
	// Never nil - always contains a valid session instance.
	Session *capabilities.Session

	// Identifier is the name or URI for the current request:
	//   - Tool requests: tool name
	//   - Prompt requests: prompt name
	//   - Resource requests: resource URI
	Identifier string

	// TokenInfo contains OAuth token information when the request includes authentication.
	// Nil if no token is present in the request.
	TokenInfo *auth.TokenInfo

	// Headers contains HTTP headers when the request was made via HTTP transport.
	// Never nil - always contains a valid (possibly empty) http.Header map.
	Headers http.Header
}

// newRequestContext creates a new RequestContext instance by extracting and formatting
// all MCP request metadata once. This "format once, use everywhere" pattern eliminates
// repeated nil checking and data extraction on every field access.
func newRequestContext(
	identifier string,
	mcpSession *mcp.ServerSession,
	extra *mcp.RequestExtra,
) *RequestContext {
	var tokenInfo *auth.TokenInfo
	var headers http.Header

	// Extract values once from mcp.RequestExtra
	if extra != nil {
		tokenInfo = extra.TokenInfo
		headers = extra.Header
	}

	// Ensure headers is never nil
	if headers == nil {
		headers = http.Header{}
	}

	return &RequestContext{
		Session:    capabilities.NewSession(mcpSession),
		Identifier: identifier,
		TokenInfo:  tokenInfo,
		Headers:    headers,
	}
}

// ToolContext interface implementation - RequestContext provides direct access
// to all request metadata through simple getter methods.

// GetSession returns the MCP session for accessing session capabilities like
// logging, elicitation, and sampling.
func (r *RequestContext) GetSession() *capabilities.Session {
	return r.Session
}

// GetIdentifier returns the identifier for the current request.
// For tools, this is the tool name. For prompts, the prompt name. For resources, the URI.
func (r *RequestContext) GetIdentifier() string {
	return r.Identifier
}

// GetTokenInfo returns OAuth token information from the request if available.
// Returns nil if no token was provided.
func (r *RequestContext) GetTokenInfo() *auth.TokenInfo {
	return r.TokenInfo
}

// GetHeaders returns all HTTP headers from the request.
// Never returns nil - returns empty http.Header{} if no headers present.
func (r *RequestContext) GetHeaders() http.Header {
	return r.Headers
}
