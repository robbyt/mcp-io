package tool

import (
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

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
			return nil, fmt.Errorf("invalid JSON schema: %s", v)
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
			return nil, fmt.Errorf("unsupported schema type %T: %w", v, err)
		}
		return json.RawMessage(bytes), nil
	}
}
