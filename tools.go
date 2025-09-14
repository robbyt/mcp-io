package mcpio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolRegisterFunc is an internal function type that registers a tool on an MCP server.
// This is used internally by the option functions to defer tool registration.
type toolRegisterFunc func(*mcp.Server)

// ToolFunc is the function signature for typed tools with automatic schema generation.
// The function receives a context and typed input, and returns typed output with an optional error.
// Schema generation is handled automatically based on the TIn and TOut types.
type ToolFunc[TIn, TOut any] func(context.Context, TIn) (TOut, error)

// RawToolFunc is the function signature for raw JSON tools.
// The function receives a context and raw JSON bytes as input, and returns JSON bytes as output.
// Schema must be provided explicitly when using WithRawTool.
type RawToolFunc func(context.Context, []byte) ([]byte, error)

// NewToolHandler creates a new MCP handler with the given options
func NewToolHandler(opts ...Option) (*Handler, error) {
	cfg := &handlerConfig{
		name:    "mcp-server",
		version: "1.0.0",
		tools:   make([]toolRegisterFunc, 0),
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

	// Register all tools
	for _, toolRegisterFunc := range cfg.tools {
		toolRegisterFunc(cfg.server)
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
