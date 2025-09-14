package mcpio

import (
	"context"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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
			wantErr:        ErrEmptyName,
			wantNilHandler: true,
		},
		{
			name:           "empty version error",
			opts:           []Option{WithVersion("")},
			wantErr:        ErrEmptyVersion,
			wantNilHandler: true,
		},
		{
			name:           "nil server error",
			opts:           []Option{WithServer(nil)},
			wantErr:        ErrNilServer,
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
			wantErr:     ErrEmptyToolName,
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
	schema := CreateObjectSchema(
		"Raw tool input",
		map[string]string{"data": "Input data"},
		[]string{"data"},
	)

	tests := []struct {
		name        string
		toolName    string
		description string
		schema      interface{}
		wantErr     error
	}{
		{
			name:        "valid raw tool",
			toolName:    "process",
			description: "Process raw data",
			schema:      schema,
			wantErr:     nil,
		},
		{
			name:        "empty tool name error",
			toolName:    "",
			description: "description",
			schema:      schema,
			wantErr:     ErrEmptyToolName,
		},
		{
			name:        "nil schema error",
			toolName:    "test",
			description: "description",
			schema:      nil,
			wantErr:     ErrNilSchema,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var schemaPtr *jsonschema.Schema
			if tt.schema != nil {
				schemaPtr = tt.schema.(*jsonschema.Schema)
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
			wantErr: ErrEmptyName,
		},
		{
			name:    "invalid version option",
			opts:    []Option{WithVersion("")},
			wantErr: ErrEmptyVersion,
		},
		{
			name:    "invalid tool name",
			opts:    []Option{WithTool("", "desc", greetFunc)},
			wantErr: ErrEmptyToolName,
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
	assert.Contains(t, err.Error(), "only supports tools")
}
