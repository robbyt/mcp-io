package mcpio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions for handler-specific tests
func simpleEchoFunc(ctx context.Context, input EchoInput) (EchoOutput, error) {
	return EchoOutput{Message: input.Text}, nil
}

func TestServeHTTP(t *testing.T) {
	// Create handler with real server for HTTP testing
	handler, err := NewToolHandler(
		WithName("test-server"),
		WithTool("echo", "Echo input", simpleEchoFunc),
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

	handler, err := NewToolHandler(
		WithServer(server),
		WithTool("echo", "Echo input", simpleEchoFunc),
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

	handler, err := NewToolHandler(WithServer(server))
	require.NoError(t, err)

	retrievedServer := handler.GetServer()
	assert.Equal(t, server, retrievedServer)
}

func TestServeSSE(t *testing.T) {
	handler, err := NewToolHandler(
		WithName("test-server"),
		WithTool("echo", "Echo input", simpleEchoFunc),
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
	handler := createTypedHandler(simpleEchoFunc)

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
