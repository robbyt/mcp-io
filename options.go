package mcpio

import (
	"context"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
func WithTool[TIn, TOut any](name, description string, fn func(ctx context.Context, input TIn) (TOut, error)) Option {
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

		cfg.tools = append(cfg.tools, &toolRegistration{
			registerFunc: registerFunc,
		})

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
			handler := createRawHandler(fn)
			server.AddTool(tool, handler)
		}

		cfg.tools = append(cfg.tools, &toolRegistration{
			registerFunc: registerFunc,
		})

		return nil
	}
}

// WithScriptTool adds a tool backed by a script evaluator
func WithScriptTool(name, description string, evaluator ScriptEvaluator) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return ErrEmptyToolName
		}
		if evaluator == nil {
			return ErrNilEvaluator
		}

		// Create a basic schema for script tools (they accept any JSON object)
		inputSchema := &jsonschema.Schema{
			Type:        "object",
			Description: "Input data for script execution",
		}

		// Wrap the evaluator in a RawToolFunc
		rawFunc := func(ctx context.Context, input []byte) ([]byte, error) {
			return evaluator.Execute(ctx, input)
		}

		return WithRawTool(name, description, inputSchema, rawFunc)(cfg)
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
