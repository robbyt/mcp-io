//go:build integration

package mcpio_test

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
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
