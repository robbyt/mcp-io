package mcpio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolSchemas allows overriding auto-generated schemas for tools.
// When provided to WithTool, these completely replace schema generation from TIn/TOut types.
//
// Examples:
//
//	// Override only input schema
//	schemas := &ToolSchemas{
//	    InputSchema: `{"type":"object","properties":{"name":{"type":"string"}}}`,
//	}
//	WithTool("create_user", "Create a user", userFunc, schemas)
//
//	// Override both schemas with json.RawMessage for best performance
//	schemas := &ToolSchemas{
//	    InputSchema:  json.RawMessage(`{"type":"object"}`),
//	    OutputSchema: json.RawMessage(`{"type":"object"}`),
//	}
//	WithTool("process", "Process data", processFunc, schemas)
type ToolSchemas struct {
	InputSchema  any // Can be json.RawMessage, string, *jsonschema.Schema, or map[string]any
	OutputSchema any // Can be json.RawMessage, string, *jsonschema.Schema, or map[string]any
}

// toolRegisterFunc is an internal function type that registers a tool on an MCP server.
// This is used internally by the option functions to defer tool registration.
type toolRegisterFunc func(*mcp.Server) error

// ToolFunc is the function signature for typed tools with automatic schema generation.
// The function receives a context and typed input, and returns typed output with an optional error.
// Schema generation is handled automatically based on the TIn and TOut types.
type ToolFunc[TIn, TOut any] func(context.Context, TIn) (TOut, error)

// RawToolFunc is the function signature for raw JSON tools.
// The function receives a context and raw JSON bytes as input, and returns JSON bytes as output.
// Schema must be provided explicitly when using WithRawTool.
type RawToolFunc func(context.Context, []byte) ([]byte, error)

// NewToolHandler creates a new MCP handler that only supports tools.
// For mixed resource types, use NewHandler instead.
func NewToolHandler(opts ...Option) (*Handler, error) {
	cfg := &handlerConfig{
		name:    "mcp-server",
		version: "1.0.0",
		tools:   make([]toolRegisterFunc, 0),
	}

	// Apply options - they will validate handler type compatibility
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Delegate to the unified constructor
	return NewHandler(opts...)
}

// createRawToolHandler wraps a raw function to match the MCP ToolHandler signature
func createRawToolHandler(fn RawToolFunc) mcp.ToolHandler {
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
