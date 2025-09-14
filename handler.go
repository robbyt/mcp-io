package mcpio

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// handlerConfig holds the configuration built by options
type handlerConfig struct {
	name    string
	version string
	tools   []toolRegisterFunc
	server  *mcp.Server // The MCP-SDK server instance
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
// TODO: Add context support and graceful shutdown
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
