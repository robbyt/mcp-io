package mcpio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Handler is the main MCP handler struct
type Handler struct {
	server      *mcp.Server
	httpHandler http.Handler
}

// New creates a new MCP handler with the given options
func New(opts ...Option) (*Handler, error) {
	cfg := &handlerConfig{
		name:    "mcp-server",
		version: "1.0.0",
		tools:   make([]*toolRegistration, 0),
	}

	// Apply all options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Use injected server or create default
	var server *mcp.Server
	if cfg.server != nil {
		server = cfg.server
	} else {
		impl := &mcp.Implementation{
			Name:    cfg.name,
			Version: cfg.version,
		}
		server = mcp.NewServer(impl, nil)
	}

	// Register all tools
	for _, tool := range cfg.tools {
		tool.registerFunc(server)
	}

	// Create transport handler
	httpHandler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		nil,
	)

	return &Handler{
		server:      server,
		httpHandler: httpHandler,
	}, nil
}

// GetServer returns the underlying MCP server for advanced usage
func (h *Handler) GetServer() *mcp.Server {
	return h.server
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
func (h *Handler) ServeStdio(stdin io.Reader, stdout io.Writer) error {
	transport := &mcp.StdioTransport{}
	return h.server.Run(context.Background(), transport)
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
func createTypedHandler[TIn, TOut any](fn func(context.Context, TIn) (TOut, error)) mcp.ToolHandlerFor[TIn, TOut] {
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

// createRawHandler wraps a raw function to match the MCP ToolHandler signature
func createRawHandler(fn RawToolFunc) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Marshal input arguments to JSON bytes
		inputJSON, err := json.Marshal(req.Params.Arguments)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Failed to marshal input: %v", err)},
				},
				IsError: true,
			}, nil
		}

		// Execute raw function
		outputJSON, err := fn(ctx, inputJSON)
		if err != nil {
			// Check if it's a tool error
			var toolErr *ToolError
			if errors.As(err, &toolErr) {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: toolErr.Message},
					},
					IsError: true,
				}, nil
			}
			// Protocol error
			return nil, err
		}

		// Parse output for structured response
		var output any
		if err := json.Unmarshal(outputJSON, &output); err != nil {
			// Raw tools must return valid JSON
			return nil, errors.Join(ErrInvalidJSON, err)
		}

		// Return structured output
		outputJSONStr := string(outputJSON)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: outputJSONStr},
			},
		}, nil
	}
}
