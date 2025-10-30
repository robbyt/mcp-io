package mcpio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/capabilities"
	mcpwrapper "github.com/robbyt/mcp-io/mcp"
)

// Resource registration function types
type (
	promptRegisterFunc           func(*mcp.Server) error
	resourceRegisterFunc         func(*mcp.Server) error
	resourceTemplateRegisterFunc func(*mcp.Server) error
)

// handlerConfig holds the configuration built by options
type handlerConfig struct {
	name                  string
	version               string
	tools                 []toolRegisterFunc
	prompts               []promptRegisterFunc
	resources             []resourceRegisterFunc
	resourceTemplates     []resourceTemplateRegisterFunc
	server                *mcp.Server // The MCP-SDK server instance
	serverOptions         *mcp.ServerOptions
	streamableHTTPOptions *mcp.StreamableHTTPOptions
	contextStore          ContextStore // Pluggable context storage
}

// MCPServer provides an interface to the MCP SDK server, allowing for
// dependency injection in tests and decoupling from the concrete SDK types.
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

	// Unwrap returns the underlying SDK server for advanced usage.
	// Returns *mcp.Server from github.com/modelcontextprotocol/go-sdk/mcp
	// Returns nil if this is a mock implementation.
	Unwrap() any
}

// Handler is the main MCP handler struct
type Handler struct {
	server       MCPServer
	handler      *mcp.StreamableHTTPHandler
	contextStore ContextStore
}

// ToolContext provides access to MCP request metadata and session capabilities.
// RequestContext implements this interface and is passed directly to tool functions,
// eliminating the need for context storage and retrieval.
type ToolContext interface {
	// GetSession returns the MCP session for accessing session capabilities like
	// logging, elicitation, and sampling.
	GetSession() *capabilities.Session

	// GetIdentifier returns the identifier for the current request.
	// For tools: tool name, for prompts: prompt name, for resources: URI.
	GetIdentifier() string

	// GetTokenInfo returns OAuth token information if present, nil otherwise.
	GetTokenInfo() *auth.TokenInfo

	// GetHeaders returns HTTP headers from the request.
	GetHeaders() http.Header
}

// GetServer returns the underlying MCP server for advanced usage
func (h *Handler) GetServer() MCPServer {
	return h.server
}

// toolRegisterFunc is an internal function type that registers a tool on an MCP server.
// This is used internally by the option functions to defer tool registration.
// It receives both the handler (for creating tool handlers) and the server (for registration).
type toolRegisterFunc func(*Handler, *mcp.Server) error

// NewHandler creates a new MCP handler that supports any combination of MCP resources.
// This is the unified constructor that can handle tools, prompts, resources, and resource templates.
func NewHandler(opts ...Option) (*Handler, error) {
	cfg := &handlerConfig{
		name:                  "mcp-server",
		version:               "1.0.0",
		tools:                 make([]toolRegisterFunc, 0),
		prompts:               make([]promptRegisterFunc, 0),
		resources:             make([]resourceRegisterFunc, 0),
		resourceTemplates:     make([]resourceTemplateRegisterFunc, 0),
		server:                nil, // Will be created if not provided
		serverOptions:         &mcp.ServerOptions{},
		streamableHTTPOptions: &mcp.StreamableHTTPOptions{},
		contextStore:          NewContextStore(DefaultContextKey),
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

	// Create the handler before registration (needed by registerFunc)
	handler := &Handler{
		server: mcpwrapper.New(cfg.server),
		handler: mcp.NewStreamableHTTPHandler(
			func(*http.Request) *mcp.Server { return cfg.server },
			cfg.streamableHTTPOptions,
		),
		contextStore: cfg.contextStore,
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

// ServeHTTP implements http.Handler for HTTP transport
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}

// ServeStdio implements stdio transport for command-line tools.
// The stdin and stdout parameters are currently unused as the MCP SDK's
// StdioTransport always uses os.Stdin and os.Stdout.
func (h *Handler) ServeStdio(ctx context.Context, _ io.Reader, _ io.Writer) error {
	// h.server is always a *mcpwrapper.Server (wrapped at creation time in NewHandler)
	// Set the stdio transport before running
	h.server.(*mcpwrapper.Server).SetTransport(&mcp.StdioTransport{})
	return h.server.Run(ctx)
}

// createTypedHandler converts a simple typed function into an MCP ToolHandlerFor.
//
// The generics TIn and TOut allow users to define their own custom input/output schemas
// as Go structs, enabling automatic JSON schema generation rather than working with
// generic maps or predefined types.
//
// The returned lambda function acts as an adapter that:
//   - Injects the session into the context
//   - Calls the user's tool function with the deserialized input
//   - Handles error classification (tool errors vs protocol errors)
//   - Returns the typed output for SDK serialization
//
// Parameters:
//   - fn: User-defined tool function with custom input/output types
//
// Returns:
//   - MCP ToolHandlerFor lambda that bridges user code to SDK interface
func createTypedHandler[TIn, TOut any](handler *Handler, fn ToolFunc[TIn, TOut]) mcp.ToolHandlerFor[TIn, TOut] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input TIn) (*mcp.CallToolResult, TOut, error) {
		// Create request context with all MCP metadata
		reqCtx := newRequestContext(req.Params.Name, req.Session, req.Extra)

		// Execute the user-provided tool function (pass reqCtx as ToolContext)
		output, err := fn(ctx, reqCtx, input)
		if err != nil {
			// Check if it's a tool error (user-facing error)
			var toolErr *ToolError
			if errors.As(err, &toolErr) {
				// Tool errors are returned as regular errors - the SDK will handle them
				var zero TOut
				return nil, zero, err
			}
			// Protocol error (system-level error) - return as Go error
			var zero TOut
			return nil, zero, err
		}

		// Success: return structured output (SDK handles serialization)
		return nil, output, nil
	}
}
