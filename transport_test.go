package mcpio

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewInMemoryPair verifies that NewInMemoryPair creates a working handler and client transport
func TestNewInMemoryPair(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()

		// Define a simple tool using typed approach
		type EchoInput struct {
			Message string `json:"message"`
		}
		type EchoOutput struct {
			Echo string `json:"echo"`
		}

		toolCalled := false
		echoTool := func(ctx context.Context, reqCtx RequestContext, input EchoInput) (EchoOutput, error) {
			toolCalled = true
			return EchoOutput{Echo: input.Message}, nil
		}

		// Create handler and client transport
		handler, clientTransport, err := NewInMemoryPair(ctx,
			WithName("test-server"),
			WithVersion("1.0.0"),
			WithTool("echo", "Echo a message", echoTool),
		)
		require.NoError(t, err)
		require.NotNil(t, handler)
		require.NotNil(t, clientTransport)

		// Start server in background with WaitGroup
		var wg sync.WaitGroup
		wg.Go(func() {
			_ = handler.Run(ctx) //nolint:errcheck
		})

		// Wait for server goroutine to start
		synctest.Wait()

		// Create a client and connect using the client transport
		client := mcp.NewClient(&mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}, nil)

		session, err := client.Connect(ctx, clientTransport, nil)
		require.NoError(t, err)
		require.NotNil(t, session)

		// List tools to verify server is responding
		tools, err := session.ListTools(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, tools.Tools, 1)
		assert.Equal(t, "echo", tools.Tools[0].Name)

		// Call the tool to verify it works
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "echo",
			Arguments: map[string]any{
				"message": "test",
			},
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, toolCalled, "Tool should have been called")

		// Verify result content
		textContent, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		var resultContent map[string]any
		err = json.Unmarshal([]byte(textContent.Text), &resultContent)
		require.NoError(t, err)
		assert.Equal(t, "test", resultContent["echo"])
	})
}

// TestNewInMemoryPair_WithMultipleResources verifies NewInMemoryPair works with tools, prompts, and resources
func TestNewInMemoryPair_WithMultipleResources(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()

		type TestInput struct {
			Value string `json:"value"`
		}
		type TestOutput struct {
			Result string `json:"result"`
		}

		testTool := func(ctx context.Context, reqCtx RequestContext, input TestInput) (TestOutput, error) {
			return TestOutput{Result: "tool result"}, nil
		}

		testPrompt := func(ctx context.Context, reqCtx RequestContext, args map[string]any) (*PromptResult, error) {
			return &PromptResult{
				Messages: []PromptMessage{
					{Role: "user", Content: "test prompt"},
				},
			}, nil
		}

		testResource := func(ctx context.Context, reqCtx RequestContext) (*ResourceContent, error) {
			return &ResourceContent{
				Content:  []byte("test content"),
				MIMEType: "text/plain",
			}, nil
		}

		// Create handler with multiple resource types
		handler, clientTransport, err := NewInMemoryPair(ctx,
			WithTool("tool1", "Test tool", testTool),
			WithPrompt("prompt1", "Test prompt", testPrompt),
			WithResource("resource://test", "Test resource", testResource),
		)
		require.NoError(t, err)
		require.NotNil(t, handler)
		require.NotNil(t, clientTransport)

		// Start server
		var wg sync.WaitGroup
		wg.Go(func() {
			_ = handler.Run(ctx) //nolint:errcheck
		})

		synctest.Wait()

		// Connect client
		client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
		session, err := client.Connect(ctx, clientTransport, nil)
		require.NoError(t, err)

		// Verify all resources are available
		tools, err := session.ListTools(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, tools.Tools, 1)

		prompts, err := session.ListPrompts(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, prompts.Prompts, 1)

		resources, err := session.ListResources(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, resources.Resources, 1)

		// Actually test calling the tool to verify it works
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "tool1",
			Arguments: map[string]any{"value": "test"},
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)

		// Test calling the prompt
		promptResult, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
			Name: "prompt1",
		})
		require.NoError(t, err)
		assert.NotNil(t, promptResult)
		assert.Len(t, promptResult.Messages, 1)

		// Test reading the resource
		resourceResult, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
			URI: "resource://test",
		})
		require.NoError(t, err)
		assert.Len(t, resourceResult.Contents, 1)
		assert.Equal(t, "test content", resourceResult.Contents[0].Text)
	})
}

// TestNewInMemoryPair_ErrorHandling verifies error cases
func TestNewInMemoryPair_ErrorHandling(t *testing.T) {
	ctx := t.Context()

	// Test with invalid option
	handler, transport, err := NewInMemoryPair(ctx, WithName(""))
	require.Error(t, err)
	assert.Nil(t, handler)
	assert.Nil(t, transport)
}

// TestHandler_GetTransport verifies GetTransport method works
func TestHandler_GetTransport(t *testing.T) {
	ctx := t.Context()

	// Create handler using NewInMemoryPair
	handler, clientTransport, err := NewInMemoryPair(ctx, WithName("test"))
	require.NoError(t, err)
	require.NotNil(t, handler)
	require.NotNil(t, clientTransport)

	// Get transport from handler
	transport := handler.GetTransport()
	require.NotNil(t, transport)

	// Verify it's an InMemoryTransport
	_, ok := transport.(*mcp.InMemoryTransport)
	assert.True(t, ok, "Expected InMemoryTransport")
}

// TestHandler_GetTransport_NormalHandler verifies GetTransport on regular handler
func TestHandler_GetTransport_NormalHandler(t *testing.T) {
	handler, err := NewHandler(WithName("test"))
	require.NoError(t, err)
	require.NotNil(t, handler)

	transport := handler.GetTransport()
	assert.Nil(t, transport, "GetTransport returns nil when no explicit transport set; Run() defaults to StdioTransport")
}

// TestWithTransport_CustomTransport verifies WithTransport option works
func TestWithTransport_CustomTransport(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()

		// Create custom transports
		serverTransport, clientTransport := mcp.NewInMemoryTransports()

		// Create handler with custom transport
		handler, err := NewHandler(
			WithName("custom-transport-test"),
			WithTransport(serverTransport),
		)
		require.NoError(t, err)
		require.NotNil(t, handler)

		// Verify the transport was set
		transport := handler.GetTransport()
		assert.Equal(t, serverTransport, transport)

		// Start server
		var wg sync.WaitGroup
		wg.Go(func() {
			_ = handler.Run(ctx) //nolint:errcheck
		})

		synctest.Wait()

		// Connect client using the paired transport
		client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
		session, err := client.Connect(ctx, clientTransport, nil)
		require.NoError(t, err)
		require.NotNil(t, session)
	})
}

// TestWithTransport_NilTransport verifies error handling for nil transport
func TestWithTransport_NilTransport(t *testing.T) {
	handler, err := NewHandler(
		WithName("test"),
		WithTransport(nil),
	)
	require.Error(t, err)
	assert.Nil(t, handler)
	assert.ErrorIs(t, err, ErrNilValue, "Expected ErrNilValue")
}
