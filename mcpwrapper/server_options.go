package mcpwrapper

import (
	"context"
	"log/slog"
	"time"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerOption is a functional option for configuring mcp.ServerOptions.
type ServerOption func(*mcpSDK.ServerOptions)

func ensureServerCapabilities(opts *mcpSDK.ServerOptions) *mcpSDK.ServerCapabilities {
	if opts.Capabilities == nil {
		opts.Capabilities = &mcpSDK.ServerCapabilities{
			Logging: &mcpSDK.LoggingCapabilities{},
		}
	}
	return opts.Capabilities
}

// WithInstructions sets optional instructions for connected clients.
//
// Example:
//
//	mcp.WithInstructions("Use this server for data processing tasks")
func WithInstructions(instructions string) ServerOption {
	return func(opts *mcpSDK.ServerOptions) {
		opts.Instructions = instructions
	}
}

// WithPageSize sets the maximum number of items to return in a single page
// for list methods (e.g., ListTools).
//
// If zero, defaults to the SDK's DefaultPageSize.
//
// Example:
//
//	mcp.WithPageSize(50)
func WithPageSize(pageSize int) ServerOption {
	return func(opts *mcpSDK.ServerOptions) {
		opts.PageSize = pageSize
	}
}

// WithKeepAlive sets the interval for regular "ping" requests.
//
// If the peer fails to respond to pings originating from the keepalive check,
// the session is automatically closed.
//
// Example:
//
//	mcp.WithKeepAlive(30 * time.Second)
func WithKeepAlive(duration time.Duration) ServerOption {
	return func(opts *mcpSDK.ServerOptions) {
		opts.KeepAlive = duration
	}
}

// WithCapabilityPrompts advertises the prompts capability during initialization,
// even if no prompts have been registered.
//
// Example:
//
//	mcp.WithCapabilityPrompts()
func WithCapabilityPrompts() ServerOption {
	return func(opts *mcpSDK.ServerOptions) {
		ensureServerCapabilities(opts).Prompts = &mcpSDK.PromptCapabilities{ListChanged: true}
	}
}

// WithCapabilityResources advertises the resources capability during initialization,
// even if no resources have been registered.
//
// Example:
//
//	mcp.WithCapabilityResources()
func WithCapabilityResources() ServerOption {
	return func(opts *mcpSDK.ServerOptions) {
		ensureServerCapabilities(opts).Resources = &mcpSDK.ResourceCapabilities{ListChanged: true}
	}
}

// WithCapabilityTools advertises the tools capability during initialization,
// even if no tools have been registered.
//
// Example:
//
//	mcp.WithCapabilityTools()
func WithCapabilityTools() ServerOption {
	return func(opts *mcpSDK.ServerOptions) {
		ensureServerCapabilities(opts).Tools = &mcpSDK.ToolCapabilities{ListChanged: true}
	}
}

// WithInitializedHandler sets a handler called when "notifications/initialized" is received.
//
// Example:
//
//	mcp.WithInitializedHandler(func(ctx context.Context, req *mcp.InitializedRequest) {
//	    log.Println("Client initialized")
//	})
func WithInitializedHandler(handler func(context.Context, *mcpSDK.InitializedRequest)) ServerOption {
	return func(opts *mcpSDK.ServerOptions) {
		opts.InitializedHandler = handler
	}
}

// WithRootsListChangedHandler sets a handler called when "notifications/roots/list_changed" is received.
//
// Example:
//
//	mcp.WithRootsListChangedHandler(func(ctx context.Context, req *mcp.RootsListChangedRequest) {
//	    log.Println("Roots list changed")
//	})
func WithRootsListChangedHandler(handler func(context.Context, *mcpSDK.RootsListChangedRequest)) ServerOption {
	return func(opts *mcpSDK.ServerOptions) {
		opts.RootsListChangedHandler = handler
	}
}

// WithProgressNotificationHandler sets a handler called when "notifications/progress" is received.
//
// Example:
//
//	mcp.WithProgressNotificationHandler(func(ctx context.Context, req *mcp.ProgressNotificationServerRequest) {
//	    log.Printf("Progress: %v\n", req)
//	})
func WithProgressNotificationHandler(handler func(context.Context, *mcpSDK.ProgressNotificationServerRequest)) ServerOption {
	return func(opts *mcpSDK.ServerOptions) {
		opts.ProgressNotificationHandler = handler
	}
}

// WithCompletionHandler sets a handler called when "completion/complete" is received.
//
// Example:
//
//	mcp.WithCompletionHandler(func(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
//	    return &mcp.CompleteResult{Completion: completions}, nil
//	})
func WithCompletionHandler(handler func(context.Context, *mcpSDK.CompleteRequest) (*mcpSDK.CompleteResult, error)) ServerOption {
	return func(opts *mcpSDK.ServerOptions) {
		opts.CompletionHandler = handler
	}
}

// WithSubscribeHandler sets a handler called when a client session subscribes to a resource.
//
// Example:
//
//	mcp.WithSubscribeHandler(func(ctx context.Context, req *mcp.SubscribeRequest) error {
//	    log.Printf("Client subscribed to: %s\n", req.Params.URI)
//	    return nil
//	})
func WithSubscribeHandler(handler func(context.Context, *mcpSDK.SubscribeRequest) error) ServerOption {
	return func(opts *mcpSDK.ServerOptions) {
		opts.SubscribeHandler = handler
	}
}

// WithUnsubscribeHandler sets a handler called when a client session unsubscribes from a resource.
//
// Example:
//
//	mcp.WithUnsubscribeHandler(func(ctx context.Context, req *mcp.UnsubscribeRequest) error {
//	    log.Printf("Client unsubscribed from: %s\n", req.Params.URI)
//	    return nil
//	})
func WithUnsubscribeHandler(handler func(context.Context, *mcpSDK.UnsubscribeRequest) error) ServerOption {
	return func(opts *mcpSDK.ServerOptions) {
		opts.UnsubscribeHandler = handler
	}
}

// WithSessionIDGenerator provides a custom function for generating session IDs.
//
// Session IDs should be globally unique across the scope of the server,
// which may span multiple processes in the case of distributed servers.
//
// As a special case, if the function returns an empty string, the
// Mcp-Session-Id header will not be set.
//
// Example:
//
//	mcp.WithSessionIDGenerator(func() string {
//	    return uuid.New().String()
//	})
func WithSessionIDGenerator(generator func() string) ServerOption {
	return func(opts *mcpSDK.ServerOptions) {
		opts.GetSessionID = generator
	}
}

// WithServerLogger sets the logger for server-side MCP protocol operations.
//
// The logger receives events about internal server activity including
// connection lifecycle, message routing, protocol errors, and session management.
// Log output goes to your logging backend, NOT to the MCP client.
//
// For logging that is visible to LLMs and MCP clients, use session.LogInfo(),
// session.LogWarning(), or session.LogError() in your tool handlers instead.
//
// If not set, the SDK uses a discard logger (no output).
//
// Example:
//
//	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
//	    Level: slog.LevelDebug,
//	}))
//	mcp.WithServerLogger(logger)
func WithServerLogger(logger *slog.Logger) ServerOption {
	return func(opts *mcpSDK.ServerOptions) {
		opts.Logger = logger
	}
}
