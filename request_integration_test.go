//go:build integration

package mcpio_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/robbyt/mcp-io/mcpwrapper"
	"github.com/stretchr/testify/require"
)

// Test that session is actually injected for typed tools (PR #1 bug fix)
func TestTypedToolSessionInjection(t *testing.T) {
	type Input struct {
		Text string `json:"text"`
	}
	type Output struct {
		SessionID string `json:"sessionId"`
	}

	handler, err := mcpio.NewHandler(
		mcpio.WithName("session-test"),
		mcpio.WithTool("typed_tool", "Test session injection", func(ctx context.Context, toolCtx mcpio.RequestContext, input Input) (Output, error) {
			session := toolCtx.GetSession()
			require.NotNil(t, session, "Session should be injected for typed tools")
			return Output{SessionID: session.SessionID()}, nil
		}),
	)
	require.NoError(t, err)

	// Connect client using testutil helper
	session := testutil.ConnectInMemory(t, handler)

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "typed_tool",
		Arguments: map[string]any{"text": "test"},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)
}

// Test that session is injected for raw tools (PR #1 bug fix)
func TestRawToolSessionInjection(t *testing.T) {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("session-test"),
		mcpio.WithRawTool("raw_tool", "Test session injection",
			`{"type":"object"}`,
			func(ctx context.Context, toolCtx mcpio.RequestContext, input []byte) ([]byte, error) {
				session := toolCtx.GetSession()
				require.NotNil(t, session, "Session should be injected for raw tools")
				return []byte(`{"session_id":"` + session.SessionID() + `"}`), nil
			},
		),
	)
	require.NoError(t, err)

	// Connect client using testutil helper
	session := testutil.ConnectInMemory(t, handler)

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "raw_tool",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)
}

// Test that session is injected for prompts (PR #1 bug fix)
func TestPromptSessionInjection(t *testing.T) {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("session-test"),
		mcpio.WithPrompt("test_prompt", "Test session injection",
			func(ctx context.Context, reqCtx mcpio.RequestContext, args map[string]any) (*mcpio.PromptResult, error) {
				session := reqCtx.GetSession()
				require.NotNil(t, session, "Session should be injected for prompts")
				return &mcpio.PromptResult{
					Messages: []mcpio.PromptMessage{{Role: "user", Content: "Session: " + session.SessionID()}},
				}, nil
			},
		),
	)
	require.NoError(t, err)

	// Connect client using testutil helper
	session := testutil.ConnectInMemory(t, handler)

	result, err := session.GetPrompt(t.Context(), &mcp.GetPromptParams{Name: "test_prompt"})
	require.NoError(t, err)
	require.Len(t, result.Messages, 1)
}

// Test that session is injected for typed prompts (PR #1 bug fix)
func TestTypedPromptSessionInjection(t *testing.T) {
	type PromptArgs struct {
		Topic string `json:"topic"`
	}

	handler, err := mcpio.NewHandler(
		mcpio.WithName("session-test"),
		mcpio.WithTypedPrompt("typed_prompt", "Test session injection",
			func(ctx context.Context, reqCtx mcpio.RequestContext, args PromptArgs) (*mcpio.PromptResult, error) {
				session := reqCtx.GetSession()
				require.NotNil(t, session, "Session should be injected for typed prompts")
				return &mcpio.PromptResult{
					Messages: []mcpio.PromptMessage{{Role: "user", Content: "Session: " + session.SessionID()}},
				}, nil
			},
		),
	)
	require.NoError(t, err)

	// Connect client using testutil helper
	session := testutil.ConnectInMemory(t, handler)

	result, err := session.GetPrompt(t.Context(), &mcp.GetPromptParams{
		Name:      "typed_prompt",
		Arguments: map[string]string{"topic": "test"},
	})
	require.NoError(t, err)
	require.Len(t, result.Messages, 1)
}

// Test that session is injected for resources (PR #1 bug fix)
func TestResourceSessionInjection(t *testing.T) {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("session-test"),
		mcpio.WithResource("test://resource", "Test session injection",
			func(ctx context.Context, reqCtx mcpio.RequestContext) (*mcpio.ResourceContent, error) {
				session := reqCtx.GetSession()
				require.NotNil(t, session, "Session should be injected for resources")
				return &mcpio.ResourceContent{
					Content:  []byte("Session: " + session.SessionID()),
					MIMEType: "text/plain",
				}, nil
			},
		),
	)
	require.NoError(t, err)

	// Connect client using testutil helper
	session := testutil.ConnectInMemory(t, handler)

	result, err := session.ReadResource(t.Context(), &mcp.ReadResourceParams{URI: "test://resource"})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
}

// Test RequestContext metadata fields (identifier, headers, token)
func TestRequestContextMetadata(t *testing.T) {
	type Output struct {
		Identifier  string `json:"identifier"`
		HasHeaders  bool   `json:"hasHeaders"`
		HeaderCount int    `json:"headerCount"`
		HasToken    bool   `json:"hasToken"`
		SessionID   string `json:"sessionId"`
	}

	handler, err := mcpio.NewHandler(
		mcpio.WithName("metadata-test"),
		mcpio.WithTool("metadata_tool", "Test RequestContext metadata", func(ctx context.Context, reqCtx mcpio.RequestContext, input struct{}) (Output, error) {
			headers := reqCtx.GetHeaders()
			require.NotNil(t, headers, "Headers should never be nil")

			return Output{
				Identifier:  reqCtx.GetIdentifier(),
				HasHeaders:  len(headers) > 0,
				HeaderCount: len(headers),
				HasToken:    reqCtx.GetTokenInfo() != nil,
				SessionID:   reqCtx.GetSession().SessionID(),
			}, nil
		}),
	)
	require.NoError(t, err)

	session := testutil.ConnectInMemory(t, handler)

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "metadata_tool",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)
}

// Test RequestContext with prompts to verify identifier
func TestPromptRequestContextIdentifier(t *testing.T) {
	var capturedIdentifier string

	handler, err := mcpio.NewHandler(
		mcpio.WithName("identifier-test"),
		mcpio.WithPrompt("test_prompt", "Test identifier",
			func(ctx context.Context, reqCtx mcpio.RequestContext, args map[string]any) (*mcpio.PromptResult, error) {
				capturedIdentifier = reqCtx.GetIdentifier()
				require.Equal(t, "test_prompt", capturedIdentifier, "Identifier should be prompt name")
				require.NotNil(t, reqCtx.GetHeaders(), "Headers should never be nil even for stdio")
				return &mcpio.PromptResult{
					Messages: []mcpio.PromptMessage{{Role: "user", Content: "OK"}},
				}, nil
			},
		),
	)
	require.NoError(t, err)

	session := testutil.ConnectInMemory(t, handler)

	result, err := session.GetPrompt(t.Context(), &mcp.GetPromptParams{Name: "test_prompt"})
	require.NoError(t, err)
	require.Len(t, result.Messages, 1)
	require.Equal(t, "test_prompt", capturedIdentifier)
}

// Test RequestContext with resources to verify identifier is URI
func TestResourceRequestContextIdentifier(t *testing.T) {
	var capturedIdentifier string
	const testURI = "test://my-resource/path"

	handler, err := mcpio.NewHandler(
		mcpio.WithName("resource-identifier-test"),
		mcpio.WithResource(testURI, "Test identifier",
			func(ctx context.Context, reqCtx mcpio.RequestContext) (*mcpio.ResourceContent, error) {
				capturedIdentifier = reqCtx.GetIdentifier()
				require.Equal(t, testURI, capturedIdentifier, "Identifier should be resource URI")
				require.NotNil(t, reqCtx.GetHeaders(), "Headers should never be nil")
				return &mcpio.ResourceContent{
					Content:  []byte("OK"),
					MIMEType: "text/plain",
				}, nil
			},
		),
	)
	require.NoError(t, err)

	session := testutil.ConnectInMemory(t, handler)

	result, err := session.ReadResource(t.Context(), &mcp.ReadResourceParams{URI: testURI})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
	require.Equal(t, testURI, capturedIdentifier)
}

// TestHTTPTransportRequestContextHeaders verifies that HTTP headers are correctly
// passed through RequestContext when using HTTP transport (covers extra != nil path)
func TestHTTPTransportRequestContextHeaders(t *testing.T) {
	type Output struct {
		HasHeaders    bool     `json:"hasHeaders"`
		CustomHeader  string   `json:"customHeader"`
		Authorization string   `json:"authorization"`
		HeaderKeys    []string `json:"headerKeys"`
	}

	var capturedHeaders http.Header

	handler, err := mcpio.NewHandler(
		mcpio.WithName("http-headers-test"),
		mcpio.WithHTTPTransport(
			mcpwrapper.WithStateless(), // Use stateless mode for simpler testing
		),
		mcpio.WithTool("header_tool", "Test HTTP headers in RequestContext", func(ctx context.Context, reqCtx mcpio.RequestContext, input struct{}) (Output, error) {
			headers := reqCtx.GetHeaders()
			capturedHeaders = headers
			require.NotNil(t, headers, "Headers should never be nil")

			var keys []string
			for k := range headers {
				keys = append(keys, k)
			}

			return Output{
				HasHeaders:    len(headers) > 0,
				CustomHeader:  headers.Get("X-Custom-Test"),
				Authorization: headers.Get("Authorization"),
				HeaderKeys:    keys,
			}, nil
		}),
	)
	require.NoError(t, err)

	// Create HTTP test server
	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer server.Close()

	// Create custom HTTP client with header injection
	customHeaders := http.Header{
		"X-Custom-Test": []string{"test-value-123"},
		"Authorization": []string{"Bearer test-token"},
	}

	client := &http.Client{
		Transport: &headerInjectTransport{
			base:    http.DefaultTransport,
			headers: customHeaders,
		},
	}

	// Create MCP client with HTTP transport
	transport := &mcp.StreamableClientTransport{
		Endpoint:   server.URL,
		HTTPClient: client,
	}

	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "test-http-client",
		Version: "1.0.0",
	}, nil)

	session, err := mcpClient.Connect(t.Context(), transport, nil)
	require.NoError(t, err)
	defer func() {
		if closeErr := session.Close(); closeErr != nil {
			t.Logf("session close error: %v", closeErr)
		}
	}()

	// Call the tool
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "header_tool",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	// Verify headers were captured
	require.NotNil(t, capturedHeaders, "Headers should have been captured")
	require.NotEmpty(t, capturedHeaders, "Should have received headers")
	require.Equal(t, "test-value-123", capturedHeaders.Get("X-Custom-Test"))
	require.Equal(t, "Bearer test-token", capturedHeaders.Get("Authorization"))
}

// headerInjectTransport is a custom http.RoundTripper that injects headers into requests
type headerInjectTransport struct {
	base    http.RoundTripper
	headers http.Header
}

func (t *headerInjectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request to avoid modifying the original
	req = req.Clone(req.Context())
	// Inject custom headers
	for k, v := range t.headers {
		for _, val := range v {
			req.Header.Add(k, val)
		}
	}
	return t.base.RoundTrip(req)
}
