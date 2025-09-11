package mcpio

import (
	"context"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolFunc is the function signature for typed tools with automatic schema generation.
// The function receives a context and typed input, and returns typed output with an optional error.
// Schema generation is handled automatically based on the TIn and TOut types.
type ToolFunc[TIn, TOut any] func(context.Context, TIn) (TOut, error)

// RawToolFunc is the function signature for raw JSON tools.
// The function receives a context and raw JSON bytes as input, and returns JSON bytes as output.
// Schema must be provided explicitly when using WithRawTool.
type RawToolFunc func(context.Context, []byte) ([]byte, error)

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
			handler := createRawHandler(fn)
			server.AddTool(tool, handler)
		}

		cfg.tools = append(cfg.tools, registerFunc)

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
