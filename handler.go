package mcpio

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/mcpwrapper"
)

// Resource registration function types
type (
	promptRegisterFunc           func(*mcp.Server) error
	resourceRegisterFunc         func(*mcp.Server) error
	resourceTemplateRegisterFunc func(*mcp.Server) error

	// toolRegisterFunc is an internal function type that registers a tool on an MCP server.
	// This is used internally by the option functions to defer tool registration.
	// It receives both the handler (for creating tool handlers) and the server (for registration).
	toolRegisterFunc func(*Handler, *mcp.Server) error
)

// handlerConfig holds the configuration built by options while setting up a Handler instance
type handlerConfig struct {
	name              string
	version           string
	tools             []toolRegisterFunc
	prompts           []promptRegisterFunc
	resources         []resourceRegisterFunc
	resourceTemplates []resourceTemplateRegisterFunc
	server            *mcp.Server
	serverOptions     *mcp.ServerOptions
	httpOpts          *mcp.StreamableHTTPOptions
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
		tools:             make([]toolRegisterFunc, 0),
		prompts:           make([]promptRegisterFunc, 0),
		resources:         make([]resourceRegisterFunc, 0),
		resourceTemplates: make([]resourceTemplateRegisterFunc, 0),
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

	// Create wrapped server with Stdio transport (default)
	wrappedServer := mcpwrapper.New(cfg.server, mcpwrapper.WithHTTPOptions(cfg.httpOpts))
	wrappedServer.SetTransport(&mcp.StdioTransport{})

	// Create the handler before registration (needed by registerFunc)
	handler := &Handler{
		server: wrappedServer,
	}

	// Register all resources
	errz := make([]error, 0)
	for _, registerFunc := range cfg.tools {
		if err := registerFunc(handler, cfg.server); err != nil {
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

// ServeHTTP implements http.Handler for Streamable HTTP transport.
// Delegates to the MCPServer's ServeHTTP method.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.server.ServeHTTP(w, r)
}

// ServeStdio runs the server using stdio transport for command-line tools.
// Uses os.Stdin and os.Stdout as configured by the MCP SDK.
func (h *Handler) ServeStdio(ctx context.Context) error {
	// Transport is already set, just run the server
	return h.server.Run(ctx)
}
