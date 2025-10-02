package mcpio

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/robbyt/mcp-io/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test input/output types for testing
type testInput struct {
	Value string `json:"value"`
}

type testOutput struct {
	Result string `json:"result"`
}

// Test tool function
func testToolFunc(ctx context.Context, input testInput) (testOutput, error) {
	return testOutput{Result: "processed: " + input.Value}, nil
}

// Test raw tool function
func testRawToolFunc(ctx context.Context, input []byte) ([]byte, error) {
	return []byte(`{"processed": true}`), nil
}

func TestWithTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		toolName    string
		description string
		toolFunc    ToolFunc[testInput, testOutput]
		wantErr     error
	}{
		{
			name:        "valid tool",
			toolName:    "test-tool",
			description: "A test tool",
			toolFunc:    testToolFunc,
			wantErr:     nil,
		},
		{
			name:        "empty tool name should return error",
			toolName:    "",
			description: "A test tool",
			toolFunc:    testToolFunc,
			wantErr:     ErrEmptyValue,
		},
		{
			name:        "empty description should be valid",
			toolName:    "test-tool",
			description: "",
			toolFunc:    testToolFunc,
			wantErr:     nil,
		},
		{
			name:        "tool with special characters in name",
			toolName:    "test_tool-v1",
			description: "A test tool with special chars",
			toolFunc:    testToolFunc,
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &handlerConfig{
				tools: make([]toolRegisterFunc, 0),
			}
			option := WithTool(tt.toolName, tt.description, tt.toolFunc)
			err := option(cfg)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				// Should not add tool registration func on error
				assert.Empty(t, cfg.tools)
			} else {
				require.NoError(t, err)
				// Should add exactly one tool registration function
				assert.Len(t, cfg.tools, 1)
			}
		})
	}
}

func TestWithRawToolOptions(t *testing.T) {
	t.Parallel()

	validSchema := schema.NewObject(
		"Test input schema",
		map[string]string{"value": "Test value"},
		[]string{"value"},
	)

	// Example function matching the documentation
	rawFunc := func(ctx context.Context, input []byte) ([]byte, error) { return input, nil }

	tests := []struct {
		name        string
		toolName    string
		description string
		schema      any
		toolFunc    RawToolFunc
		wantErr     error
	}{
		{
			name:        "valid raw tool",
			toolName:    "raw-tool",
			description: "A raw test tool",
			schema:      validSchema,
			toolFunc:    testRawToolFunc,
			wantErr:     nil,
		},
		{
			name:        "WithRawTool with JSON string schema",
			toolName:    "process",
			description: "Process JSON",
			schema:      `{"type":"object"}`,
			toolFunc:    rawFunc,
			wantErr:     nil,
		},
		{
			name:        "empty tool name should return error",
			toolName:    "",
			description: "A raw test tool",
			schema:      validSchema,
			toolFunc:    testRawToolFunc,
			wantErr:     ErrEmptyValue,
		},
		{
			name:        "nil schema should return error",
			toolName:    "raw-tool",
			description: "A raw test tool",
			schema:      nil,
			toolFunc:    testRawToolFunc,
			wantErr:     ErrNilValue,
		},
		{
			name:        "empty description should be valid",
			toolName:    "raw-tool",
			description: "",
			schema:      validSchema,
			toolFunc:    testRawToolFunc,
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &handlerConfig{
				tools: make([]toolRegisterFunc, 0),
			}
			option := WithRawTool(tt.toolName, tt.description, tt.schema, tt.toolFunc)
			err := option(cfg)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				// Should not add tool registration func on error
				assert.Empty(t, cfg.tools)
			} else {
				require.NoError(t, err)
				// Should add exactly one tool registration function
				assert.Len(t, cfg.tools, 1)
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
		name     string
		schemas  *ToolSchemas
		toolFunc func(context.Context, map[string]any) (map[string]any, error)
		toolName string
		wantErr  bool
	}{
		{
			name: "InputSchema with email validation",
			schemas: &ToolSchemas{
				InputSchema: `{"type":"object","properties":{"email":{"format":"email"}}}`,
			},
			toolFunc: userFunc,
			toolName: "user",
			wantErr:  false,
		},
		{
			name: "InputSchema using json.RawMessage for performance",
			schemas: &ToolSchemas{
				InputSchema: json.RawMessage(`{"type":"object","additionalProperties":true}`),
			},
			toolFunc: fastFunc,
			toolName: "fast",
			wantErr:  false,
		},
		{
			name: "OutputSchema with score property",
			schemas: &ToolSchemas{
				OutputSchema: `{"type":"object","properties":{"score":{"type":"number"}}}`,
			},
			toolFunc: analyzeFunc,
			toolName: "analyze",
			wantErr:  false,
		},
		{
			name: "Both input and output schemas",
			schemas: &ToolSchemas{
				InputSchema:  `{"type":"object"}`,
				OutputSchema: `{"type":"object"}`,
			},
			toolFunc: convertFunc,
			toolName: "converter",
			wantErr:  false,
		},
		{
			name: "Invalid JSON input schema should error",
			schemas: &ToolSchemas{
				InputSchema: `{invalid json}`,
			},
			toolFunc: userFunc,
			toolName: "invalid-input",
			wantErr:  true,
		},
		{
			name:     "Nil schemas should work (uses auto-generated)",
			schemas:  nil,
			toolFunc: userFunc,
			toolName: "auto-schema",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewToolHandler(
				WithName("test-server"),
				WithToolWithSchema(tt.toolName, "Test tool", tt.toolFunc, tt.schemas),
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
