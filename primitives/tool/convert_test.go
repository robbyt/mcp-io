package tool

import (
	"encoding/json"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToRawMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       any
		wantError   bool
		validateOut func(t *testing.T, result json.RawMessage)
	}{
		{
			name:      "json.RawMessage input",
			input:     json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`),
			wantError: false,
			validateOut: func(t *testing.T, result json.RawMessage) {
				t.Helper()
				assert.JSONEq(t, `{"type":"object","properties":{"name":{"type":"string"}}}`, string(result))
			},
		},
		{
			name:      "string JSON input",
			input:     `{"type":"object","properties":{"value":{"type":"integer"}}}`,
			wantError: false,
			validateOut: func(t *testing.T, result json.RawMessage) {
				t.Helper()
				assert.JSONEq(t, `{"type":"object","properties":{"value":{"type":"integer"}}}`, string(result))
			},
		},
		{
			name: "*jsonschema.Schema input",
			input: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"count": {Type: "number", Description: "A count value"},
				},
				Required: []string{"count"},
			},
			wantError: false,
			validateOut: func(t *testing.T, result json.RawMessage) {
				t.Helper()
				var parsed map[string]any
				require.NoError(t, json.Unmarshal(result, &parsed))
				assert.Equal(t, "object", parsed["type"])
				assert.Contains(t, parsed, "properties")
				assert.Contains(t, parsed, "required")
			},
		},
		{
			name: "map[string]any input",
			input: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{
						"type": "string",
						"enum": []string{"active", "inactive"},
					},
				},
				"required": []string{"status"},
			},
			wantError: false,
			validateOut: func(t *testing.T, result json.RawMessage) {
				t.Helper()
				assert.JSONEq(t, `{"type":"object","properties":{"status":{"type":"string","enum":["active","inactive"]}},"required":["status"]}`, string(result))
			},
		},
		{
			name:      "nil input should error",
			input:     nil,
			wantError: true,
		},
		{
			name:      "invalid JSON string should error",
			input:     `{invalid json}`,
			wantError: true,
		},
		{
			name:      "unmarshalable type should error",
			input:     make(chan int),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertToRawMessage(tt.input)

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateOut != nil {
					tt.validateOut(t, result)
				}
			}
		})
	}
}

func TestConvertToRawMessagePerformance(t *testing.T) {
	t.Parallel()

	inputSchema := `{"type":"object","properties":{"data":{"type":"string"}},"required":["data"]}`

	// Test conversion from string to json.RawMessage
	result1, err1 := convertToRawMessage(inputSchema)
	require.NoError(t, err1)

	// Test conversion from json.RawMessage (should be identity)
	result2, err2 := convertToRawMessage(json.RawMessage(inputSchema))
	require.NoError(t, err2)

	// Both should produce identical json.RawMessage
	assert.Equal(t, result1, result2)
	assert.IsType(t, json.RawMessage{}, result1)
	assert.IsType(t, json.RawMessage{}, result2)

	// Verify the content is correct
	assert.JSONEq(t, inputSchema, string(result1))
	assert.JSONEq(t, inputSchema, string(result2))
}
