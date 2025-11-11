package mcpio

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/mcpwrapper"
)

// registerFunc is used internally to defer resource registration until the server is created
type registerFunc func(*mcp.Server) error

// handlerConfig holds the configuration built by options while setting up a Handler instance
type handlerConfig struct {
	name              string
	version           string
	tools             []registerFunc
	prompts           []registerFunc
	resources         []registerFunc
	resourceTemplates []registerFunc
	server            *mcp.Server
	serverOptions     *mcp.ServerOptions
	httpOpts          *mcp.StreamableHTTPOptions
	transport         mcp.Transport
}

// MCPServer provides an interface to the concrete MCP SDK server instance.
//
// The Unwrap method provides an escape hatch for power users who need
// access to features not yet wrapped by mcp-io.
type MCPServer interface {
	// Tool registration
	AddTool(tool *mcp.Tool, handler mcp.ToolHandler)

	// Prompt registration
	AddPrompt(prompt *mcp.Prompt, handler mcp.PromptHandler)

	// Resource registration
	AddResource(resource *mcp.Resource, handler mcp.ResourceHandler)
	AddResourceTemplate(template *mcp.ResourceTemplate, handler mcp.ResourceHandler)

	// Transport - run the server with configured transport
	Run(ctx context.Context) error

	// GetTransport returns the currently configured transport
	GetTransport() mcp.Transport

	// ServeHTTP implements http.Handler for Streamable HTTP transport
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// Unwrap returns the underlying SDK server for advanced usage.
	// Returns *mcp.Server from github.com/modelcontextprotocol/go-sdk/mcp
	// Returns nil if this is a mock implementation.
	Unwrap() any
}

// Handler is the main MCP handler struct
type Handler struct {
	server MCPServer
}

// NewHandler creates a new MCP handler that supports any combination of MCP resources.
// This is the unified constructor that can handle tools, prompts, resources, and resource templates.
func NewHandler(opts ...Option) (*Handler, error) {
	cfg := &handlerConfig{
		name:              "mcp-server",
		version:           "1.0.0",
		tools:             make([]registerFunc, 0),
		prompts:           make([]registerFunc, 0),
		resources:         make([]registerFunc, 0),
		resourceTemplates: make([]registerFunc, 0),
		server:            nil, // Will be created if not provided
		serverOptions:     &mcp.ServerOptions{},
		httpOpts: &mcp.StreamableHTTPOptions{
			Stateless:    false, // Default: stateful sessions
			JSONResponse: false, // Default: SSE streaming
		},
	}

	// Apply all options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Create a new MCP server if not provided
	if cfg.server == nil {
		impl := &mcp.Implementation{
			Name:    cfg.name,
			Version: cfg.version,
		}
		cfg.server = mcp.NewServer(impl, cfg.serverOptions)
	}

	// Create wrapped server with configured transport
	// Run() will default to StdioTransport if nil
	wrappedServer := mcpwrapper.New(
		cfg.server,
		mcpwrapper.WithHTTPOptions(cfg.httpOpts),
		mcpwrapper.WithTransport(cfg.transport),
	)

	// Create the handler before registration (needed by registerFunc)
	handler := &Handler{
		server: wrappedServer,
	}

	// Register all resources
	errz := make([]error, 0)
	for _, registerFunc := range cfg.tools {
		if err := registerFunc(cfg.server); err != nil {
			errz = append(errz, fmt.Errorf("failed to register tool: %w", err))
		}
	}
	for _, registerFunc := range cfg.prompts {
		if err := registerFunc(cfg.server); err != nil {
			errz = append(errz, fmt.Errorf("failed to register prompt: %w", err))
		}
	}
	for _, registerFunc := range cfg.resources {
		if err := registerFunc(cfg.server); err != nil {
			errz = append(errz, fmt.Errorf("failed to register resource: %w", err))
		}
	}
	for _, registerFunc := range cfg.resourceTemplates {
		if err := registerFunc(cfg.server); err != nil {
			errz = append(errz, fmt.Errorf("failed to register resource template: %w", err))
		}
	}
	if len(errz) > 0 {
		return nil, errors.Join(errz...)
	}

	return handler, nil
}

// GetServer returns the underlying MCP server for advanced usage
func (h *Handler) GetServer() MCPServer {
	return h.server
}

// GetTransport returns the currently configured transport for this handler.
// Returns nil if no transport has been set.
//
// This is useful for advanced use cases where access to the underlying transport
// is needed, such as when integrating with external MCP client libraries like Google ADK.
//
// For common use cases with in-memory transports, consider using NewInMemoryPair()
// instead, which returns both the handler and client transport in one call.
//
// Example:
//
//	handler, _ := mcpio.NewHandler(mcpio.WithTool(...))
//	transport := handler.GetTransport()
//	// transport can now be used with external MCP clients
func (h *Handler) GetTransport() mcp.Transport {
	return h.server.GetTransport()
}

// Run starts the MCP server with the configured transport.
// This is the main entry point for starting the server.
//
// For stdio transport (default), this reads from os.Stdin and writes to os.Stdout.
// For in-memory transport (from NewInMemoryPair), this connects to the paired client transport.
// For HTTP transport, use ServeHTTP instead.
func (h *Handler) Run(ctx context.Context) error {
	return h.server.Run(ctx)
}

// ServeHTTP implements http.Handler for Streamable HTTP transport.
// Delegates to the MCPServer's ServeHTTP method.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.server.ServeHTTP(w, r)
}

// ServeStdio runs the server using stdio transport for command-line tools.
// Uses os.Stdin and os.Stdout as configured by the MCP SDK.
//
// Deprecated: Use Run() instead. ServeStdio is kept for backward compatibility
// but Run() is the preferred method as it works with any transport type.
func (h *Handler) ServeStdio(ctx context.Context) error {
	return h.Run(ctx)
}
