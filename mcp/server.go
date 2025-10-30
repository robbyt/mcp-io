package mcp

import (
	"context"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server struct {
	server    *mcpSDK.Server
	transport mcpSDK.Transport
}

// New wraps an SDK server without specifying a transport.
// The transport can be set later using SetTransport.
func New(server *mcpSDK.Server) *Server {
	return &Server{
		server:    server,
		transport: nil,
	}
}

// SetTransport sets the transport for the server
func (s *Server) SetTransport(transport mcpSDK.Transport) {
	s.transport = transport
}

// NewInMemoryServer creates a server with in-memory transport for testing.
// Returns both the wrapped server and the client transport for connecting test clients.
func NewInMemoryServer(server *mcpSDK.Server) (*Server, *mcpSDK.InMemoryTransport) {
	serverTransport, clientTransport := mcpSDK.NewInMemoryTransports()
	return &Server{
		server:    server,
		transport: serverTransport,
	}, clientTransport
}

// Run starts the MCP server with the configured transport.
func (s *Server) Run(ctx context.Context) error {
	return s.server.Run(ctx, s.transport)
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
