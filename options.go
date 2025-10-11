package mcpio

import (
	"errors"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/schema"
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

// WithServerOptions sets the MCP server options
func WithServerOptions(opts *mcp.ServerOptions) Option {
	return func(cfg *handlerConfig) error {
		if opts == nil {
			return fmt.Errorf("server options cannot be nil: %w", ErrNilValue)
		}
		cfg.serverOptions = opts
		return nil
	}
}

// WithStreamableHTTPOptions sets the streamable HTTP handler options
func WithStreamableHTTPOptions(opts *mcp.StreamableHTTPOptions) Option {
	return func(cfg *handlerConfig) error {
		if opts == nil {
			return fmt.Errorf("streamable HTTP options cannot be nil: %w", ErrNilValue)
		}
		cfg.streamableHTTPOptions = opts
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
		s, err := schema.New[TArgs]()
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

// WithToolWithSchema adds a type-safe tool with automatic schema generation from Go types.
// Optional schemas parameter overrides the auto-generated schemas from TIn/TOut types.
//
// Examples:
//
//	// Auto-generated schemas: Uses TIn/TOut struct types
//	WithToolWithSchema("to_upper", "Convert text to uppercase", toUpperFunc, nil)
//
//	// Schema override: Replaces auto-generated input/output
//	customSchemas := &ToolSchemas{
//	    InputSchema:  `{"type":"object","properties":{"text":{"type":"string"}}}`,
//	    OutputSchema: json.RawMessage(`{"type":"object"}`),
//	}
//	WithToolWithSchema("to_upper", "Convert text", toUpperFunc, customSchemas)
func WithToolWithSchema[TIn, TOut any](name, description string, fn ToolFunc[TIn, TOut], schemas *ToolSchemas) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return fmt.Errorf("tool name cannot be empty: %w", ErrEmptyValue)
		}

		// Create registration function that uses the generic AddTool
		registerFunc := func(server *mcp.Server) error {
			tool := &mcp.Tool{
				Name:        name,
				Description: description,
			}

			// Apply custom schemas if provided
			if schemas != nil {
				if schemas.InputSchema != nil {
					converted, err := schema.ConvertToRawMessage(schemas.InputSchema)
					if err != nil {
						return fmt.Errorf("tool %q: %w", name, errors.Join(ErrInvalidJSONSchema, err))
					}
					tool.InputSchema = converted
				}
				if schemas.OutputSchema != nil {
					converted, err := schema.ConvertToRawMessage(schemas.OutputSchema)
					if err != nil {
						return fmt.Errorf("tool %q: %w", name, errors.Join(ErrInvalidJSONSchema, err))
					}
					tool.OutputSchema = converted
				}
			}

			handler := createTypedHandler(fn)
			mcp.AddTool(server, tool, handler)
			return nil
		}

		cfg.tools = append(cfg.tools, registerFunc)

		return nil
	}
}

// WithTool adds a type-safe tool with automatic schema generation from Go types.
// This is a convenience wrapper around WithToolWithSchema with nil schemas.
//
// Example:
//
//	WithTool("to_upper", "Convert text to uppercase", toUpperFunc)
func WithTool[TIn, TOut any](name, description string, fn ToolFunc[TIn, TOut]) Option {
	return WithToolWithSchema(name, description, fn, nil)
}

// WithRawTool adds a tool with manual JSON handling and explicit schema.
// Use this when you need direct control over JSON processing or dynamic schemas.
//
//	rawFunc := func(ctx context.Context, input []byte) ([]byte, error) { return input, nil }
//	WithRawTool("process", "Process JSON", `{"type":"object"}`, rawFunc)
func WithRawTool(name, description string, inputSchema any, fn RawToolFunc) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return fmt.Errorf("tool name cannot be empty: %w", ErrEmptyValue)
		}
		if inputSchema == nil {
			return fmt.Errorf("input schema cannot be nil: %w", ErrNilValue)
		}
		// Check for typed nil pointers
		if schema, ok := inputSchema.(*jsonschema.Schema); ok && schema == nil {
			return fmt.Errorf("input schema cannot be nil: %w", ErrNilValue)
		}

		// Create registration function that uses the low-level AddTool
		registerFunc := func(server *mcp.Server) error {
			converted, err := schema.ConvertToRawMessage(inputSchema)
			if err != nil {
				return fmt.Errorf("tool %q: %w", name, errors.Join(ErrInvalidJSONSchema, err))
			}

			tool := &mcp.Tool{
				Name:        name,
				Description: description,
				InputSchema: converted,
			}
			handler := createRawToolHandler(fn)
			server.AddTool(tool, handler)
			return nil
		}

		cfg.tools = append(cfg.tools, registerFunc)

		return nil
	}
}
