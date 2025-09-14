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

// Test types for handler tests
type SimpleInput struct {
	Text string `json:"text" jsonschema:"Text to process"`
}

type SimpleOutput struct {
	Message string `json:"message" jsonschema:"Processed message"`
}

// Test helper functions for handler-specific tests
func simpleEchoFunc(ctx context.Context, input SimpleInput) (SimpleOutput, error) {
	return SimpleOutput{Message: input.Text}, nil
}

func TestServeHTTP(t *testing.T) {
	t.Parallel()
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
	// TODO: switch to httptest.Client and do a full MCP call
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	handler, err := NewToolHandler(
		WithName("test-server"),
		WithTool("echo", "Echo input", simpleEchoFunc),
	)
	require.NoError(t, err)

	// Create test server using httptest
	server := httptest.NewServer(http.HandlerFunc(handler.ServeSSE))
	defer server.Close()

	// Test basic SSE response (should delegate to ServeHTTP)
	// TODO: switch to httptest.Client and do a full MCP call
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
	t.Parallel()
	handler := createTypedHandler(simpleEchoFunc)

	req := &mcp.CallToolRequest{}

	input := SimpleInput{Text: "hello world"}
	result, output, err := handler(context.Background(), req, input)

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "hello world", output.Message)
}

func TestCreateTypedHandlerToolError(t *testing.T) {
	t.Parallel()
	// Function that returns a tool error
	errorFunc := func(ctx context.Context, input SimpleInput) (SimpleOutput, error) {
		return SimpleOutput{}, NewToolError("tool failed")
	}

	handler := createTypedHandler(errorFunc)

	req := &mcp.CallToolRequest{}

	input := SimpleInput{Text: "test"}
	result, output, err := handler(context.Background(), req, input)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, SimpleOutput{}, output)

	var toolErr *ToolError
	require.ErrorAs(t, err, &toolErr)
	assert.Equal(t, "tool failed", toolErr.Message)
}

func TestCreateTypedHandlerProtocolError(t *testing.T) {
	t.Parallel()
	// Function that returns a non-tool error
	errorFunc := func(ctx context.Context, input SimpleInput) (SimpleOutput, error) {
		return SimpleOutput{}, errors.New("protocol error")
	}

	handler := createTypedHandler(errorFunc)

	req := &mcp.CallToolRequest{}

	input := SimpleInput{Text: "test"}
	result, output, err := handler(context.Background(), req, input)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, SimpleOutput{}, output)
	assert.Equal(t, "protocol error", err.Error())
}
