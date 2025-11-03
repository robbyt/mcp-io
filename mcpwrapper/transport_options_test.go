package mcpwrapper

import (
	"log/slog"
	"os"
	"testing"
	"time"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithHTTPLogger(t *testing.T) {
	t.Run("sets logger on HTTP options", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		opts := &mcpSDK.StreamableHTTPOptions{}
		option := WithHTTPLogger(logger)
		err := option(opts)

		require.NoError(t, err)
		assert.Equal(t, logger, opts.Logger)
	})

	t.Run("nil logger is accepted", func(t *testing.T) {
		opts := &mcpSDK.StreamableHTTPOptions{}
		option := WithHTTPLogger(nil)
		err := option(opts)

		require.NoError(t, err)
		assert.Nil(t, opts.Logger)
	})
}

func TestWithHTTPSessionTimeout(t *testing.T) {
	t.Run("sets session timeout", func(t *testing.T) {
		timeout := 5 * time.Minute

		opts := &mcpSDK.StreamableHTTPOptions{}
		option := WithHTTPSessionTimeout(timeout)
		err := option(opts)

		require.NoError(t, err)
		assert.Equal(t, timeout, opts.SessionTimeout)
	})

	t.Run("zero timeout is accepted", func(t *testing.T) {
		opts := &mcpSDK.StreamableHTTPOptions{}
		option := WithHTTPSessionTimeout(0)
		err := option(opts)

		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), opts.SessionTimeout)
	})
}

func TestWithEventStore(t *testing.T) {
	t.Run("sets event store", func(t *testing.T) {
		store := mcpSDK.NewMemoryEventStore(nil)

		opts := &mcpSDK.StreamableHTTPOptions{}
		option := WithEventStore(store)
		err := option(opts)

		require.NoError(t, err)
		assert.Equal(t, store, opts.EventStore)
	})

	t.Run("nil event store is accepted", func(t *testing.T) {
		opts := &mcpSDK.StreamableHTTPOptions{}
		option := WithEventStore(nil)
		err := option(opts)

		require.NoError(t, err)
		assert.Nil(t, opts.EventStore)
	})
}

func TestWithStateless(t *testing.T) {
	t.Run("sets stateless mode", func(t *testing.T) {
		opts := &mcpSDK.StreamableHTTPOptions{}
		option := WithStateless()
		err := option(opts)

		require.NoError(t, err)
		assert.True(t, opts.Stateless)
	})
}

func TestWithJSONResponse(t *testing.T) {
	t.Run("sets JSON response mode", func(t *testing.T) {
		opts := &mcpSDK.StreamableHTTPOptions{}
		option := WithJSONResponse()
		err := option(opts)

		require.NoError(t, err)
		assert.True(t, opts.JSONResponse)
	})
}
