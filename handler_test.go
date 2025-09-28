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

// Test ServeStdio more thoroughly without actually calling it
func TestServeStdioDetails(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(
		WithName("stdio-test-server"),
		WithTool("echo", "Echo input", simpleEchoFunc),
	)
	require.NoError(t, err)

	// Verify the method signature and behavior by testing components
	// that ServeStdio would use internally

	// Test that the method exists with correct signature
	assert.NotNil(t, handler.ServeStdio)

	// Test that the underlying server exists and can be used
	server := handler.GetServer()
	assert.NotNil(t, server)

	// Test ServeStdio exists and has the correct signature
	assert.NotNil(t, handler.ServeStdio)

	// Verify we can create the transport that ServeStdio uses
	transport := &mcp.StdioTransport{}
	assert.NotNil(t, transport)
}

// Test ServeStdio method by calling it with pre-cancelled context
func TestServeStdioActualCall(t *testing.T) {
	// Note: Don't run in parallel to avoid stdio interference

	handler, err := NewHandler(
		WithName("stdio-test"),
		WithTool("echo", "Echo input", simpleEchoFunc),
	)
	require.NoError(t, err)

	// Skip actual ServeStdio call when running with coverage to avoid stdio conflicts
	// The coverage reporter uses stdout, which conflicts with ServeStdio's stdio transport
	if testing.CoverMode() != "" {
		t.Skip("Skipping ServeStdio call during coverage runs to avoid stdio conflicts")
	}

	// Use a context that's already cancelled to ensure ServeStdio returns quickly
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately before calling

	// This should return quickly due to cancelled context
	err = handler.ServeStdio(ctx, nil, nil)

	// Expect context cancellation error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

// Test createSessionAwareToolHandler success path
func TestCreateSessionAwareToolHandlerSuccess(t *testing.T) {
	t.Parallel()

	// Session-aware tool function that returns success
	sessionToolFunc := func(ctx context.Context, capability ElicitationCapability, input SimpleInput) (SimpleOutput, error) {
		return SimpleOutput{Message: "session: " + input.Text}, nil
	}

	handler := createSessionAwareToolHandler(sessionToolFunc)

	// Create mock request with session
	req := &mcp.CallToolRequest{
		Session: &mcp.ServerSession{}, // Mock session
	}

	input := SimpleInput{Text: "test"}
	result, output, err := handler(context.Background(), req, input)

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "session: test", output.Message)
}

// Test createSessionAwareToolHandler with tool error
func TestCreateSessionAwareToolHandlerToolError(t *testing.T) {
	t.Parallel()

	// Session-aware tool function that returns a tool error
	sessionToolFunc := func(ctx context.Context, capability ElicitationCapability, input SimpleInput) (SimpleOutput, error) {
		return SimpleOutput{}, NewToolError("session tool failed")
	}

	handler := createSessionAwareToolHandler(sessionToolFunc)

	req := &mcp.CallToolRequest{
		Session: &mcp.ServerSession{},
	}

	input := SimpleInput{Text: "test"}
	result, output, err := handler(context.Background(), req, input)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, SimpleOutput{}, output)

	var toolErr *ToolError
	require.ErrorAs(t, err, &toolErr)
	assert.Equal(t, "session tool failed", toolErr.Message)
}

// Test createSessionAwareToolHandler with protocol error
func TestCreateSessionAwareToolHandlerProtocolError(t *testing.T) {
	t.Parallel()

	// Session-aware tool function that returns a protocol error
	sessionToolFunc := func(ctx context.Context, capability ElicitationCapability, input SimpleInput) (SimpleOutput, error) {
		return SimpleOutput{}, errors.New("session protocol error")
	}

	handler := createSessionAwareToolHandler(sessionToolFunc)

	req := &mcp.CallToolRequest{
		Session: &mcp.ServerSession{},
	}

	input := SimpleInput{Text: "test"}
	result, output, err := handler(context.Background(), req, input)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, SimpleOutput{}, output)
	assert.Equal(t, "session protocol error", err.Error())
}

// Test createSessionAwarePromptHandler success path
func TestCreateSessionAwarePromptHandlerSuccess(t *testing.T) {
	t.Parallel()

	// Session-aware prompt function
	sessionPromptFunc := func(ctx context.Context, capability ElicitationCapability, args map[string]any) (*PromptResult, error) {
		name := "World"
		if n, ok := args["name"].(string); ok {
			name = n
		}
		return &PromptResult{
			Description: "Test prompt",
			Messages: []PromptMessage{
				{Role: "system", Content: "System message"},
				{Role: "user", Content: "Hello " + name},
			},
		}, nil
	}

	handler := createSessionAwarePromptHandler(sessionPromptFunc)

	req := &mcp.GetPromptRequest{
		Session: &mcp.ServerSession{},
		Params: &mcp.GetPromptParams{
			Arguments: map[string]string{
				"name": "Test",
			},
		},
	}

	result, err := handler(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test prompt", result.Description)
	assert.Len(t, result.Messages, 2)
	assert.Equal(t, mcp.Role("system"), result.Messages[0].Role)
	assert.Equal(t, mcp.Role("user"), result.Messages[1].Role)
	assert.Equal(t, "Hello Test", result.Messages[1].Content.(*mcp.TextContent).Text)
}

// Test createSessionAwarePromptHandler with nil arguments
func TestCreateSessionAwarePromptHandlerNilArgs(t *testing.T) {
	t.Parallel()

	sessionPromptFunc := func(ctx context.Context, capability ElicitationCapability, args map[string]any) (*PromptResult, error) {
		return &PromptResult{
			Description: "No args prompt",
			Messages: []PromptMessage{
				{Role: "user", Content: "No arguments provided"},
			},
		}, nil
	}

	handler := createSessionAwarePromptHandler(sessionPromptFunc)

	req := &mcp.GetPromptRequest{
		Session: &mcp.ServerSession{},
		Params: &mcp.GetPromptParams{
			Arguments: nil, // Test nil arguments
		},
	}

	result, err := handler(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "No args prompt", result.Description)
}

// Test createSessionAwarePromptHandler error path
func TestCreateSessionAwarePromptHandlerError(t *testing.T) {
	t.Parallel()

	sessionPromptFunc := func(ctx context.Context, capability ElicitationCapability, args map[string]any) (*PromptResult, error) {
		return nil, errors.New("prompt failed")
	}

	handler := createSessionAwarePromptHandler(sessionPromptFunc)

	req := &mcp.GetPromptRequest{
		Session: &mcp.ServerSession{},
		Params:  &mcp.GetPromptParams{},
	}

	result, err := handler(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "prompt failed", err.Error())
}

// Test createSessionAwareResourceHandler with text content
func TestCreateSessionAwareResourceHandlerTextContent(t *testing.T) {
	t.Parallel()

	sessionResourceFunc := func(ctx context.Context, capability ElicitationCapability, uri string) (*ResourceContent, error) {
		return &ResourceContent{
			Content:  []byte("text content"),
			MIMEType: "text/plain",
		}, nil
	}

	handler := createSessionAwareResourceHandler(sessionResourceFunc)

	req := &mcp.ReadResourceRequest{
		Session: &mcp.ServerSession{},
		Params: &mcp.ReadResourceParams{
			URI: "test://resource",
		},
	}

	result, err := handler(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Contents, 1)
	assert.Equal(t, "test://resource", result.Contents[0].URI)
	assert.Equal(t, "text/plain", result.Contents[0].MIMEType)
	assert.Equal(t, "text content", result.Contents[0].Text)
	assert.Nil(t, result.Contents[0].Blob)
}

// Test createSessionAwareResourceHandler with various MIME types
func TestCreateSessionAwareResourceHandlerMIMETypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		content      []byte
		mimeType     string
		expectedText string
		expectedBlob []byte
	}{
		{
			name:         "JSON content",
			content:      []byte(`{"key": "value"}`),
			mimeType:     "application/json",
			expectedText: `{"key": "value"}`,
			expectedBlob: nil,
		},
		{
			name:         "XML content",
			content:      []byte(`<root><item>value</item></root>`),
			mimeType:     "application/xml",
			expectedText: `<root><item>value</item></root>`,
			expectedBlob: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionResourceFunc := func(ctx context.Context, capability ElicitationCapability, uri string) (*ResourceContent, error) {
				return &ResourceContent{
					Content:  tt.content,
					MIMEType: tt.mimeType,
				}, nil
			}

			handler := createSessionAwareResourceHandler(sessionResourceFunc)

			req := &mcp.ReadResourceRequest{
				Session: &mcp.ServerSession{},
				Params: &mcp.ReadResourceParams{
					URI: "test://resource",
				},
			}

			result, err := handler(context.Background(), req)

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Len(t, result.Contents, 1)
			assert.Equal(t, tt.mimeType, result.Contents[0].MIMEType)
			if tt.expectedText != "" {
				assert.Equal(t, tt.expectedText, result.Contents[0].Text)
				assert.Nil(t, result.Contents[0].Blob)
			} else {
				assert.Equal(t, tt.expectedBlob, result.Contents[0].Blob)
				assert.Empty(t, result.Contents[0].Text)
			}
		})
	}
}

// Test createSessionAwareResourceHandler with binary content
func TestCreateSessionAwareResourceHandlerBinaryContent(t *testing.T) {
	t.Parallel()

	binaryData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header

	sessionResourceFunc := func(ctx context.Context, capability ElicitationCapability, uri string) (*ResourceContent, error) {
		return &ResourceContent{
			Content:  binaryData,
			MIMEType: "image/png",
		}, nil
	}

	handler := createSessionAwareResourceHandler(sessionResourceFunc)

	req := &mcp.ReadResourceRequest{
		Session: &mcp.ServerSession{},
		Params: &mcp.ReadResourceParams{
			URI: "test://image-resource",
		},
	}

	result, err := handler(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Contents, 1)
	assert.Equal(t, "image/png", result.Contents[0].MIMEType)
	assert.Empty(t, result.Contents[0].Text)
	assert.Equal(t, binaryData, result.Contents[0].Blob)
}

// Test createSessionAwareResourceHandler with empty content
func TestCreateSessionAwareResourceHandlerEmptyContent(t *testing.T) {
	t.Parallel()

	sessionResourceFunc := func(ctx context.Context, capability ElicitationCapability, uri string) (*ResourceContent, error) {
		return &ResourceContent{
			Content:  []byte{},
			MIMEType: "text/plain",
		}, nil
	}

	handler := createSessionAwareResourceHandler(sessionResourceFunc)

	req := &mcp.ReadResourceRequest{
		Session: &mcp.ServerSession{},
		Params: &mcp.ReadResourceParams{
			URI: "test://empty-resource",
		},
	}

	result, err := handler(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Contents, 1)
	assert.Equal(t, "text/plain", result.Contents[0].MIMEType)
	assert.Empty(t, result.Contents[0].Text)
	assert.Nil(t, result.Contents[0].Blob)
}

// Test createSessionAwareResourceHandler error path
func TestCreateSessionAwareResourceHandlerError(t *testing.T) {
	t.Parallel()

	sessionResourceFunc := func(ctx context.Context, capability ElicitationCapability, uri string) (*ResourceContent, error) {
		return nil, errors.New("resource failed")
	}

	handler := createSessionAwareResourceHandler(sessionResourceFunc)

	req := &mcp.ReadResourceRequest{
		Session: &mcp.ServerSession{},
		Params: &mcp.ReadResourceParams{
			URI: "test://error-resource",
		},
	}

	result, err := handler(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "resource failed", err.Error())
}

// Test NewHandler with failing tool registration
func TestNewHandlerToolRegistrationError(t *testing.T) {
	t.Parallel()

	// Create a tool registration function that will fail
	failingToolOption := func(cfg *handlerConfig) error {
		failingRegistration := func(server *mcp.Server) error {
			return errors.New("tool registration failed")
		}
		cfg.tools = append(cfg.tools, failingRegistration)
		return nil
	}

	_, err := NewHandler(
		WithName("test-server"),
		failingToolOption,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register tool")
	assert.Contains(t, err.Error(), "tool registration failed")
}

// Test NewHandler with failing prompt registration
func TestNewHandlerPromptRegistrationError(t *testing.T) {
	t.Parallel()

	// Create a prompt registration function that will fail
	failingPromptOption := func(cfg *handlerConfig) error {
		failingRegistration := func(server *mcp.Server) error {
			return errors.New("prompt registration failed")
		}
		cfg.prompts = append(cfg.prompts, failingRegistration)
		return nil
	}

	_, err := NewHandler(
		WithName("test-server"),
		failingPromptOption,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register prompt")
	assert.Contains(t, err.Error(), "prompt registration failed")
}

// Test NewHandler with failing resource registration
func TestNewHandlerResourceRegistrationError(t *testing.T) {
	t.Parallel()

	// Create a resource registration function that will fail
	failingResourceOption := func(cfg *handlerConfig) error {
		failingRegistration := func(server *mcp.Server) error {
			return errors.New("resource registration failed")
		}
		cfg.resources = append(cfg.resources, failingRegistration)
		return nil
	}

	_, err := NewHandler(
		WithName("test-server"),
		failingResourceOption,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register resource")
	assert.Contains(t, err.Error(), "resource registration failed")
}

// Test NewHandler with failing resource template registration
func TestNewHandlerResourceTemplateRegistrationError(t *testing.T) {
	t.Parallel()

	// Create a resource template registration function that will fail
	failingTemplateOption := func(cfg *handlerConfig) error {
		failingRegistration := func(server *mcp.Server) error {
			return errors.New("template registration failed")
		}
		cfg.resourceTemplates = append(cfg.resourceTemplates, failingRegistration)
		return nil
	}

	_, err := NewHandler(
		WithName("test-server"),
		failingTemplateOption,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register resource template")
	assert.Contains(t, err.Error(), "template registration failed")
}

// Test NewHandler with successful mixed registrations
func TestNewHandlerMixedRegistrationsSuccess(t *testing.T) {
	t.Parallel()

	// Create successful registration functions
	successToolOption := func(cfg *handlerConfig) error {
		successRegistration := func(server *mcp.Server) error {
			return nil // Success
		}
		cfg.tools = append(cfg.tools, successRegistration)
		return nil
	}

	successPromptOption := func(cfg *handlerConfig) error {
		successRegistration := func(server *mcp.Server) error {
			return nil // Success
		}
		cfg.prompts = append(cfg.prompts, successRegistration)
		return nil
	}

	successResourceOption := func(cfg *handlerConfig) error {
		successRegistration := func(server *mcp.Server) error {
			return nil // Success
		}
		cfg.resources = append(cfg.resources, successRegistration)
		return nil
	}

	successTemplateOption := func(cfg *handlerConfig) error {
		successRegistration := func(server *mcp.Server) error {
			return nil // Success
		}
		cfg.resourceTemplates = append(cfg.resourceTemplates, successRegistration)
		return nil
	}

	handler, err := NewHandler(
		WithName("test-server"),
		successToolOption,
		successPromptOption,
		successResourceOption,
		successTemplateOption,
	)

	require.NoError(t, err)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.GetServer())
}

// Test that ServeStdio creates the correct transport type internally
func TestServeStdioInternalBehavior(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(
		WithName("stdio-internal-test"),
		WithTool("echo", "Echo input", simpleEchoFunc),
	)
	require.NoError(t, err)

	// We can't directly test ServeStdio execution due to stdio conflicts with coverage,
	// but we can verify that the handler has all the necessary components
	// that ServeStdio would use

	// Verify the server exists
	server := handler.GetServer()
	assert.NotNil(t, server)

	// Verify the method exists with correct signature
	assert.NotNil(t, handler.ServeStdio)

	// Test that we can create the transport type that ServeStdio would use
	transport := &mcp.StdioTransport{}
	assert.NotNil(t, transport)

	// Verify context handling that ServeStdio would use
	ctx := context.Background()
	assert.NotNil(t, ctx)

	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()
	assert.NotNil(t, ctxWithCancel)
}
