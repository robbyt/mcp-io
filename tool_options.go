package mcpio

import (
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolOption allows customizing tool schema configuration
type ToolOption interface {
	apply(tool *mcp.Tool) error
}

// inputSchemaOption sets the input schema for a tool
type inputSchemaOption struct {
	schema any
}

func (o inputSchemaOption) apply(tool *mcp.Tool) error {
	converted, err := convertToRawMessage(o.schema)
	if err != nil {
		return fmt.Errorf("invalid input schema: %w", err)
	}
	tool.InputSchema = converted
	return nil
}

// outputSchemaOption sets the output schema for a tool
type outputSchemaOption struct {
	schema any
}

func (o outputSchemaOption) apply(tool *mcp.Tool) error {
	converted, err := convertToRawMessage(o.schema)
	if err != nil {
		return fmt.Errorf("invalid output schema: %w", err)
	}
	tool.OutputSchema = converted
	return nil
}

// WithInputSchema overrides the auto-generated input schema from Go struct types.
// When used with WithTool, this completely replaces schema generation from the TIn type.
// Useful for adding validation constraints, descriptions, or supporting flexible input.
//
// Examples:
//
//	// Add validation to struct-based tool
//	WithTool("user", "Create user", userFunc,
//	    WithInputSchema(`{"type":"object","properties":{"email":{"format":"email"}}}`))
//
//	// Performance optimization with json.RawMessage
//	schema := json.RawMessage(`{"type":"object","additionalProperties":true}`)
//	WithTool("fast", "High-speed processing", fastFunc, WithInputSchema(schema))
func WithInputSchema(schema any) ToolOption {
	return inputSchemaOption{schema: schema}
}

// WithOutputSchema overrides the auto-generated output schema from Go struct types.
// When used with WithTool, this completely replaces schema generation from the TOut type.
// Use this to document API responses, add metadata, or specify flexible output.
//
//	WithTool("analyze", "Text analysis", analyzeFunc,
//	    WithOutputSchema(`{"type":"object","properties":{"score":{"type":"number"}}}`)
func WithOutputSchema(schema any) ToolOption {
	return outputSchemaOption{schema: schema}
}

// WithSchemas overrides both auto-generated input and output schemas.
// When used with WithTool, this completely replaces schema generation from TIn/TOut types.
// Convenience function for setting both schemas at once.
//
//	WithTool("converter", "Convert units", convertFunc,
//	    WithSchemas(`{"type":"object"}`, `{"type":"object"}`))
func WithSchemas(inputSchema, outputSchema any) ToolOption {
	return combinedSchemaOption{
		input:  inputSchema,
		output: outputSchema,
	}
}

type combinedSchemaOption struct {
	input  any
	output any
}

func (o combinedSchemaOption) apply(tool *mcp.Tool) error {
	if err := WithInputSchema(o.input).apply(tool); err != nil {
		return err
	}
	return WithOutputSchema(o.output).apply(tool)
}

// convertToRawMessage converts any schema type to json.RawMessage for optimal performance with
// the MCP SDK. json.RawMessage has zero marshaling overhead during tool listing, so it uses fewer
// resources.
func convertToRawMessage(schema any) (json.RawMessage, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema cannot be nil")
	}

	switch v := schema.(type) {
	case json.RawMessage:
		// Already optimal format
		return v, nil

	case string:
		// Convert JSON string to json.RawMessage
		if !json.Valid([]byte(v)) {
			return nil, fmt.Errorf("invalid JSON string: %s", v)
		}
		return json.RawMessage(v), nil

	case *jsonschema.Schema:
		// Convert to json.RawMessage for optimal performance
		bytes, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal jsonschema.Schema: %w", err)
		}
		return json.RawMessage(bytes), nil

	case map[string]any:
		// Convert to json.RawMessage
		bytes, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal map[string]any: %w", err)
		}
		return json.RawMessage(bytes), nil

	default:
		// Try to marshal any other type
		bytes, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("unsupported schema type %T: failed to marshal: %w", v, err)
		}
		return json.RawMessage(bytes), nil
	}
}

// convertToJSONSchema converts any schema type to *jsonschema.Schema for
// internal processing that requires accessing specific schema fields like
// Properties and Required (e.g., schemaToPromptArguments function).
func convertToJSONSchema(schema any) (*jsonschema.Schema, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema cannot be nil")
	}

	// If it's already a *jsonschema.Schema, use it directly
	if s, ok := schema.(*jsonschema.Schema); ok {
		return s, nil
	}

	// Convert other types by marshaling to JSON and unmarshaling to *jsonschema.Schema
	jsonBytes, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema to JSON: %w", err)
	}

	var result *jsonschema.Schema
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema from JSON: %w", err)
	}

	return result, nil
}
