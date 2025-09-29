package mcpio

import (
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
			return ErrEmptyName
		}
		cfg.name = name
		return nil
	}
}

// WithVersion sets the server version
func WithVersion(version string) Option {
	return func(cfg *handlerConfig) error {
		if version == "" {
			return ErrEmptyVersion
		}
		cfg.version = version
		return nil
	}
}

// WithTool adds a type-safe tool with automatic schema generation from Go types.
// Schema options (WithInputSchema, WithOutputSchema, WithSchemas) completely
// override the auto-generated schemas from TIn/TOut types.
//
// Examples:
//
//	// Auto-generated schemas: Uses TIn/TOut struct types
//	WithTool("to_upper", "Convert text to uppercase", toUpperFunc)
//
//	// Schema override: Replaces auto-generated input schema
//	WithTool("to_upper", "Convert text", toUpperFunc,
//	    WithInputSchema(`{"type":"object","properties":{"text":{"type":"string"}}}`))
//
//	// Schema override: Replaces both auto-generated schemas
//	WithTool("calc", "Calculator", calcFunc,
//	    WithSchemas(inputSchemaJSON, outputSchemaJSON))
func WithTool[TIn, TOut any](name, description string, fn ToolFunc[TIn, TOut], opts ...ToolOption) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return ErrEmptyToolName
		}

		if cfg.tools == nil {
			return ErrIncompatibleHandler
		}

		// Create registration function that uses the generic AddTool
		registerFunc := func(server *mcp.Server) error {
			tool := &mcp.Tool{
				Name:        name,
				Description: description,
				// Let the generic AddTool handle schema generation initially
			}

			// Apply any custom schema options after tool creation but before registration
			for _, opt := range opts {
				if err := opt.apply(tool); err != nil {
					return fmt.Errorf("failed to apply tool option for %q: %w", name, err)
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

// WithRawTool adds a tool with manual JSON handling and explicit schema.
// Use this when you need direct control over JSON processing or dynamic schemas.
//
//	rawFunc := func(ctx context.Context, input []byte) ([]byte, error) { return input, nil }
//	WithRawTool("process", "Process JSON", `{"type":"object"}`, rawFunc)
func WithRawTool(name, description string, inputSchema any, fn RawToolFunc) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return ErrEmptyToolName
		}
		if inputSchema == nil {
			return ErrNilSchema
		}
		// Check for typed nil pointers
		if schema, ok := inputSchema.(*jsonschema.Schema); ok && schema == nil {
			return ErrNilSchema
		}

		if cfg.tools == nil {
			return ErrIncompatibleHandler
		}

		// Create registration function that uses the low-level AddTool
		registerFunc := func(server *mcp.Server) error {
			tool := &mcp.Tool{
				Name:        name,
				Description: description,
				InputSchema: inputSchema,
			}
			handler := createRawToolHandler(fn)
			server.AddTool(tool, handler)
			return nil
		}

		cfg.tools = append(cfg.tools, registerFunc)

		return nil
	}
}

// WithPrompt adds a prompt to the handler
func WithPrompt(name, description string, fn PromptFunc) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return ErrEmptyPromptName
		}

		if cfg.prompts == nil {
			return ErrIncompatibleHandler
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
			return ErrEmptyPromptName
		}

		if cfg.prompts == nil {
			return ErrIncompatibleHandler
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

// schemaToPromptArguments converts a JSON schema to MCP prompt arguments
func schemaToPromptArguments(schema any) []*mcp.PromptArgument {
	if schema == nil {
		return nil
	}

	// Convert to *jsonschema.Schema to access Properties and Required fields
	jsonSchemaObj, err := convertToJSONSchema(schema)
	if err != nil || jsonSchemaObj == nil || jsonSchemaObj.Properties == nil {
		return nil
	}

	var args []*mcp.PromptArgument
	requiredMap := make(map[string]bool)

	// Create a map of required fields
	for _, field := range jsonSchemaObj.Required {
		requiredMap[field] = true
	}

	// Convert schema properties to prompt arguments
	for name, propSchema := range jsonSchemaObj.Properties {
		arg := &mcp.PromptArgument{
			Name:        name,
			Description: propSchema.Description,
			Required:    requiredMap[name],
		}
		args = append(args, arg)
	}

	return args
}

// WithTypedPrompt adds a type-safe prompt with automatic schema generation
func WithTypedPrompt[TArgs any](name, description string, fn TypedPromptFunc[TArgs]) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return ErrEmptyPromptName
		}

		if cfg.prompts == nil {
			return ErrIncompatibleHandler
		}

		// Generate schema from the TArgs type
		s, err := schema.New[TArgs]()
		if err != nil {
			return fmt.Errorf("failed to generate schema for prompt %s: %w", name, err)
		}

		// Convert schema to prompt arguments
		args := schemaToPromptArguments(s)

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
			return ErrEmptyResourceURI
		}

		if cfg.resources == nil {
			return ErrIncompatibleHandler
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
			return ErrEmptyResourceTemplate
		}

		if cfg.resourceTemplates == nil {
			return ErrIncompatibleHandler
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

// WithServer allows injecting a custom server for testing
func WithServer(server *mcp.Server) Option {
	return func(cfg *handlerConfig) error {
		if server == nil {
			return ErrNilServer
		}
		cfg.server = server
		return nil
	}
}
