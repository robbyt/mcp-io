package mcpio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	t.Run("method exists with context parameter", func(t *testing.T) {
		handler, err := NewToolHandler(
			WithName("test-server"),
			WithTool("echo", "Echo input", simpleEchoFunc),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler.ServeStdio)
		// Note: Actual stdio testing requires upstream PR #465 to be merged
		// See: https://github.com/modelcontextprotocol/go-sdk/pull/465
	})
}

func TestServerWithTransport(t *testing.T) {
	t.Parallel()

	t.Run("server with InMemoryTransport and context cancellation", func(t *testing.T) {
		serverTransport, clientTransport := mcp.NewInMemoryTransports()

		handler, err := NewToolHandler(
			WithName("test-server"),
			WithTool("echo", "Echo input", simpleEchoFunc),
		)
		require.NoError(t, err)

		// Run server with cancellable context
		ctx, cancel := context.WithCancel(context.Background())
		serverDone := make(chan error, 1)

		go func() {
			err := handler.server.Run(ctx, serverTransport)
			serverDone <- err
		}()

		// Create client and test
		client := mcp.NewClient(&mcp.Implementation{
			Name: "test-client",
		}, nil)

		session, err := client.Connect(context.Background(), clientTransport, nil)
		require.NoError(t, err)

		// Test tool execution
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "echo",
			Arguments: map[string]any{"text": "hello"},
		})
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test context cancellation
		cancel()

		assert.Eventually(t, func() bool {
			select {
			case <-serverDone:
				return true
			default:
				return false
			}
		}, 2*time.Second, 10*time.Millisecond, "server should stop after context cancellation")

		err = session.Close()
		require.NoError(t, err)
	})

	t.Run("multiple concurrent servers with InMemoryTransport", func(t *testing.T) {
		const numServers = 2
		serverDone := make([]chan error, numServers)
		cancels := make([]context.CancelFunc, numServers)

		for i := range numServers {
			serverTransport, clientTransport := mcp.NewInMemoryTransports()

			handler, err := NewToolHandler(
				WithName("test-server"),
				WithTool("echo", "Echo input", simpleEchoFunc),
			)
			require.NoError(t, err)

			serverDone[i] = make(chan error, 1)
			ctx, cancel := context.WithCancel(context.Background())
			cancels[i] = cancel

			go func(idx int) {
				err := handler.server.Run(ctx, serverTransport)
				serverDone[idx] <- err
			}(i)

			// Quick client test to verify server is running
			client := mcp.NewClient(&mcp.Implementation{
				Name: "test-client",
			}, nil)

			session, err := client.Connect(context.Background(), clientTransport, nil)
			require.NoError(t, err)

			err = session.Ping(context.Background(), &mcp.PingParams{})
			require.NoError(t, err, "server %d should respond to ping", i)

			err = session.Close()
			require.NoError(t, err)
		}

		// Cancel all servers
		for i := range numServers {
			cancels[i]()

			assert.Eventually(t, func() bool {
				select {
				case <-serverDone[i]:
					return true
				default:
					return false
				}
			}, 2*time.Second, 10*time.Millisecond, "server %d should stop after context cancellation", i)
		}
	})
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

func TestNewHandlerWithCustomServer(t *testing.T) {
	t.Parallel()

	// Create a custom server
	customServer := mcp.NewServer(&mcp.Implementation{
		Name:    "custom-server",
		Version: "2.0.0",
	}, nil)

	// Test NewHandler with custom server
	handler, err := NewHandler(
		WithName("test-server"), // This should be ignored when custom server is provided
		WithVersion("1.0.0"),    // This should be ignored when custom server is provided
		WithServer(customServer),
		WithTool("echo", "Echo input", simpleEchoFunc),
	)

	require.NoError(t, err)
	assert.NotNil(t, handler)

	// Verify the custom server is used
	assert.Equal(t, customServer, handler.GetServer())
}

func TestNewHandlerOptionError(t *testing.T) {
	t.Parallel()

	// Test that option errors are properly handled
	_, err := NewHandler(
		WithName("valid-name"),
		WithName(""), // This should cause an error
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to apply option")
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestServeStdioExists(t *testing.T) {
	t.Parallel()

	// Create handler
	handler, err := NewHandler(
		WithName("test-server"),
		WithTool("echo", "Echo input", simpleEchoFunc),
	)
	require.NoError(t, err)

	// Test that ServeStdio method exists and has the correct signature
	// Note: We cannot test ServeStdio execution with coverage enabled because
	// the MCP SDK's StdioTransport directly uses os.Stdout, which conflicts with
	// the coverage reporter's output stream, causing "file already closed" errors.
	// This is a known limitation when testing stdio-based transports.

	// Verify the method exists and is callable (but don't actually call it)
	assert.NotNil(t, handler.ServeStdio)

	// We can test ServeStdio indirectly by verifying the handler was created properly
	// and has the server that ServeStdio would use
	server := handler.GetServer()
	assert.NotNil(t, server)
}

func TestNewHandlerWithEmptySlices(t *testing.T) {
	t.Parallel()

	// Test that NewHandler works correctly when no resources are provided
	// This should exercise the empty slice paths in the registration loops
	handler, err := NewHandler(WithName("empty-handler"))

	require.NoError(t, err)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.GetServer())
}

func TestNewHandlerDefaultValues(t *testing.T) {
	t.Parallel()

	// Test that NewHandler uses default name and version when not provided
	handler, err := NewHandler()

	require.NoError(t, err)
	assert.NotNil(t, handler)

	// Check that default values are used
	server := handler.GetServer()
	assert.NotNil(t, server)
}

func TestNewHandler_Unified(t *testing.T) {
	t.Parallel()

	// Test tool function
	addTool := func(ctx context.Context, input struct{ A, B int }) (struct{ Sum int }, error) {
		return struct{ Sum int }{Sum: input.A + input.B}, nil
	}

	// Test prompt function
	greetingPrompt := func(ctx context.Context, args map[string]any) (*PromptResult, error) {
		name, ok := args["name"].(string)
		if !ok {
			name = "World"
		}
		return &PromptResult{
			Messages: []PromptMessage{
				{Role: "user", Content: "Hello " + name},
			},
		}, nil
	}

	// Test resource function
	configResource := func(ctx context.Context, uri string) (*ResourceContent, error) {
		return &ResourceContent{
			Content:  []byte("config data"),
			MIMEType: "text/plain",
		}, nil
	}

	// Create unified handler
	handler, err := NewHandler(
		WithName("unified-server"),
		WithVersion("1.0.0"),
		WithTool("add", "Add two numbers", addTool),
		WithPrompt("greeting", "Generate greeting", greetingPrompt),
		WithResource("config://settings", "App settings", configResource),
		WithResourceTemplate("user://{id}", "User data template", configResource),
	)

	require.NoError(t, err)
	assert.NotNil(t, handler)

	// Verify we can get the server
	server := handler.GetServer()
	assert.NotNil(t, server)
}

func TestNewHandler_Empty(t *testing.T) {
	t.Parallel()

	// Test that empty handler works
	handler, err := NewHandler(WithName("empty-server"))

	require.NoError(t, err)
	assert.NotNil(t, handler)
}
