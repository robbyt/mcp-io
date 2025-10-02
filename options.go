package mcpio

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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
