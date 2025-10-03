package mcpio

import (
	"errors"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/schema"
)

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

		if cfg.tools == nil {
			return ErrIncompatibleHandler
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

		if cfg.tools == nil {
			return ErrIncompatibleHandler
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
