package mcpio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Resource registration function types
type (
	promptRegisterFunc           func(*mcp.Server) error
	resourceRegisterFunc         func(*mcp.Server) error
	resourceTemplateRegisterFunc func(*mcp.Server) error
)

// handlerConfig holds the configuration built by options
type handlerConfig struct {
	name              string
	version           string
	tools             []toolRegisterFunc
	prompts           []promptRegisterFunc
	resources         []resourceRegisterFunc
	resourceTemplates []resourceTemplateRegisterFunc
	server            *mcp.Server // The MCP-SDK server instance
}

// Handler is the main MCP handler struct
type Handler struct {
	server      *mcp.Server
	httpHandler http.Handler
}

// GetServer returns the underlying MCP server for advanced usage
func (h *Handler) GetServer() *mcp.Server {
	return h.server
}

// NewHandler creates a new MCP handler that supports any combination of MCP resources.
// This is the unified constructor that can handle tools, prompts, resources, and resource templates.
func NewHandler(opts ...Option) (*Handler, error) {
	cfg := &handlerConfig{
		name:              "mcp-server",
		version:           "1.0.0",
		tools:             make([]toolRegisterFunc, 0),
		prompts:           make([]promptRegisterFunc, 0),
		resources:         make([]resourceRegisterFunc, 0),
		resourceTemplates: make([]resourceTemplateRegisterFunc, 0),
	}

	// Apply all options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Create a new MCP server if not provided
	if cfg.server == nil {
		impl := &mcp.Implementation{
			Name:    cfg.name,
			Version: cfg.version,
		}
		cfg.server = mcp.NewServer(impl, nil)
	}

	// Register all resources
	errz := make([]error, 0)
	for _, registerFunc := range cfg.tools {
		if err := registerFunc(cfg.server); err != nil {
			errz = append(errz, fmt.Errorf("failed to register tool: %w", err))
		}
	}
	for _, registerFunc := range cfg.prompts {
		if err := registerFunc(cfg.server); err != nil {
			errz = append(errz, fmt.Errorf("failed to register prompt: %w", err))
		}
	}
	for _, registerFunc := range cfg.resources {
		if err := registerFunc(cfg.server); err != nil {
			errz = append(errz, fmt.Errorf("failed to register resource: %w", err))
		}
	}
	for _, registerFunc := range cfg.resourceTemplates {
		if err := registerFunc(cfg.server); err != nil {
			errz = append(errz, fmt.Errorf("failed to register resource template: %w", err))
		}
	}
	if len(errz) > 0 {
		return nil, errors.Join(errz...)
	}

	// Create transport handler
	httpHandler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return cfg.server },
		nil,
	)

	return &Handler{
		server:      cfg.server,
		httpHandler: httpHandler,
	}, nil
}

// ServeHTTP implements http.Handler for HTTP transport
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.httpHandler.ServeHTTP(w, r)
}

// ServeSSE implements SSE transport by delegating to ServeHTTP
// The MCP SDK handles the transport differences internally
func (h *Handler) ServeSSE(w http.ResponseWriter, r *http.Request) {
	h.ServeHTTP(w, r)
}

// ServeStdio implements stdio transport for command-line tools
//
// NOTE: The stdin and stdout parameters are currently unused. The MCP SDK's
// StdioTransport always uses os.Stdin and os.Stdout. This is being addressed
// in upstream PR: https://github.com/modelcontextprotocol/go-sdk/pull/465
//
// TODO: Once PR #465 is merged, update to use custom stdin/stdout
func (h *Handler) ServeStdio(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	transport := &mcp.StdioTransport{}
	return h.server.Run(ctx, transport)
}

// createTypedHandler converts a simple typed function into an MCP ToolHandlerFor.
//
// The generics TIn and TOut allow users to define their own custom input/output schemas
// as Go structs, enabling automatic JSON schema generation rather than working with
// generic maps or predefined types.
//
// The returned lambda function acts as an adapter that:
//   - Calls the user's tool function with the deserialized input
//   - Handles error classification (tool errors vs protocol errors)
//   - Returns the typed output for SDK serialization
//
// Parameters:
//   - fn: User-defined tool function with custom input/output types
//
// Returns:
//   - MCP ToolHandlerFor lambda that bridges user code to SDK interface
func createTypedHandler[TIn, TOut any](fn ToolFunc[TIn, TOut]) mcp.ToolHandlerFor[TIn, TOut] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input TIn) (*mcp.CallToolResult, TOut, error) {
		// Execute the user-provided tool function
		output, err := fn(ctx, input)
		if err != nil {
			// Check if it's a tool error (user-facing error)
			var toolErr *ToolError
			if errors.As(err, &toolErr) {
				// Tool errors are returned as regular errors - the SDK will handle them
				var zero TOut
				return nil, zero, err
			}
			// Protocol error (system-level error) - return as Go error
			var zero TOut
			return nil, zero, err
		}

		// Success: return structured output (SDK handles serialization)
		return nil, output, nil
	}
}

// createSessionAwareToolHandler converts a session-aware tool function into an MCP ToolHandlerFor.
// This variant provides access to the ServerSession for elicitation capabilities.
func createSessionAwareToolHandler[TIn, TOut any](fn SessionAwareToolFunc[TIn, TOut]) mcp.ToolHandlerFor[TIn, TOut] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input TIn) (*mcp.CallToolResult, TOut, error) {
		// Extract the session from the request and create elicitation capability
		capability := GetElicitationCapability(req.Session)

		// Execute the user-provided tool function with elicitation capability
		output, err := fn(ctx, capability, input)
		if err != nil {
			// Check if it's a tool error (user-facing error)
			var toolErr *ToolError
			if errors.As(err, &toolErr) {
				// Tool errors are returned as regular errors - the SDK will handle them
				var zero TOut
				return nil, zero, err
			}
			// Protocol error (system-level error) - return as Go error
			var zero TOut
			return nil, zero, err
		}

		// Success: return structured output (SDK handles serialization)
		return nil, output, nil
	}
}

// createSessionAwarePromptHandler wraps a session-aware prompt function to match MCP PromptHandler signature
func createSessionAwarePromptHandler(fn SessionAwarePromptFunc) mcp.PromptHandler {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Extract the session from the request and create elicitation capability
		capability := GetElicitationCapability(req.Session)

		// Convert request parameters from map[string]string to map[string]any
		args := make(map[string]any)
		if req.Params.Arguments != nil {
			for k, v := range req.Params.Arguments {
				args[k] = v
			}
		}

		// Execute the user-provided prompt function with elicitation capability
		result, err := fn(ctx, capability, args)
		if err != nil {
			return nil, err
		}

		// Convert PromptResult to MCP format
		mcpMessages := make([]*mcp.PromptMessage, len(result.Messages))
		for i, msg := range result.Messages {
			mcpMessages[i] = &mcp.PromptMessage{
				Role:    mcp.Role(msg.Role),
				Content: &mcp.TextContent{Text: msg.Content},
			}
		}

		return &mcp.GetPromptResult{
			Description: result.Description,
			Messages:    mcpMessages,
		}, nil
	}
}

// createSessionAwareResourceHandler wraps a session-aware resource function to match MCP ResourceHandler signature
func createSessionAwareResourceHandler(fn SessionAwareResourceFunc) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		// Extract the session from the request and create elicitation capability
		capability := GetElicitationCapability(req.Session)

		// Execute the user-provided resource function with elicitation capability
		content, err := fn(ctx, capability, req.Params.URI)
		if err != nil {
			return nil, err
		}

		// Convert ResourceContent to MCP format
		mcpContent := &mcp.ResourceContents{
			URI:      req.Params.URI,
			MIMEType: content.MIMEType,
		}

		// Set content - ResourceContent has Content []byte field
		if len(content.Content) > 0 {
			// If it's text-like MIME type, set as text
			if strings.HasPrefix(content.MIMEType, "text/") ||
				content.MIMEType == "application/json" ||
				content.MIMEType == "application/xml" {
				mcpContent.Text = string(content.Content)
			} else {
				mcpContent.Blob = content.Content
			}
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{mcpContent},
		}, nil
	}
}
