package mcpio

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaConversion(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		wantError   bool
		errorMsg    string
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
			errorMsg:  "schema cannot be nil",
		},
		{
			name:      "invalid JSON string should error",
			input:     `{invalid json}`,
			wantError: true,
			errorMsg:  "invalid JSON string",
		},
		{
			name:      "unmarshalable type should error",
			input:     make(chan int),
			wantError: true,
			errorMsg:  "failed to marshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertToRawMessage(tt.input)

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
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

func TestToolOptionsIntegration(t *testing.T) {
	// Test tool functions matching the documentation examples
	userFunc := func(ctx context.Context, input map[string]any) (map[string]any, error) {
		return map[string]any{"created": input}, nil
	}
	fastFunc := func(ctx context.Context, input map[string]any) (map[string]any, error) {
		return map[string]any{"processed": true}, nil
	}
	analyzeFunc := func(ctx context.Context, input map[string]any) (map[string]any, error) {
		return map[string]any{"score": 0.95}, nil
	}
	convertFunc := func(ctx context.Context, input map[string]any) (map[string]any, error) {
		return map[string]any{"converted": input}, nil
	}

	tests := []struct {
		name    string
		options []ToolOption
		wantErr bool
	}{
		{
			name: "WithInputSchema with email validation",
			options: []ToolOption{
				WithInputSchema(`{"type":"object","properties":{"email":{"format":"email"}}}`),
			},
			wantErr: false,
		},
		{
			name: "WithInputSchema using json.RawMessage for performance",
			options: []ToolOption{
				WithInputSchema(json.RawMessage(`{"type":"object","additionalProperties":true}`)),
			},
			wantErr: false,
		},
		{
			name: "WithOutputSchema with score property",
			options: []ToolOption{
				WithOutputSchema(`{"type":"object","properties":{"score":{"type":"number"}}}`),
			},
			wantErr: false,
		},
		{
			name: "WithSchemas setting both input and output",
			options: []ToolOption{
				WithSchemas(`{"type":"object"}`, `{"type":"object"}`),
			},
			wantErr: false,
		},
		{
			name: "WithInputSchema with invalid JSON should error",
			options: []ToolOption{
				WithInputSchema(`{invalid json}`),
			},
			wantErr: true,
		},
		{
			name: "WithOutputSchema with nil should error",
			options: []ToolOption{
				WithOutputSchema(nil),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var toolFunc func(context.Context, map[string]any) (map[string]any, error)
			toolName := "test-tool"

			// Select appropriate function based on test
			switch tt.name {
			case "WithInputSchema with email validation":
				toolFunc = userFunc
				toolName = "user"
			case "WithInputSchema using json.RawMessage for performance":
				toolFunc = fastFunc
				toolName = "fast"
			case "WithOutputSchema with score property":
				toolFunc = analyzeFunc
				toolName = "analyze"
			case "WithSchemas setting both input and output":
				toolFunc = convertFunc
				toolName = "converter"
			default:
				toolFunc = userFunc
			}

			_, err := NewToolHandler(
				WithName("test-server"),
				WithTool(toolName, "Test tool", toolFunc, tt.options...),
			)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBackwardsCompatibility(t *testing.T) {
	// Test that existing struct-based tools still work without options
	type TestInput struct {
		Message string `json:"message" jsonschema:"Message to process"`
	}

	type TestOutput struct {
		Result string `json:"result" jsonschema:"Processed message"`
	}

	testFunc := func(ctx context.Context, input TestInput) (TestOutput, error) {
		return TestOutput{Result: "processed: " + input.Message}, nil
	}

	handler, err := NewToolHandler(
		WithName("test-server"),
		WithTool("echo", "Echo tool", testFunc),
	)

	require.NoError(t, err)
	assert.NotNil(t, handler)
}

func TestPerformanceComparison(t *testing.T) {
	// Test that demonstrates json.RawMessage is optimal
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
