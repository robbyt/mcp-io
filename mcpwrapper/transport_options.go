package mcpwrapper

import (
	"log/slog"
	"time"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TransportOptionStreamableHTTP is a functional option for configuring mcp.StreamableHTTPOptions.
type TransportOptionStreamableHTTP func(*mcpSDK.StreamableHTTPOptions) error

// WithStateless makes the server stateless (no session validation).
//
// A stateless server does not validate the Mcp-Session-Id header, and uses a
// temporary session with default initialization parameters. Any server->client
// request is rejected immediately as there's no way for the client to respond.
//
// Example:
//
//	mcp.WithStateless()
func WithStateless() TransportOptionStreamableHTTP {
	return func(opts *mcpSDK.StreamableHTTPOptions) error {
		opts.Stateless = true
		return nil
	}
}

// WithJSONResponse returns application/json instead of text/event-stream.
//
// Per MCP spec §2.1.5, this causes streamable responses to return JSON
// rather than server-sent events.
//
// Example:
//
//	mcp.WithJSONResponse()
func WithJSONResponse() TransportOptionStreamableHTTP {
	return func(opts *mcpSDK.StreamableHTTPOptions) error {
		opts.JSONResponse = true
		return nil
	}
}

// WithHTTPLogger sets the logger for HTTP transport operations.
//
// The logger receives events about HTTP transport activity including
// SSE stream lifecycle, session management, and HTTP request handling.
//
// This is separate from server-side logging (WithServerLogger). You can set
// different loggers for server protocol events vs HTTP transport events.
//
// If not set, the SDK uses a discard logger (no output).
//
// Example:
//
//	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
//	    Level: slog.LevelInfo,
//	}))
//	mcp.WithHTTPLogger(logger)
func WithHTTPLogger(logger *slog.Logger) TransportOptionStreamableHTTP {
	return func(opts *mcpSDK.StreamableHTTPOptions) error {
		opts.Logger = logger
		return nil
	}
}

// WithHTTPSessionTimeout sets the idle timeout for HTTP sessions.
//
// When set, inactive sessions (no POST requests) are automatically closed
// after the specified duration to prevent resource leaks. The timeout is
// paused during active requests.
//
// A zero value (default) means sessions never timeout.
//
// Example:
//
//	mcp.WithHTTPSessionTimeout(5 * time.Minute)
func WithHTTPSessionTimeout(timeout time.Duration) TransportOptionStreamableHTTP {
	return func(opts *mcpSDK.StreamableHTTPOptions) error {
		opts.SessionTimeout = timeout
		return nil
	}
}

// WithEventStore enables SSE stream resumption using the provided EventStore.
//
// Stream resumption allows clients with flaky connections (mobile, unreliable
// networks) to reconnect and continue receiving events from where they left off.
// The EventStore persists events for each session's streams.
//
// By default (nil), stream resumption is disabled and events are only delivered
// once in real-time.
//
// Use mcp.NewMemoryEventStore(nil) for single-server deployments. For multi-instance
// servers, implement a distributed EventStore using Redis, etcd, or similar.
//
// Example:
//
//	eventStore := mcp.NewMemoryEventStore(nil)
//	mcp.WithEventStore(eventStore)
func WithEventStore(store mcpSDK.EventStore) TransportOptionStreamableHTTP {
	return func(opts *mcpSDK.StreamableHTTPOptions) error {
		opts.EventStore = store
		return nil
	}
}
