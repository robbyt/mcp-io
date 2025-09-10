// Package mcpio provides a minimal wrapper around the MCP SDK for dynamic tool registration
package mcpio

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Option is a functional option for configuring handlers
type Option func(*handlerConfig) error

// RawToolFunc is the function signature for raw JSON tools
type RawToolFunc func(context.Context, []byte) ([]byte, error)

// ScriptEvaluator abstracts different script engines for dynamic tool execution
type ScriptEvaluator interface {
	Execute(ctx context.Context, input []byte) ([]byte, error)
	GetTimeout() time.Duration
}

// handlerConfig holds the configuration built by options
type handlerConfig struct {
	name    string
	version string
	tools   []*toolRegistration
	server  *mcp.Server // The MCP server instance
}

// toolRegistration holds a function to register a tool on the server
type toolRegistration struct {
	registerFunc func(*mcp.Server)
}
