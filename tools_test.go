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
type EchoInput struct {
	Text string `json:"text" jsonschema:"Text to echo"`
}

type EchoOutput struct {
	Message string `json:"message" jsonschema:"Echoed message"`
}

type CalculateInput struct {
	Operation string  `json:"operation" jsonschema:"Operation to perform"`
	A         float64 `json:"a"         jsonschema:"First number"`
	B         float64 `json:"b"         jsonschema:"Second number"`
}

type CalculateOutput struct {
	Result float64 `json:"result" jsonschema:"Calculation result"`
}

// Test helper functions
func echoFunc(ctx context.Context, input EchoInput) (EchoOutput, error) {
	return EchoOutput{Message: input.Text}, nil
}

func calculateFunc(ctx context.Context, input CalculateInput) (CalculateOutput, error) {
	var result float64
	switch input.Operation {
	case "add":
		result = input.A + input.B
	case "subtract":
		result = input.A - input.B
	case "multiply":
		result = input.A * input.B
	case "divide":
		if input.B == 0 {
			return CalculateOutput{}, NewToolError("division by zero")
		}
		result = input.A / input.B
	default:
		return CalculateOutput{}, ValidationError("unsupported operation: " + input.Operation)
	}
	return CalculateOutput{Result: result}, nil
}

func rawFunc(ctx context.Context, input []byte) ([]byte, error) {
	return []byte(`{"result": "processed"}`), nil
}

func TestHandlerConstruction(t *testing.T) {
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
	tests := []struct {
		name        string
		toolName    string
		description string
		wantErr     error
	}{
		{
			name:        "valid tool",
			toolName:    "echo",
			description: "Echo input text",
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
			_, err := NewToolHandler(WithTool(tt.toolName, tt.description, echoFunc))

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWithRawTool(t *testing.T) {
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
	// Use real MCP server instead of mock
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	handler, err := NewToolHandler(
		WithServer(server),
		WithTool("echo", "Echo text", echoFunc),
		WithTool("calc", "Calculate", calculateFunc),
	)

	require.NoError(t, err)
	assert.NotNil(t, handler)
	assert.Equal(t, server, handler.GetServer())
}

func TestMultipleOptions(t *testing.T) {
	// Use real MCP server instead of mock
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	handler, err := NewToolHandler(
		WithName("multi-tool-server"),
		WithVersion("1.2.3"),
		WithServer(server),
		WithTool("echo", "Echo input", echoFunc),
		WithTool("calculate", "Perform arithmetic", calculateFunc),
	)

	require.NoError(t, err)
	assert.NotNil(t, handler)
	assert.Equal(t, server, handler.GetServer())
}

func TestErrorHandling(t *testing.T) {
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
			opts:    []Option{WithTool("", "desc", echoFunc)},
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
	handler, err := NewToolHandler(
		WithName("concurrent-test"),
		WithTool("echo", "Echo input", echoFunc),
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
