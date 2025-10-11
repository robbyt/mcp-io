package mcpio

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for tool examples
type GreetInput struct {
	Name string `json:"name" jsonschema:"Name to greet"`
}

type GreetOutput struct {
	Message string `json:"message" jsonschema:"Greeting message"`
}

// Test helper functions
func greetFunc(ctx context.Context, input GreetInput) (GreetOutput, error) {
	return GreetOutput{Message: "Hello, " + input.Name}, nil
}

func farewellFunc(ctx context.Context, input GreetInput) (GreetOutput, error) {
	return GreetOutput{Message: "Goodbye, " + input.Name}, nil
}

func rawFunc(ctx context.Context, input []byte) ([]byte, error) {
	return []byte(`{"result": "processed"}`), nil
}

func TestHandlerConstruction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		opts           []Option
		wantErr        error
		wantNilHandler bool
	}{
		{
			name:    "basic handler",
			opts:    nil,
			wantErr: nil,
		},
		{
			name:    "with name and version",
			opts:    []Option{WithName("test-server"), WithVersion("1.2.3")},
			wantErr: nil,
		},
		{
			name:           "empty name error",
			opts:           []Option{WithName("")},
			wantErr:        ErrEmptyValue,
			wantNilHandler: true,
		},
		{
			name:           "empty version error",
			opts:           []Option{WithVersion("")},
			wantErr:        ErrEmptyValue,
			wantNilHandler: true,
		},
		{
			name:           "nil server error",
			opts:           []Option{WithServer(nil)},
			wantErr:        ErrNilValue,
			wantNilHandler: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewToolHandler(tt.opts...)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				if tt.wantNilHandler {
					assert.Nil(t, handler)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, handler)
				assert.NotNil(t, handler.server)
			}
		})
	}
}

func TestWithTypedTool(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		toolName    string
		description string
		wantErr     error
	}{
		{
			name:        "valid tool",
			toolName:    "greet",
			description: "Greet someone by name",
			wantErr:     nil,
		},
		{
			name:        "empty tool name error",
			toolName:    "",
			description: "description",
			wantErr:     ErrEmptyValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewToolHandler(WithTool(tt.toolName, tt.description, greetFunc))

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWithRawTool(t *testing.T) {
	t.Parallel()
	s := schema.NewObject(
		"Raw tool input",
		map[string]string{"data": "Input data"},
		[]string{"data"},
	)

	tests := []struct {
		name        string
		toolName    string
		description string
		s           interface{}
		wantErr     error
	}{
		{
			name:        "valid raw tool",
			toolName:    "process",
			description: "Process raw data",
			s:           s,
			wantErr:     nil,
		},
		{
			name:        "empty tool name error",
			toolName:    "",
			description: "description",
			s:           s,
			wantErr:     ErrEmptyValue,
		},
		{
			name:        "nil schema error",
			toolName:    "test",
			description: "description",
			s:           nil,
			wantErr:     ErrNilValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var schemaPtr *jsonschema.Schema
			if tt.s != nil {
				schemaPtr = tt.s.(*jsonschema.Schema)
			}

			_, err := NewToolHandler(WithRawTool(tt.toolName, tt.description, schemaPtr, rawFunc))

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestToolRegistration(t *testing.T) {
	t.Parallel()
	// Use real MCP server instead of mock
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	handler, err := NewToolHandler(
		WithServer(server),
		WithTool("greet", "Greet someone", greetFunc),
		WithTool("farewell", "Say goodbye", farewellFunc),
	)

	require.NoError(t, err)
	assert.NotNil(t, handler)
	assert.Equal(t, server, handler.GetServer())
}

func TestMultipleOptions(t *testing.T) {
	t.Parallel()
	// Use real MCP server instead of mock
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	handler, err := NewToolHandler(
		WithName("multi-tool-server"),
		WithVersion("1.2.3"),
		WithServer(server),
		WithTool("greet", "Greet someone", greetFunc),
		WithTool("farewell", "Say goodbye", farewellFunc),
	)

	require.NoError(t, err)
	assert.NotNil(t, handler)
	assert.Equal(t, server, handler.GetServer())
}

func TestErrorHandling(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		opts    []Option
		wantErr error
	}{
		{
			name:    "invalid name option",
			opts:    []Option{WithName("")},
			wantErr: ErrEmptyValue,
		},
		{
			name:    "invalid version option",
			opts:    []Option{WithVersion("")},
			wantErr: ErrEmptyValue,
		},
		{
			name:    "invalid tool name",
			opts:    []Option{WithTool("", "desc", greetFunc)},
			wantErr: ErrEmptyValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewToolHandler(tt.opts...)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, handler)
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()
	handler, err := NewToolHandler(
		WithName("concurrent-test"),
		WithTool("greet", "Greet someone", greetFunc),
	)
	require.NoError(t, err)

	// Multiple goroutines accessing the same handler should be safe
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			server := handler.GetServer()
			assert.NotNil(t, server)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestNewToolHandler_Validation(t *testing.T) {
	t.Parallel()

	// Test that NewToolHandler rejects non-tool resources
	promptFunc := func(ctx context.Context, args map[string]any) (*PromptResult, error) {
		return &PromptResult{Messages: []PromptMessage{{Role: "user", Content: "test"}}}, nil
	}

	_, err := NewToolHandler(
		WithName("invalid"),
		WithPrompt("test", "test prompt", promptFunc),
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrIncompatibleHandler)
}

// Helper function to create test requests for raw tool handlers
func createTestCallToolRequest(t *testing.T, args map[string]any) *mcp.CallToolRequest {
	t.Helper()
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal test arguments: %v", err)
	}
	return &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(argsJSON),
		},
	}
}

// Test createRawToolHandler function directly
func TestCreateRawToolHandler(t *testing.T) {
	t.Parallel()

	t.Run("success path with valid JSON", func(t *testing.T) {
		// Raw function that returns valid JSON
		rawFunc := func(ctx context.Context, input []byte) ([]byte, error) {
			return []byte(`{"result": "success", "input_length": ` + strconv.Itoa(len(input)) + `}`), nil
		}

		handler := createRawToolHandler(rawFunc)

		req := createTestCallToolRequest(t, map[string]any{
			"test": "value",
			"num":  42,
		})

		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		assert.True(t, ok)
		assert.Contains(t, textContent.Text, `"result": "success"`)
	})

	t.Run("input marshaling error", func(t *testing.T) {
		// Raw function (won't be called due to marshaling error)
		rawFunc := func(ctx context.Context, input []byte) ([]byte, error) {
			return []byte(`{"result": "should not reach"}`), nil
		}

		handler := createRawToolHandler(rawFunc)

		// Create request with invalid JSON that will cause marshaling to fail
		// We'll pass invalid JSON as RawMessage directly
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Arguments: json.RawMessage(`{invalid json content`), // Invalid JSON
			},
		}

		result, err := handler(context.Background(), req)

		require.NoError(t, err) // Handler should not return protocol error
		assert.NotNil(t, result)
		assert.True(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		assert.True(t, ok)
		assert.Contains(t, textContent.Text, "Failed to marshal input")
	})

	t.Run("raw function returns ToolError", func(t *testing.T) {
		// Raw function that returns a ToolError
		rawFunc := func(ctx context.Context, input []byte) ([]byte, error) {
			return nil, NewToolError("validation failed")
		}

		handler := createRawToolHandler(rawFunc)

		req := createTestCallToolRequest(t, map[string]any{"test": "data"})

		result, err := handler(context.Background(), req)

		require.NoError(t, err) // Handler should not return protocol error
		assert.NotNil(t, result)
		assert.True(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		assert.True(t, ok)
		assert.Equal(t, "validation failed", textContent.Text)
	})

	t.Run("raw function returns ToolError with code", func(t *testing.T) {
		// Raw function that returns a ToolError with error code
		rawFunc := func(ctx context.Context, input []byte) ([]byte, error) {
			return nil, ValidationError("invalid input format")
		}

		handler := createRawToolHandler(rawFunc)

		req := createTestCallToolRequest(t, map[string]any{"data": "invalid"})

		result, err := handler(context.Background(), req)

		require.NoError(t, err) // Handler should not return protocol error
		assert.NotNil(t, result)
		assert.True(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		assert.True(t, ok)
		// Error codes should be included with [CODE] prefix
		assert.Equal(t, "[VALIDATION_ERROR] invalid input format", textContent.Text)
	})

	t.Run("raw function returns protocol error", func(t *testing.T) {
		errDatabaseFailed := errors.New("database connection failed")

		// Raw function that returns a regular Go error (protocol error)
		rawFunc := func(ctx context.Context, input []byte) ([]byte, error) {
			return nil, errDatabaseFailed
		}

		handler := createRawToolHandler(rawFunc)

		req := createTestCallToolRequest(t, map[string]any{"query": "test"})

		result, err := handler(context.Background(), req)

		// Protocol errors should be returned as Go errors, not CallToolResult
		require.ErrorIs(t, err, errDatabaseFailed)
		assert.Nil(t, result)
	})

	t.Run("invalid JSON output", func(t *testing.T) {
		// Raw function that returns invalid JSON
		rawFunc := func(ctx context.Context, input []byte) ([]byte, error) {
			return []byte(`{invalid json:`), nil
		}

		handler := createRawToolHandler(rawFunc)

		req := createTestCallToolRequest(t, map[string]any{"test": "data"})

		result, err := handler(context.Background(), req)

		// Invalid JSON should be a protocol error
		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrInvalidJSON)
	})

	t.Run("empty JSON object output", func(t *testing.T) {
		// Raw function that returns empty JSON object
		rawFunc := func(ctx context.Context, input []byte) ([]byte, error) {
			return []byte(`{}`), nil
		}

		handler := createRawToolHandler(rawFunc)

		req := createTestCallToolRequest(t, map[string]any{"test": "data"})

		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		assert.True(t, ok)
		assert.Equal(t, "{}", textContent.Text)
	})

	t.Run("JSON array output", func(t *testing.T) {
		// Raw function that returns JSON array
		rawFunc := func(ctx context.Context, input []byte) ([]byte, error) {
			return []byte(`[1, 2, 3]`), nil
		}

		handler := createRawToolHandler(rawFunc)

		req := createTestCallToolRequest(t, map[string]any{"count": 3})

		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		assert.True(t, ok)
		assert.Equal(t, "[1, 2, 3]", textContent.Text)
	})
}
