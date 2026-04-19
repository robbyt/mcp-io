// Package mcpwrapper provides a wrapper around the MCP SDK server to simplify
// transport configuration and provide a consistent interface for both stdio
// and HTTP transports.
package mcpwrapper

import (
	"context"
	"net/http"
	"sync"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
)

// WrapperOption configures a Server wrapper
type WrapperOption func(*Server)

// WithHTTPOptions sets custom HTTP transport options for the wrapper
func WithHTTPOptions(opts *mcpSDK.StreamableHTTPOptions) WrapperOption {
	return func(s *Server) {
		s.httpOpts = opts
	}
}

// WithTransport sets the transport for the wrapper
func WithTransport(transport mcpSDK.Transport) WrapperOption {
	return func(s *Server) {
		s.transport = transport
	}
}

type Server struct {
	server      *mcpSDK.Server
	transport   mcpSDK.Transport
	httpOpts    *mcpSDK.StreamableHTTPOptions
	httpOnce    sync.Once
	httpHandler *mcpSDK.StreamableHTTPHandler
}

// New wraps an SDK server with optional configuration.
// The transport can be configured using WithTransport option.
// HTTP options default to stateful + SSE, override with WithHTTPOptions().
func New(server *mcpSDK.Server, opts ...WrapperOption) *Server {
	s := &Server{
		server:    server,
		transport: nil,
		httpOpts: &mcpSDK.StreamableHTTPOptions{
			Stateless:    false, // Default: stateful sessions
			JSONResponse: false, // Default: SSE streaming
		},
	}

	// Apply optional configurations
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// GetTransport returns the currently configured transport for the server.
// Returns nil if no transport has been set.
//
// This is useful for advanced use cases where access to the underlying transport
// is needed, such as integrating with external MCP client libraries.
func (s *Server) GetTransport() mcpSDK.Transport {
	return s.transport
}

// NewInMemoryServer creates a server with in-memory transport for testing.
// Returns both the wrapped server and the client transport for connecting test clients.
// HTTP options default to stateful + SSE, override with WithHTTPOptions().
func NewInMemoryServer(server *mcpSDK.Server, opts ...WrapperOption) (*Server, *mcpSDK.InMemoryTransport) {
	serverTransport, clientTransport := mcpSDK.NewInMemoryTransports()

	s := &Server{
		server:    server,
		transport: serverTransport,
		httpOpts: &mcpSDK.StreamableHTTPOptions{
			Stateless:    false, // Default: stateful sessions
			JSONResponse: false, // Default: SSE streaming
		},
	}

	// Apply optional configurations
	for _, opt := range opts {
		opt(s)
	}

	return s, clientTransport
}

// AddTool registers a tool with the server
func (s *Server) AddTool(tool *mcpSDK.Tool, handler mcpSDK.ToolHandler) {
	s.server.AddTool(tool, handler)
}

// AddPrompt registers a prompt with the server
func (s *Server) AddPrompt(prompt *mcpSDK.Prompt, handler mcpSDK.PromptHandler) {
	s.server.AddPrompt(prompt, handler)
}

// AddResource registers a resource with the server
func (s *Server) AddResource(resource *mcpSDK.Resource, handler mcpSDK.ResourceHandler) {
	s.server.AddResource(resource, handler)
}

// AddResourceTemplate registers a resource template with the server
func (s *Server) AddResourceTemplate(template *mcpSDK.ResourceTemplate, handler mcpSDK.ResourceHandler) {
	s.server.AddResourceTemplate(template, handler)
}

// Unwrap returns the underlying SDK server for advanced usage
func (s *Server) Unwrap() any {
	return s.server
}

// Run starts the MCP server with the configured transport.
// If no transport was set via WithTransport(), defaults to StdioTransport.
func (s *Server) Run(ctx context.Context) error {
	transport := s.transport
	if transport == nil {
		transport = &mcpSDK.StdioTransport{}
	}
	return s.server.Run(ctx, transport)
}

// ServeHTTP implements http.Handler for Streamable HTTP transport.
// Creates a StreamableHTTPHandler using the server's configured HTTP options.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.httpOnce.Do(func() {
		s.httpHandler = mcpSDK.NewStreamableHTTPHandler(
			func(*http.Request) *mcpSDK.Server { return s.server },
			s.httpOpts,
		)
	})
	s.httpHandler.ServeHTTP(w, r)
}
