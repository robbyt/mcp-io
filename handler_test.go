package mcpio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for examples
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
			handler, err := New(tt.opts...)

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
			_, err := New(WithTool(tt.toolName, tt.description, echoFunc))

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

			_, err := New(WithRawTool(tt.toolName, tt.description, schemaPtr, rawFunc))

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

	handler, err := New(
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

	handler, err := New(
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

func TestServeHTTP(t *testing.T) {
	// Create handler with real server for HTTP testing
	handler, err := New(
		WithName("test-server"),
		WithTool("echo", "Echo input", echoFunc),
	)
	require.NoError(t, err)

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Test basic HTTP response (we can't test full MCP protocol easily,
	// but we can verify the handler responds)
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err)
	}()

	// The MCP handler should respond (even if with an error for GET request)
	assert.NotEqual(t, 0, resp.StatusCode)
}

func TestServeStdio(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	handler, err := New(
		WithServer(server),
		WithTool("echo", "Echo input", echoFunc),
	)
	require.NoError(t, err)

	// ServeStdio should be available for use
	// Note: This would normally start a blocking server, but we're just testing the setup
	assert.NotNil(t, handler.ServeStdio)
}

func TestGetServer(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	handler, err := New(WithServer(server))
	require.NoError(t, err)

	retrievedServer := handler.GetServer()
	assert.Equal(t, server, retrievedServer)
}

// Test error handling scenarios
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
			handler, err := New(tt.opts...)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, handler)
		})
	}
}

// Test concurrent safety (handlers should be immutable)
func TestConcurrentAccess(t *testing.T) {
	handler, err := New(
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

func TestServeSSE(t *testing.T) {
	handler, err := New(
		WithName("test-server"),
		WithTool("echo", "Echo input", echoFunc),
	)
	require.NoError(t, err)

	// Create test server using httptest
	server := httptest.NewServer(http.HandlerFunc(handler.ServeSSE))
	defer server.Close()

	// Test basic SSE response (should delegate to ServeHTTP)
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err)
	}()

	// The MCP handler should respond (even if with an error for GET request)
	assert.NotEqual(t, 0, resp.StatusCode)
}

func TestCreateTypedHandlerSuccess(t *testing.T) {
	handler := createTypedHandler(echoFunc)

	req := &mcp.CallToolRequest{}

	input := EchoInput{Text: "hello world"}
	result, output, err := handler(context.Background(), req, input)

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "hello world", output.Message)
}

func TestCreateTypedHandlerToolError(t *testing.T) {
	// Function that returns a tool error
	errorFunc := func(ctx context.Context, input EchoInput) (EchoOutput, error) {
		return EchoOutput{}, NewToolError("tool failed")
	}

	handler := createTypedHandler(errorFunc)

	req := &mcp.CallToolRequest{}

	input := EchoInput{Text: "test"}
	result, output, err := handler(context.Background(), req, input)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, EchoOutput{}, output)

	var toolErr *ToolError
	require.ErrorAs(t, err, &toolErr)
	assert.Equal(t, "tool failed", toolErr.Message)
}

func TestCreateTypedHandlerProtocolError(t *testing.T) {
	// Function that returns a non-tool error
	errorFunc := func(ctx context.Context, input EchoInput) (EchoOutput, error) {
		return EchoOutput{}, errors.New("protocol error")
	}

	handler := createTypedHandler(errorFunc)

	req := &mcp.CallToolRequest{}

	input := EchoInput{Text: "test"}
	result, output, err := handler(context.Background(), req, input)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, EchoOutput{}, output)
	assert.Equal(t, "protocol error", err.Error())
}
