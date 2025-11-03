package testutil

import (
	"context"
	"errors"
	"log"
	"sync"
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
	ctx := t.Context()

	// Set the transport on the server before running
	server := handler.GetServer()
	if mcpServer, ok := server.Unwrap().(*mcp.Server); ok {
		// Need to get the wrapper and set transport
		// For in-memory testing, we need to use the SDK server directly
		go func() {
			if runErr := mcpServer.Run(ctx, serverTransport); runErr != nil {
				// Don't log expected context cancellation when test ends
				if !errors.Is(runErr, context.Canceled) {
					// Use log.Printf instead of t.Logf to avoid data race
					log.Printf("server run error: %v", runErr)
				}
			}
		}()
	} else {
		t.Fatal("failed to unwrap MCP server for in-memory testing")
	}

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

// ConnectWithProgressCapture sets up an in-memory MCP client/server connection
// that captures progress notifications. Returns the session and a function to
// retrieve captured notifications (thread-safe).
func ConnectWithProgressCapture(t *testing.T, handler *mcpio.Handler) (*MCPSession, func() []*mcp.ProgressNotificationParams) {
	t.Helper()

	// Capture progress notifications with mutex for thread safety
	var notifications []*mcp.ProgressNotificationParams
	var mu sync.Mutex

	// Create in-memory transports
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ctx := t.Context()

	// Set up server
	server := handler.GetServer()
	mcpServer, ok := server.Unwrap().(*mcp.Server)
	if !ok {
		t.Fatal("failed to unwrap MCP server")
	}

	go func() {
		if runErr := mcpServer.Run(ctx, serverTransport); runErr != nil {
			if !errors.Is(runErr, context.Canceled) {
				log.Printf("server run error: %v", runErr)
			}
		}
	}()

	// Connect client with progress handler
	client := mcp.NewClient(DefaultTestImplementation(), &mcp.ClientOptions{
		ProgressNotificationHandler: func(ctx context.Context, req *mcp.ProgressNotificationClientRequest) {
			mu.Lock()
			defer mu.Unlock()
			notifications = append(notifications, req.Params)
		},
	})

	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect MCP client: %v", err)
	}

	mcpSession := &MCPSession{ClientSession: session, t: t}
	t.Cleanup(func() { mcpSession.Close() })

	// Return getter that returns a copy for thread safety
	getNotifications := func() []*mcp.ProgressNotificationParams {
		mu.Lock()
		defer mu.Unlock()
		result := make([]*mcp.ProgressNotificationParams, len(notifications))
		copy(result, notifications)
		return result
	}

	return mcpSession, getNotifications
}
