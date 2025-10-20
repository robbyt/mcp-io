// Package tool provides constructors for MCP Tool primitives using functional options.
// This package implements the Tool primitive as defined in the MCP specification
// (https://modelcontextprotocol.io/specification/2025-06-18/server/tools).
package tool

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/schema"
)

// New creates a new mcp.Tool with required fields and optional configuration.
//
// This function is primarily intended for use with WithRawTool where inputSchema
// must be provided explicitly. For typed tools using WithTool, the input schema
// is auto-generated from the generic type parameter.
//
// Parameters:
//   - name: Unique identifier for the tool (required, non-empty)
//   - description: Human-readable description for the LLM (required, non-empty)
//   - inputSchema: JSON Schema defining expected parameters (required, non-nil)
//     Can be json.RawMessage, string, *jsonschema.Schema, or map[string]any
//   - opts: Optional configuration functions (WithTitle, WithReadOnly, etc.)
//
// Example:
//
//	tool, err := tool.New("get_weather", "Get weather for a location", inputSchema,
//	    tool.WithTitle("Weather Getter"),
//	    tool.WithReadOnly(),
//	)
func New(name, description string, inputSchema any, opts ...Option) (*mcp.Tool, error) {
	// Validate required fields
	if name == "" {
		return nil, fmt.Errorf("tool name cannot be empty")
	}
	if description == "" {
		return nil, fmt.Errorf("tool description cannot be empty")
	}
	if inputSchema == nil {
		return nil, fmt.Errorf("tool input schema cannot be nil")
	}

	// Convert input schema to json.RawMessage for optimal SDK performance
	convertedInput, err := schema.ConvertToRawMessage(inputSchema)
	if err != nil {
		return nil, fmt.Errorf("invalid input schema for tool %q: %w", name, err)
	}

	// Build the tool
	tool := &mcp.Tool{
		Name:        name,
		Description: description,
		InputSchema: convertedInput,
	}

	// Apply options directly to the tool
	for _, opt := range opts {
		if err := opt(tool); err != nil {
			return nil, err
		}
	}

	return tool, nil
}
