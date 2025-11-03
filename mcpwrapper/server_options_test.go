package mcpwrapper

import (
	"log/slog"
	"os"
	"testing"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

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
