package mcpio

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// WithTool adds a type-safe tool with automatic schema generation
func WithTool[TIn, TOut any](name, description string, fn ToolFunc[TIn, TOut]) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return ErrEmptyToolName
		}

		// Create registration function that uses the generic AddTool
		registerFunc := func(server *mcp.Server) {
			tool := &mcp.Tool{
				Name:        name,
				Description: description,
				// Let the generic AddTool handle schema generation
			}
			handler := createTypedHandler(fn)
			mcp.AddTool(server, tool, handler)
		}

		cfg.tools = append(cfg.tools, registerFunc)

		return nil
	}
}

// WithRawTool adds a tool with manual JSON handling and explicit schema
func WithRawTool(name, description string, inputSchema *jsonschema.Schema, fn RawToolFunc) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return ErrEmptyToolName
		}
		if inputSchema == nil {
			return ErrNilSchema
		}

		// Create registration function that uses the low-level AddTool
		registerFunc := func(server *mcp.Server) {
			tool := &mcp.Tool{
				Name:        name,
				Description: description,
				InputSchema: inputSchema,
			}
			handler := createRawToolHandler(fn)
			server.AddTool(tool, handler)
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

		registerFunc := func(server *mcp.Server) {
			prompt := &mcp.Prompt{
				Name:        name,
				Description: description,
			}
			handler := createPromptHandler(fn)
			server.AddPrompt(prompt, handler)
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

		registerFunc := func(server *mcp.Server) {
			prompt := &mcp.Prompt{
				Name:        name,
				Description: description,
				Arguments:   args,
			}
			handler := createPromptHandler(fn)
			server.AddPrompt(prompt, handler)
		}

		cfg.prompts = append(cfg.prompts, registerFunc)
		return nil
	}
}

// schemaToPromptArguments converts a JSON schema to MCP prompt arguments
func schemaToPromptArguments(schema *jsonschema.Schema) []*mcp.PromptArgument {
	if schema == nil || schema.Properties == nil {
		return nil
	}

	var args []*mcp.PromptArgument
	requiredMap := make(map[string]bool)

	// Create a map of required fields
	for _, field := range schema.Required {
		requiredMap[field] = true
	}

	// Convert schema properties to prompt arguments
	for name, propSchema := range schema.Properties {
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

		// Generate schema from the TArgs type
		schema, err := GenerateSchema[TArgs]()
		if err != nil {
			return fmt.Errorf("failed to generate schema for prompt %s: %w", name, err)
		}

		// Convert schema to prompt arguments
		args := schemaToPromptArguments(schema)

		registerFunc := func(server *mcp.Server) {
			prompt := &mcp.Prompt{
				Name:        name,
				Description: description,
				Arguments:   args,
			}
			handler := createTypedPromptHandler(fn)
			server.AddPrompt(prompt, handler)
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

		registerFunc := func(server *mcp.Server) {
			resource := &mcp.Resource{
				URI:         uri,
				Name:        uri, // Use URI as name by default
				Description: description,
			}
			handler := createResourceHandler(fn)
			server.AddResource(resource, handler)
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

		registerFunc := func(server *mcp.Server) {
			template := &mcp.ResourceTemplate{
				URITemplate: uriTemplate,
				Name:        uriTemplate, // Use template as name by default
				Description: description,
			}
			handler := createResourceHandler(fn) // Same handler type
			server.AddResourceTemplate(template, handler)
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
