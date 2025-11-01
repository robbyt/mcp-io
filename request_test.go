package mcpio_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: newRequestContext integration tests are in request_integration_test.go
// These unit tests focus on the getter methods and struct behavior

func TestMCPRequestContext_GetIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
	}{
		{name: "tool name", identifier: "my-tool"},
		{name: "prompt name", identifier: "my-prompt"},
		{name: "resource URI", identifier: "file:///path/to/resource"},
		{name: "empty identifier", identifier: ""},
		{name: "resource template", identifier: "template://{id}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSession := testutil.NewMockSession()
			reqCtx := &mcpio.MCPRequestContext{
				Identifier: tt.identifier,
				Session:    mockSession.Session,
				Headers:    http.Header{},
			}
			assert.Equal(t, tt.identifier, reqCtx.GetIdentifier())
		})
	}
}

func TestMCPRequestContext_GetTokenInfo(t *testing.T) {
	t.Parallel()

	t.Run("nil token", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		reqCtx := &mcpio.MCPRequestContext{
			TokenInfo: nil,
			Session:   mockSession.Session,
			Headers:   http.Header{},
		}
		assert.Nil(t, reqCtx.GetTokenInfo())
	})

	t.Run("with complete token info", func(t *testing.T) {
		expiration := time.Now().Add(time.Hour)
		token := &auth.TokenInfo{
			Scopes:     []string{"read", "write"},
			Expiration: expiration,
			Extra:      map[string]any{"custom": "value"},
		}
		mockSession := testutil.NewMockSession()
		reqCtx := &mcpio.MCPRequestContext{
			TokenInfo: token,
			Session:   mockSession.Session,
			Headers:   http.Header{},
		}

		result := reqCtx.GetTokenInfo()
		require.NotNil(t, result)
		assert.Equal(t, []string{"read", "write"}, result.Scopes)
		assert.Equal(t, expiration, result.Expiration)
		assert.Equal(t, "value", result.Extra["custom"])
	})

	t.Run("with scopes only", func(t *testing.T) {
		token := &auth.TokenInfo{
			Scopes: []string{"read"},
		}
		mockSession := testutil.NewMockSession()
		reqCtx := &mcpio.MCPRequestContext{
			TokenInfo: token,
			Session:   mockSession.Session,
			Headers:   http.Header{},
		}

		result := reqCtx.GetTokenInfo()
		require.NotNil(t, result)
		assert.Equal(t, []string{"read"}, result.Scopes)
		assert.True(t, result.Expiration.IsZero())
		assert.Nil(t, result.Extra)
	})
}

func TestMCPRequestContext_GetHeaders(t *testing.T) {
	t.Parallel()

	t.Run("empty headers", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		reqCtx := &mcpio.MCPRequestContext{
			Headers: http.Header{},
			Session: mockSession.Session,
		}
		headers := reqCtx.GetHeaders()
		assert.NotNil(t, headers, "Headers should never be nil")
		assert.Empty(t, headers)
	})

	t.Run("with single header", func(t *testing.T) {
		headers := http.Header{
			"X-Custom-Header": []string{"value1"},
		}
		mockSession := testutil.NewMockSession()
		reqCtx := &mcpio.MCPRequestContext{
			Headers: headers,
			Session: mockSession.Session,
		}

		result := reqCtx.GetHeaders()
		assert.Equal(t, headers, result)
		assert.Equal(t, "value1", result.Get("X-Custom-Header"))
	})

	t.Run("with multiple headers and values", func(t *testing.T) {
		headers := http.Header{
			"X-Custom-Header": []string{"value1", "value2"},
			"Authorization":   []string{"Bearer token"},
			"Content-Type":    []string{"application/json"},
		}
		mockSession := testutil.NewMockSession()
		reqCtx := &mcpio.MCPRequestContext{
			Headers: headers,
			Session: mockSession.Session,
		}

		result := reqCtx.GetHeaders()
		assert.Equal(t, headers, result)
		assert.Equal(t, "value1", result.Get("X-Custom-Header"))
		assert.Equal(t, "Bearer token", result.Get("Authorization"))
		assert.Equal(t, "application/json", result.Get("Content-Type"))
	})

	t.Run("header case insensitivity", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": []string{"text/plain"},
		}
		mockSession := testutil.NewMockSession()
		reqCtx := &mcpio.MCPRequestContext{
			Headers: headers,
			Session: mockSession.Session,
		}

		assert.Equal(t, "text/plain", reqCtx.GetHeaders().Get("Content-Type"))
		assert.Equal(t, "text/plain", reqCtx.GetHeaders().Get("content-type"))
		assert.Equal(t, "text/plain", reqCtx.GetHeaders().Get("CONTENT-TYPE"))
	})
}
