package mcpio

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// createTypedHandler converts a simple typed function into an MCP ToolHandlerFor.
//
// The generics TIn and TOut allow users to define their own custom input/output schemas
// as Go structs, enabling automatic JSON schema generation rather than working with
// generic maps or predefined types.
//
// The returned lambda function acts as an adapter that:
//   - Injects the session into the context
//   - Calls the user's tool function with the deserialized input
//   - Handles error classification (tool errors vs protocol errors)
//   - Returns the typed output for SDK serialization
//
// Parameters:
//   - fn: User-defined tool function with custom input/output types
//
// Returns:
//   - MCP ToolHandlerFor lambda that bridges user code to SDK interface
func createTypedHandler[TIn, TOut any](handler *Handler, fn ToolFunc[TIn, TOut]) mcp.ToolHandlerFor[TIn, TOut] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input TIn) (*mcp.CallToolResult, TOut, error) {
		// Create request context with all MCP metadata
		reqCtx := newRequestContext(req.Params.Name, req.Session, req.Extra)

		// Execute the user-provided tool function (pass reqCtx as ToolContext)
		output, err := fn(ctx, reqCtx, input)
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
