package mcpwrapper

import (
	"log/slog"
	"os"
	"testing"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

func TestCapabilityOptions_PreserveCallerValues(t *testing.T) {
	t.Parallel()

	t.Run("WithCapabilityPrompts preserves pre-set ListChanged", func(t *testing.T) {
		opts := &mcpSDK.ServerOptions{
			Capabilities: &mcpSDK.ServerCapabilities{
				Prompts: &mcpSDK.PromptCapabilities{ListChanged: true},
			},
		}
		WithCapabilityPrompts()(opts)

		assert.True(t, opts.Capabilities.Prompts.ListChanged)
	})

	t.Run("WithCapabilityResources preserves pre-set ListChanged and Subscribe", func(t *testing.T) {
		opts := &mcpSDK.ServerOptions{
			Capabilities: &mcpSDK.ServerCapabilities{
				Resources: &mcpSDK.ResourceCapabilities{ListChanged: true, Subscribe: true},
			},
		}
		WithCapabilityResources()(opts)

		assert.True(t, opts.Capabilities.Resources.ListChanged)
		assert.True(t, opts.Capabilities.Resources.Subscribe)
	})

	t.Run("WithCapabilityTools preserves pre-set ListChanged", func(t *testing.T) {
		opts := &mcpSDK.ServerOptions{
			Capabilities: &mcpSDK.ServerCapabilities{
				Tools: &mcpSDK.ToolCapabilities{ListChanged: true},
			},
		}
		WithCapabilityTools()(opts)

		assert.True(t, opts.Capabilities.Tools.ListChanged)
	})

	t.Run("ensureServerCapabilities does not seed Logging into caller-provided Capabilities", func(t *testing.T) {
		opts := &mcpSDK.ServerOptions{
			Capabilities: &mcpSDK.ServerCapabilities{},
		}
		WithCapabilityPrompts()(opts)

		assert.Nil(t, opts.Capabilities.Logging)
	})
}

func TestWithServerLogger(t *testing.T) {
	t.Run("sets logger on server options", func(t *testing.T) {
		logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		opts := &mcpSDK.ServerOptions{}
		option := WithServerLogger(logger)
		option(opts)

		assert.Equal(t, logger, opts.Logger)
	})

	t.Run("nil logger is accepted", func(t *testing.T) {
		opts := &mcpSDK.ServerOptions{}
		option := WithServerLogger(nil)
		option(opts)

		assert.Nil(t, opts.Logger)
	})
}
