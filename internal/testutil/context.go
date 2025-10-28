package testutil

import (
	"github.com/robbyt/mcp-io/capabilities"
)

// WithSession injects a session into the context for testing.
// Use this in tests to provide session context without setting up a full MCP handler.
//
// Example:
//
//	mockSession := testutil.NewMockSession()
//	ctx := testutil.WithSession(t.Context(), mockSession.Session)
//	result, err := myToolFunc(ctx, input)
var WithSession = capabilities.WithSession
