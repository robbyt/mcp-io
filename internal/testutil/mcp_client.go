package testutil

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
)

// MCPSession wraps an MCP client session with automatic cleanup
type MCPSession struct {
	*mcp.ClientSession
	t *testing.T
}

// Close closes the MCP session and logs any errors
func (s *MCPSession) Close() {
	if err := s.ClientSession.Close(); err != nil {
		s.t.Logf("error closing MCP session: %v", err)
	}
}

// DefaultTestImplementation returns the standard test client implementation
func DefaultTestImplementation() *mcp.Implementation {
	return &mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}
}

// ConnectInMemory sets up an in-memory MCP client/server connection for testing.
// It starts the server in a goroutine and connects a client, returning the session.
// Cleanup is automatically registered via t.Cleanup().
func ConnectInMemory(t *testing.T, handler *mcpio.Handler) *MCPSession {
	t.Helper()

	// Create in-memory transports
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()

	// Start server in background
	go func() {
		if runErr := handler.GetServer().Run(ctx, serverTransport); runErr != nil {
			t.Logf("server run error: %v", runErr)
		}
	}()

	// Connect client
	client := mcp.NewClient(DefaultTestImplementation(), nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect MCP client: %v", err)
	}

	// Wrap session with cleanup
	mcpSession := &MCPSession{
		ClientSession: session,
		t:             t,
	}

	// Register cleanup
	t.Cleanup(func() {
		mcpSession.Close()
	})

	return mcpSession
}
