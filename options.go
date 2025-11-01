package mcpio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/mcpwrapper"
	"github.com/robbyt/mcp-io/primitives/tool"
)

// Option is a functional option for configuring handlers
type Option func(*handlerConfig) error

// WithName sets the server name
func WithName(name string) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return fmt.Errorf("name cannot be empty: %w", ErrEmptyValue)
		}
		cfg.name = name
		return nil
	}
}

// WithVersion sets the server version
func WithVersion(version string) Option {
	return func(cfg *handlerConfig) error {
		if version == "" {
			return fmt.Errorf("version cannot be empty: %w", ErrEmptyValue)
		}
		cfg.version = version
		return nil
	}
}

// WithServer allows injecting a custom server for testing
func WithServer(server *mcp.Server) Option {
	return func(cfg *handlerConfig) error {
		if server == nil {
			return fmt.Errorf("server cannot be nil: %w", ErrNilValue)
		}
		cfg.server = server
		return nil
	}
}

// WithServerConfig sets the complete MCP server options struct.
// Use this for advanced configurations requiring handler functions.
// For simple configurations, prefer WithServerOptions with functional options.
func WithServerConfig(opts *mcp.ServerOptions) Option {
	return func(cfg *handlerConfig) error {
		if opts == nil {
			return fmt.Errorf("server options cannot be nil: %w", ErrNilValue)
		}
		cfg.serverOptions = opts
		return nil
	}
}

// WithHTTPTransport configures the Streamable HTTP transport.
// Accepts variadic TransportOption functions for configuration (e.g., mcp.WithStateless()).
//
// Example:
//
//	mcpio.WithHTTPTransport(
//	    mcp.WithStateless(),
//	    mcp.WithJSONResponse(),
//	)
func WithHTTPTransport(opts ...mcpwrapper.TransportOptionStreamableHTTP) Option {
	return func(cfg *handlerConfig) error {
		// Apply options to existing httpOpts (modify defaults)
		for _, opt := range opts {
			if err := opt(cfg.httpOpts); err != nil {
				return err
			}
		}
		return nil
	}
}

// WithServerOptions configures MCP server options using functional options.
// Accepts variadic ServerOption functions for configuration.
//
// Example:
//
//	mcpio.WithServerOptions(
//	    mcp.WithInstructions("Use this server for data processing"),
//	    mcp.WithPageSize(50),
//	    mcp.WithCapabilityPrompts(),
//	)
func WithServerOptions(opts ...mcpwrapper.ServerOption) Option {
	return func(cfg *handlerConfig) error {
		// Apply options to existing serverOptions (modify defaults)
		for _, opt := range opts {
			opt(cfg.serverOptions)
		}
		return nil
	}
}

// WithPrompt adds a prompt to the handler
func WithPrompt(name, description string, fn PromptFunc) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return fmt.Errorf("prompt name cannot be empty: %w", ErrEmptyValue)
		}

		registerFunc := func(server *mcp.Server) error {
			prompt := &mcp.Prompt{
				Name:        name,
				Description: description,
			}
			handler := createPromptHandler(fn)
			server.AddPrompt(prompt, handler)
			return nil
		}

		cfg.prompts = append(cfg.prompts, registerFunc)
		return nil
	}
}

// WithPromptWithArgs adds a prompt with argument definitions
func WithPromptWithArgs(name, description string, args []*mcp.PromptArgument, fn PromptFunc) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return fmt.Errorf("prompt name cannot be empty: %w", ErrEmptyValue)
		}

		registerFunc := func(server *mcp.Server) error {
			prompt := &mcp.Prompt{
				Name:        name,
				Description: description,
				Arguments:   args,
			}
			handler := createPromptHandler(fn)
			server.AddPrompt(prompt, handler)
			return nil
		}

		cfg.prompts = append(cfg.prompts, registerFunc)
		return nil
	}
}

// WithTypedPrompt adds a type-safe prompt with automatic schema generation
func WithTypedPrompt[TArgs any](name, description string, fn TypedPromptFunc[TArgs]) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return fmt.Errorf("prompt name cannot be empty: %w", ErrEmptyValue)
		}

		// Generate schema from the TArgs type
		s, err := jsonschema.For[TArgs](nil)
		if err != nil {
			return fmt.Errorf("failed to generate schema for prompt %s: %w", name, err)
		}

		// Convert schema to prompt arguments
		args, err := schemaToPromptArguments(s)
		if err != nil {
			return fmt.Errorf("failed to convert schema to prompt arguments for %s: %w", name, err)
		}

		registerFunc := func(server *mcp.Server) error {
			prompt := &mcp.Prompt{
				Name:        name,
				Description: description,
				Arguments:   args,
			}
			handler := createTypedPromptHandler(fn)
			server.AddPrompt(prompt, handler)
			return nil
		}

		cfg.prompts = append(cfg.prompts, registerFunc)
		return nil
	}
}

// WithResource adds a resource to the handler
func WithResource(uri, description string, fn ResourceFunc) Option {
	return func(cfg *handlerConfig) error {
		if uri == "" {
			return fmt.Errorf("resource URI cannot be empty: %w", ErrEmptyValue)
		}

		registerFunc := func(server *mcp.Server) error {
			resource := &mcp.Resource{
				URI:         uri,
				Name:        uri, // Use URI as name by default
				Description: description,
			}
			handler := createResourceHandler(fn)
			server.AddResource(resource, handler)
			return nil
		}

		cfg.resources = append(cfg.resources, registerFunc)
		return nil
	}
}

// WithResourceTemplate adds a resource template to the handler
func WithResourceTemplate(uriTemplate, description string, fn ResourceFunc) Option {
	return func(cfg *handlerConfig) error {
		if uriTemplate == "" {
			return fmt.Errorf("resource template cannot be empty: %w", ErrEmptyValue)
		}

		registerFunc := func(server *mcp.Server) error {
			template := &mcp.ResourceTemplate{
				URITemplate: uriTemplate,
				Name:        uriTemplate, // Use template as name by default
				Description: description,
			}
			handler := createResourceHandler(fn) // Same handler type
			server.AddResourceTemplate(template, handler)
			return nil
		}

		cfg.resourceTemplates = append(cfg.resourceTemplates, registerFunc)
		return nil
	}
}

// ToolFunc is the function signature for typed tools with automatic schema generation.
// The function receives a context, ToolContext for accessing session/metadata, typed input,
// and returns typed output with an optional error.
// Schema generation is handled automatically based on the TIn and TOut types.
type ToolFunc[TIn, TOut any] func(context.Context, RequestContext, TIn) (TOut, error)

// WithTool adds a type-safe tool with automatic schema generation from Go types.
//
// The InputSchema is automatically generated from the TIn type parameter.
// Optional metadata can be provided using functional options from the primitives/tool package.
//
// Examples:
//
//	// Simple case: Auto-generated schemas only
//	WithTool("to_upper", "Convert text to uppercase", toUpperFunc)
//
//	// With metadata options (recommended import alias: import toolOption "github.com/robbyt/mcp-io/primitives/tool")
//	WithTool("to_upper", "Convert text to uppercase", toUpperFunc,
//	    toolOption.WithTitle("Text Uppercaser"),
//	    toolOption.WithReadOnly(),
//	    toolOption.WithIdempotent(),
//	)
func WithTool[TIn, TOut any](name, description string, fn ToolFunc[TIn, TOut], opts ...tool.Option) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return fmt.Errorf("tool name cannot be empty: %w", ErrEmptyValue)
		}

		registerFunc := func(handler *Handler, server *mcp.Server) error {
			// Create tool with metadata
			mcpTool := &mcp.Tool{
				Name:        name,
				Description: description,
			}

			// Apply metadata options directly to the tool
			for _, opt := range opts {
				if err := opt(mcpTool); err != nil {
					return err
				}
			}

			handlerFunc := createTypedHandler(handler, fn)
			mcp.AddTool(server, mcpTool, handlerFunc) // Auto-generates InputSchema from TIn
			return nil
		}

		cfg.tools = append(cfg.tools, registerFunc)
		return nil
	}
}

// RawToolFunc is the function signature for raw JSON tools.
// The function receives a context, ToolContext for accessing session/metadata,
// and raw JSON bytes as input, and returns JSON bytes as output.
// Schema must be provided explicitly when using WithRawTool.
type RawToolFunc func(context.Context, RequestContext, []byte) ([]byte, error)

// WithRawTool adds a tool with manual JSON handling and explicit schema.
// Use this when you need direct control over JSON processing or dynamic schemas.
// Optional metadata can be provided via functional options from primitives/tool package.
//
// Examples:
//
//	rawFunc := func(ctx context.Context, input []byte) ([]byte, error) { return input, nil }
//	WithRawTool("process", "Process JSON", `{"type":"object"}`, rawFunc)
//
//	// With metadata (recommended import alias: import toolOption "github.com/robbyt/mcp-io/primitives/tool")
//	WithRawTool("process", "Process JSON", inputSchema, rawFunc,
//	    toolOption.WithTitle("JSON Processor"),
//	    toolOption.WithReadOnly(),
//	)
func WithRawTool(name, description string, inputSchema any, fn RawToolFunc, opts ...tool.Option) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return fmt.Errorf("tool name cannot be empty: %w", ErrEmptyValue)
		}
		if inputSchema == nil {
			return fmt.Errorf("input schema cannot be nil: %w", ErrNilValue)
		}
		// Check for typed nil pointers
		if s, ok := inputSchema.(*jsonschema.Schema); ok && s == nil {
			return fmt.Errorf("input schema cannot be nil: %w", ErrNilValue)
		}

		// Create registration function
		registerFunc := func(handler *Handler, server *mcp.Server) error {
			// Use primitives/tool to construct the tool
			mcpTool, err := tool.New(name, description, inputSchema, opts...)
			if err != nil {
				return err
			}

			handlerFunc := createRawToolHandler(handler, fn)
			server.AddTool(mcpTool, handlerFunc)
			return nil
		}

		cfg.tools = append(cfg.tools, registerFunc)

		return nil
	}
}

// createRawToolHandler wraps a raw function to match the MCP ToolHandler signature
func createRawToolHandler(handler *Handler, fn RawToolFunc) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Create request context with all MCP metadata
		reqCtx := newRequestContext(req.Params.Name, req.Session, req.Extra)

		// Marshal input arguments to JSON bytes
		inputJSON, err := json.Marshal(req.Params.Arguments)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Failed to marshal input: %v", err)},
				},
				IsError: true,
			}, nil
		}

		// Execute raw function (pass reqCtx as ToolContext)
		outputJSON, err := fn(ctx, reqCtx, inputJSON)
		if err != nil {
			// Check if it's a tool error
			var toolErr *ToolError
			if errors.As(err, &toolErr) {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: toolErr.Error()},
					},
					IsError: true,
				}, nil
			}
			// Protocol error
			return nil, err
		}

		// Parse output for structured response
		var output any
		if err := json.Unmarshal(outputJSON, &output); err != nil {
			// Raw tools must return valid JSON
			return nil, errors.Join(ErrInvalidJSON, err)
		}

		// Return structured output
		outputJSONStr := string(outputJSON)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: outputJSONStr},
			},
		}, nil
	}
}
