package mcpio

import (
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
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

// convertToRawMessage converts any schema type to json.RawMessage for optimal performance with
// the MCP SDK. json.RawMessage has zero marshaling overhead during tool listing, so it uses fewer
// resources.
func convertToRawMessage(schema any) (json.RawMessage, error) {
	if schema == nil {
		return nil, ErrNilSchema
	}

	switch v := schema.(type) {
	case json.RawMessage:
		// Already optimal format
		return v, nil

	case string:
		// Convert JSON string to json.RawMessage
		if !json.Valid([]byte(v)) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidJSONSchema, v)
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
			return nil, fmt.Errorf("%w %T: %w", ErrUnsupportedSchemaType, v, err)
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
