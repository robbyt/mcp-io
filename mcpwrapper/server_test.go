package mcpwrapper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("creates server with defaults", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		server := New(sdkServer)

		assert.NotNil(t, server)
		assert.NotNil(t, server.server)
		assert.Nil(t, server.transport)
		assert.NotNil(t, server.httpOpts)
		assert.False(t, server.httpOpts.Stateless)
		assert.False(t, server.httpOpts.JSONResponse)
	})

	t.Run("applies WithHTTPOptions", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		customOpts := &mcpSDK.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true,
		}

		server := New(sdkServer, WithHTTPOptions(customOpts))

		assert.NotNil(t, server)
		assert.NotNil(t, server.httpOpts)
		assert.True(t, server.httpOpts.Stateless)
		assert.True(t, server.httpOpts.JSONResponse)
	})

	t.Run("applies multiple options", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		opts1 := &mcpSDK.StreamableHTTPOptions{
			Stateless: true,
		}
		opts2 := &mcpSDK.StreamableHTTPOptions{
			JSONResponse: true,
		}

		server := New(sdkServer, WithHTTPOptions(opts1), WithHTTPOptions(opts2))

		assert.NotNil(t, server)
		assert.True(t, server.httpOpts.JSONResponse)
	})
}

func TestWithHTTPOptions(t *testing.T) {
	t.Run("sets custom HTTP options", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		customOpts := &mcpSDK.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true,
		}

		server := New(sdkServer, WithHTTPOptions(customOpts))

		assert.Equal(t, customOpts, server.httpOpts)
	})
}

func TestGetTransport(t *testing.T) {
	t.Run("returns nil when no transport set", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		server := New(sdkServer)

		transport := server.GetTransport()
		assert.Nil(t, transport)
	})

	t.Run("returns transport when set with WithTransport", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		customTransport := &mcpSDK.StdioTransport{}
		server := New(sdkServer, WithTransport(customTransport))

		transport := server.GetTransport()
		assert.NotNil(t, transport)
		assert.Equal(t, customTransport, transport)
	})

	t.Run("returns in-memory transport for NewInMemoryServer", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		server, _ := NewInMemoryServer(sdkServer)

		transport := server.GetTransport()
		assert.NotNil(t, transport)
		_, ok := transport.(*mcpSDK.InMemoryTransport)
		assert.True(t, ok)
	})
}

func TestWithTransport(t *testing.T) {
	t.Run("sets custom transport", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		customTransport := &mcpSDK.StdioTransport{}
		server := New(sdkServer, WithTransport(customTransport))

		assert.Equal(t, customTransport, server.GetTransport())
	})

	t.Run("nil transport is accepted", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		server := New(sdkServer, WithTransport(nil))

		assert.Nil(t, server.GetTransport())
	})
}

func TestNewInMemoryServer(t *testing.T) {
	t.Run("creates server with in-memory transport", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		server, clientTransport := NewInMemoryServer(sdkServer)

		assert.NotNil(t, server)
		assert.NotNil(t, server.server)
		assert.NotNil(t, server.transport)
		assert.NotNil(t, clientTransport)
		assert.NotNil(t, server.httpOpts)
		assert.False(t, server.httpOpts.Stateless)
		assert.False(t, server.httpOpts.JSONResponse)
	})

	t.Run("applies WithHTTPOptions", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		customOpts := &mcpSDK.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true,
		}

		server, clientTransport := NewInMemoryServer(sdkServer, WithHTTPOptions(customOpts))

		assert.NotNil(t, server)
		assert.NotNil(t, clientTransport)
		assert.True(t, server.httpOpts.Stateless)
		assert.True(t, server.httpOpts.JSONResponse)
	})
}

func TestAddTool(t *testing.T) {
	t.Run("registers tool with server", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})
		server := New(sdkServer)

		tool := &mcpSDK.Tool{
			Name:        "test-tool",
			Description: "A test tool",
			InputSchema: map[string]any{
				"type": "object",
			},
		}
		handler := func(ctx context.Context, req *mcpSDK.CallToolRequest) (*mcpSDK.CallToolResult, error) {
			return &mcpSDK.CallToolResult{
				Content: []mcpSDK.Content{
					&mcpSDK.TextContent{Text: "test result"},
				},
			}, nil
		}

		server.AddTool(tool, handler)
	})
}

func TestAddPrompt(t *testing.T) {
	t.Run("registers prompt with server", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})
		server := New(sdkServer)

		prompt := &mcpSDK.Prompt{
			Name:        "test-prompt",
			Description: "A test prompt",
		}
		handler := func(ctx context.Context, req *mcpSDK.GetPromptRequest) (*mcpSDK.GetPromptResult, error) {
			return &mcpSDK.GetPromptResult{
				Messages: []*mcpSDK.PromptMessage{
					{
						Role: "user",
						Content: mcpSDK.Content(&mcpSDK.TextContent{
							Text: "test prompt",
						}),
					},
				},
			}, nil
		}

		server.AddPrompt(prompt, handler)
	})
}

func TestAddResource(t *testing.T) {
	t.Run("registers resource with server", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})
		server := New(sdkServer)

		resource := &mcpSDK.Resource{
			URI:         "test://resource",
			Name:        "test-resource",
			Description: "A test resource",
		}
		handler := func(ctx context.Context, req *mcpSDK.ReadResourceRequest) (*mcpSDK.ReadResourceResult, error) {
			return &mcpSDK.ReadResourceResult{
				Contents: []*mcpSDK.ResourceContents{
					{
						URI:  "test://resource",
						Text: "test content",
					},
				},
			}, nil
		}

		server.AddResource(resource, handler)
	})
}

func TestAddResourceTemplate(t *testing.T) {
	t.Run("registers resource template with server", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})
		server := New(sdkServer)

		template := &mcpSDK.ResourceTemplate{
			URITemplate: "test://{id}",
			Name:        "test-template",
			Description: "A test template",
		}
		handler := func(ctx context.Context, req *mcpSDK.ReadResourceRequest) (*mcpSDK.ReadResourceResult, error) {
			return &mcpSDK.ReadResourceResult{
				Contents: []*mcpSDK.ResourceContents{
					{
						URI:  req.Params.URI,
						Text: "test content",
					},
				},
			}, nil
		}

		server.AddResourceTemplate(template, handler)
	})
}

func TestUnwrap(t *testing.T) {
	t.Run("returns underlying SDK server", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})
		server := New(sdkServer)

		unwrapped := server.Unwrap()

		assert.NotNil(t, unwrapped)
		assert.Equal(t, sdkServer, unwrapped)
	})
}

func TestRun(t *testing.T) {
	t.Run("runs server with transport", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		server, clientTransport := NewInMemoryServer(sdkServer)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		errChan := make(chan error, 1)
		go func() {
			errChan <- server.Run(ctx)
		}()

		client := mcpSDK.NewClient(impl, &mcpSDK.ClientOptions{})
		_, err := client.Connect(ctx, clientTransport, nil)
		require.NoError(t, err)

		cancel()

		err = <-errChan
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

func TestServeHTTP(t *testing.T) {
	t.Run("handles HTTP request with default options", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})
		server := New(sdkServer)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		assert.NotPanics(t, func() {
			server.ServeHTTP(rec, req)
		})

		assert.NotEqual(t, 0, rec.Code)
	})

	t.Run("handles HTTP request with custom options", func(t *testing.T) {
		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		customOpts := &mcpSDK.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true,
		}
		server := New(sdkServer, WithHTTPOptions(customOpts))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		assert.NotPanics(t, func() {
			server.ServeHTTP(rec, req)
		})

		assert.NotEqual(t, 0, rec.Code)
	})
}
