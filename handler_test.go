package mcpio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/mcpwrapper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
func simpleEchoFunc(ctx context.Context, toolCtx RequestContext, input SimpleInput) (SimpleOutput, error) {
	return SimpleOutput{Message: input.Text}, nil
}

// mockMCPServer is a testify/mock implementation of the MCPServer interface,
// used to verify that Handler delegates correctly to its underlying server
// without spinning up a real MCP transport.
type mockMCPServer struct {
	mock.Mock
}

func newMockMCPServer() *mockMCPServer {
	return &mockMCPServer{}
}

func (m *mockMCPServer) AddTool(tool *mcp.Tool, handler mcp.ToolHandler) {
	m.Called(tool, handler)
}

func (m *mockMCPServer) AddPrompt(prompt *mcp.Prompt, handler mcp.PromptHandler) {
	m.Called(prompt, handler)
}

func (m *mockMCPServer) AddResource(resource *mcp.Resource, handler mcp.ResourceHandler) {
	m.Called(resource, handler)
}

func (m *mockMCPServer) AddResourceTemplate(
	template *mcp.ResourceTemplate,
	handler mcp.ResourceHandler,
) {
	m.Called(template, handler)
}

func (m *mockMCPServer) Run(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *mockMCPServer) GetTransport() mcp.Transport {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(mcp.Transport)
}

func (m *mockMCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

func (m *mockMCPServer) Unwrap() any {
	return m.Called().Get(0)
}

func TestServeHTTP(t *testing.T) {
	t.Parallel()
	// Create handler with real server for HTTP testing
	handler, err := NewHandler(
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
		handler, err := NewHandler(
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
		handler, err := NewHandler(
			WithName("test-server"),
			WithTool("echo", "Echo input", simpleEchoFunc),
		)
		require.NoError(t, err)

		// Create in-memory server with transport
		sdkServer, ok := handler.server.Unwrap().(*mcp.Server)
		require.True(t, ok, "failed to unwrap SDK server")
		wrappedServer, clientTransport := mcpwrapper.NewInMemoryServer(sdkServer)

		// Run server with cancellable context
		ctx, cancel := context.WithCancel(t.Context())
		serverDone := make(chan error, 1)

		go func() {
			err := wrappedServer.Run(ctx)
			serverDone <- err
		}()

		// Create client and test
		client := mcp.NewClient(&mcp.Implementation{
			Name: "test-client",
		}, nil)

		session, err := client.Connect(t.Context(), clientTransport, nil)
		require.NoError(t, err)

		// Test tool execution
		result, err := session.CallTool(t.Context(), &mcp.CallToolParams{
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
			handler, err := NewHandler(
				WithName("test-server"),
				WithTool("echo", "Echo input", simpleEchoFunc),
			)
			require.NoError(t, err)

			// Create in-memory server with transport
			sdkServer, ok := handler.server.Unwrap().(*mcp.Server)
			require.True(t, ok, "failed to unwrap SDK server")
			wrappedServer, clientTransport := mcpwrapper.NewInMemoryServer(sdkServer)

			serverDone[i] = make(chan error, 1)
			ctx, cancel := context.WithCancel(t.Context())
			cancels[i] = cancel

			go func(idx int) {
				err := wrappedServer.Run(ctx)
				serverDone[idx] <- err
			}(i)

			// Quick client test to verify server is running
			client := mcp.NewClient(&mcp.Implementation{
				Name: "test-client",
			}, nil)

			session, err := client.Connect(t.Context(), clientTransport, nil)
			require.NoError(t, err)

			err = session.Ping(t.Context(), &mcp.PingParams{})
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

	handler, err := NewHandler(WithServer(server))
	require.NoError(t, err)

	retrievedServer := handler.GetServer()
	assert.Equal(t, server, retrievedServer.Unwrap())
}

func TestServeHTTP_BasicResponse(t *testing.T) {
	t.Parallel()
	handler, err := NewHandler(
		WithName("test-server"),
		WithTool("echo", "Echo input", simpleEchoFunc),
	)
	require.NoError(t, err)

	// Create test server using httptest
	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer server.Close()

	// Test basic Streamable HTTP response
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

func TestServeSSEDelegatesToServeHTTP(t *testing.T) {
	t.Parallel()

	mockServer := newMockMCPServer()
	handler := &Handler{server: mockServer}
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()

	mockServer.On("ServeHTTP", rec, req).Run(func(args mock.Arguments) {
		w := args.Get(0).(http.ResponseWriter)
		w.Header().Set("X-Test-Handler", "serve-http")
		w.WriteHeader(http.StatusAccepted)
		_, err := w.Write([]byte("served"))
		require.NoError(t, err)
	}).Return()

	handler.ServeSSE(rec, req)

	mockServer.AssertNumberOfCalls(t, "ServeHTTP", 1)
	mockServer.AssertCalled(t, "ServeHTTP", rec, req)
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Equal(t, "serve-http", rec.Header().Get("X-Test-Handler"))
	assert.Equal(t, "served", rec.Body.String())
	mockServer.AssertExpectations(t)
}

func TestHandler_ServeHTTP_DelegatesToServer(t *testing.T) {
	t.Parallel()

	mockServer := newMockMCPServer()
	handler := &Handler{server: mockServer}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	mockServer.On("ServeHTTP", rec, req).Run(func(args mock.Arguments) {
		args.Get(0).(http.ResponseWriter).WriteHeader(http.StatusTeapot)
	}).Return()

	handler.ServeHTTP(rec, req)

	mockServer.AssertCalled(t, "ServeHTTP", rec, req)
	mockServer.AssertNumberOfCalls(t, "ServeHTTP", 1)
	assert.Equal(t, http.StatusTeapot, rec.Code)
	mockServer.AssertExpectations(t)
}

func TestHandler_Run_DelegatesToServer(t *testing.T) {
	t.Parallel()

	t.Run("returns error from underlying server", func(t *testing.T) {
		t.Parallel()
		mockServer := newMockMCPServer()
		handler := &Handler{server: mockServer}
		ctx := t.Context()
		sentinel := errors.New("run failed")

		mockServer.On("Run", ctx).Return(sentinel)

		err := handler.Run(ctx)

		require.ErrorIs(t, err, sentinel)
		mockServer.AssertCalled(t, "Run", ctx)
		mockServer.AssertExpectations(t)
	})

	t.Run("returns nil on success", func(t *testing.T) {
		t.Parallel()
		mockServer := newMockMCPServer()
		handler := &Handler{server: mockServer}
		ctx := t.Context()

		mockServer.On("Run", ctx).Return(nil)

		require.NoError(t, handler.Run(ctx))
		mockServer.AssertExpectations(t)
	})
}

func TestHandler_ServeStdio_DelegatesToRun(t *testing.T) {
	t.Parallel()

	mockServer := newMockMCPServer()
	handler := &Handler{server: mockServer}
	ctx := t.Context()

	mockServer.On("Run", ctx).Return(nil)

	require.NoError(t, handler.ServeStdio(ctx))

	mockServer.AssertCalled(t, "Run", ctx)
	mockServer.AssertNumberOfCalls(t, "Run", 1)
	mockServer.AssertExpectations(t)
}

func TestHandler_GetTransport_DelegatesToServer(t *testing.T) {
	t.Parallel()

	t.Run("returns transport from server", func(t *testing.T) {
		t.Parallel()
		mockServer := newMockMCPServer()
		handler := &Handler{server: mockServer}
		transport := &mcp.StdioTransport{}

		mockServer.On("GetTransport").Return(transport)

		assert.Same(t, transport, handler.GetTransport())
		mockServer.AssertExpectations(t)
	})

	t.Run("returns nil when server has no transport", func(t *testing.T) {
		t.Parallel()
		mockServer := newMockMCPServer()
		handler := &Handler{server: mockServer}

		mockServer.On("GetTransport").Return(nil)

		assert.Nil(t, handler.GetTransport())
		mockServer.AssertExpectations(t)
	})
}

func TestServeHTTP_StreamableClientLifecycle(t *testing.T) {
	type EchoInput struct {
		Text string `json:"text" jsonschema:"Text to echo"`
	}
	type EchoOutput struct {
		Message string `json:"message" jsonschema:"Echoed message"`
	}

	handler, err := NewHandler(
		WithName("streamable-test-server"),
		WithTool("echo", "Echo input", func(ctx context.Context, reqCtx RequestContext, input EchoInput) (EchoOutput, error) {
			return EchoOutput{Message: input.Text}, nil
		}),
		WithTool("fail", "Return a tool error", func(ctx context.Context, reqCtx RequestContext, input struct{}) (EchoOutput, error) {
			return EchoOutput{}, ValidationError("bad input")
		}),
		WithPrompt("prompt1", "Test prompt", func(ctx context.Context, reqCtx RequestContext, args map[string]any) (*PromptResult, error) {
			return &PromptResult{
				Messages: []PromptMessage{{Role: "user", Content: "test prompt"}},
			}, nil
		}),
		WithResource("resource://test", "Test resource", func(ctx context.Context, reqCtx RequestContext) (*ResourceContent, error) {
			return &ResourceContent{
				Content:  []byte("test content"),
				MIMEType: "text/plain",
			}, nil
		}),
	)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer server.Close()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "streamable-test-client",
		Version: "1.0.0",
	}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint:   server.URL,
		HTTPClient: server.Client(),
	}, nil)
	require.NoError(t, err)
	defer func() {
		err := session.Close()
		require.NoError(t, err)
	}()

	tools, err := session.ListTools(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, tools.Tools, 2)

	toolResult, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "echo",
		Arguments: map[string]any{"text": "hello"},
	})
	require.NoError(t, err)
	require.False(t, toolResult.IsError)
	require.NotEmpty(t, toolResult.Content)

	errorResult, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "fail",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.True(t, errorResult.IsError)
	require.NotEmpty(t, errorResult.Content)
	errorText, ok := errorResult.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	require.Contains(t, errorText.Text, "bad input")

	prompts, err := session.ListPrompts(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, prompts.Prompts, 1)

	promptResult, err := session.GetPrompt(t.Context(), &mcp.GetPromptParams{Name: "prompt1"})
	require.NoError(t, err)
	require.Len(t, promptResult.Messages, 1)

	resources, err := session.ListResources(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, resources.Resources, 1)

	resourceResult, err := session.ReadResource(t.Context(), &mcp.ReadResourceParams{URI: "resource://test"})
	require.NoError(t, err)
	require.Len(t, resourceResult.Contents, 1)
	require.Equal(t, "test content", resourceResult.Contents[0].Text)

	_, err = session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "missing",
		Arguments: map[string]any{},
	})
	require.Error(t, err)
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
	assert.Equal(t, customServer, handler.GetServer().Unwrap())
}

func TestNewHandlerOptionError(t *testing.T) {
	t.Parallel()

	// Test that option errors are properly handled
	_, err := NewHandler(
		WithName("valid-name"),
		WithName(""), // This should cause an error
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyValue)
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
	addTool := func(ctx context.Context, toolCtx RequestContext, input struct{ A, B int }) (struct{ Sum int }, error) {
		return struct{ Sum int }{Sum: input.A + input.B}, nil
	}

	// Test prompt function
	greetingPrompt := func(ctx context.Context, reqCtx RequestContext, args map[string]any) (*PromptResult, error) {
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
	configResource := func(ctx context.Context, reqCtx RequestContext) (*ResourceContent, error) {
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
// Test NewHandler with failing tool registration
func TestNewHandlerToolRegistrationError(t *testing.T) {
	t.Parallel()
	errToolRegistration := errors.New("tool registration failed")

	// Create a tool registration function that will fail
	failingToolOption := func(cfg *handlerConfig) error {
		failingRegistration := func(server *mcp.Server) error {
			return errToolRegistration
		}
		cfg.tools = append(cfg.tools, failingRegistration)
		return nil
	}

	_, err := NewHandler(
		WithName("test-server"),
		failingToolOption,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, errToolRegistration)
}

// Test NewHandler with failing prompt registration
func TestNewHandlerPromptRegistrationError(t *testing.T) {
	t.Parallel()
	errPromptRegistration := errors.New("prompt registration failed")

	// Create a prompt registration function that will fail
	failingPromptOption := func(cfg *handlerConfig) error {
		failingRegistration := func(server *mcp.Server) error {
			return errPromptRegistration
		}
		cfg.prompts = append(cfg.prompts, failingRegistration)
		return nil
	}

	_, err := NewHandler(
		WithName("test-server"),
		failingPromptOption,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, errPromptRegistration)
}

// Test NewHandler with failing resource registration
func TestNewHandlerResourceRegistrationError(t *testing.T) {
	t.Parallel()
	errResourceRegistration := errors.New("resource registration failed")

	// Create a resource registration function that will fail
	failingResourceOption := func(cfg *handlerConfig) error {
		failingRegistration := func(server *mcp.Server) error {
			return errResourceRegistration
		}
		cfg.resources = append(cfg.resources, failingRegistration)
		return nil
	}

	_, err := NewHandler(
		WithName("test-server"),
		failingResourceOption,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, errResourceRegistration)
}

// Test NewHandler with failing resource template registration
func TestNewHandlerResourceTemplateRegistrationError(t *testing.T) {
	t.Parallel()
	errResourceTemplateRegistration := errors.New("resource template registration failed")

	// Create a resource template registration function that will fail
	failingTemplateOption := func(cfg *handlerConfig) error {
		failingRegistration := func(server *mcp.Server) error {
			return errResourceTemplateRegistration
		}
		cfg.resourceTemplates = append(cfg.resourceTemplates, failingRegistration)
		return nil
	}

	_, err := NewHandler(
		WithName("test-server"),
		failingTemplateOption,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, errResourceTemplateRegistration)
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
			handler, err := NewHandler(tt.opts...)

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

func TestToolRegistration(t *testing.T) {
	t.Parallel()
	// Use real MCP server instead of mock
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	handler, err := NewHandler(
		WithServer(server),
		WithTool("greet", "Greet someone", greetFunc),
		WithTool("farewell", "Say goodbye", farewellFunc),
	)

	require.NoError(t, err)
	assert.NotNil(t, handler)
	assert.Equal(t, server, handler.GetServer().Unwrap())
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()
	handler, err := NewHandler(
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
