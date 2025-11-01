package mcpwrapper

import (
	"context"
	"net/http"

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

type Server struct {
	server    *mcpSDK.Server
	transport mcpSDK.Transport
	httpOpts  *mcpSDK.StreamableHTTPOptions
}

// New wraps an SDK server with optional configuration.
// The transport can be set later using SetTransport.
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

// SetTransport sets the transport for the server
func (s *Server) SetTransport(transport mcpSDK.Transport) {
	s.transport = transport
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
func (s *Server) Run(ctx context.Context) error {
	return s.server.Run(ctx, s.transport)
}

// ServeHTTP implements http.Handler for Streamable HTTP transport.
// Creates a StreamableHTTPHandler using the server's configured HTTP options.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	httpHandler := mcpSDK.NewStreamableHTTPHandler(
		func(*http.Request) *mcpSDK.Server { return s.server },
		s.httpOpts,
	)
	httpHandler.ServeHTTP(w, r)
}
