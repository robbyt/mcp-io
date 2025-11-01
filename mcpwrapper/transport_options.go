package mcpwrapper

import (
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
