//go:build integration

package mcpwrapper

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerLoggerIntegration(t *testing.T) {
	t.Run("server logger captures MCP protocol events", func(t *testing.T) {
		var logBuf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		serverOpts := &mcpSDK.ServerOptions{}
		WithServerLogger(logger)(serverOpts)

		sdkServer := mcpSDK.NewServer(impl, serverOpts)
		server, clientTransport := NewInMemoryServer(sdkServer)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		errChan := make(chan error, 1)
		go func() {
			errChan <- server.Run(ctx)
		}()

		client := mcpSDK.NewClient(impl, &mcpSDK.ClientOptions{})
		_, err := client.Connect(ctx, clientTransport, nil)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		cancel()
		<-errChan

		logs := logBuf.String()
		assert.NotEmpty(t, logs, "Expected server logger to capture events")
	})
}

func TestHTTPLoggerIntegration(t *testing.T) {
	t.Run("HTTP logger captures transport events", func(t *testing.T) {
		var logBuf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		httpOpts := &mcpSDK.StreamableHTTPOptions{}
		err := WithHTTPLogger(logger)(httpOpts)
		require.NoError(t, err)

		server := New(sdkServer, WithHTTPOptions(httpOpts))
		assert.NotNil(t, server)

		logs := logBuf.String()
		assert.NotNil(t, logs)
	})
}

func TestSeparateLoggers(t *testing.T) {
	t.Run("server and HTTP loggers can be different", func(t *testing.T) {
		var serverLogBuf bytes.Buffer
		var httpLogBuf bytes.Buffer

		serverLogger := slog.New(slog.NewTextHandler(&serverLogBuf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		httpLogger := slog.New(slog.NewTextHandler(&httpLogBuf, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}

		serverOpts := &mcpSDK.ServerOptions{}
		WithServerLogger(serverLogger)(serverOpts)

		sdkServer := mcpSDK.NewServer(impl, serverOpts)

		httpOpts := &mcpSDK.StreamableHTTPOptions{}
		err := WithHTTPLogger(httpLogger)(httpOpts)
		require.NoError(t, err)

		server := New(sdkServer, WithHTTPOptions(httpOpts))
		assert.NotNil(t, server)

		assert.Equal(t, serverLogger, serverOpts.Logger)
		assert.Equal(t, httpLogger, httpOpts.Logger)
	})
}

func TestEventStoreIntegration(t *testing.T) {
	t.Run("event store enables stream resumption", func(t *testing.T) {
		eventStore := mcpSDK.NewMemoryEventStore(nil)

		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		httpOpts := &mcpSDK.StreamableHTTPOptions{}
		err := WithEventStore(eventStore)(httpOpts)
		require.NoError(t, err)

		server := New(sdkServer, WithHTTPOptions(httpOpts))
		assert.NotNil(t, server)
		assert.NotNil(t, server.httpOpts.EventStore)
	})
}

func TestSessionTimeoutIntegration(t *testing.T) {
	t.Run("session timeout is configured", func(t *testing.T) {
		timeout := 5 * time.Minute

		impl := &mcpSDK.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		}
		sdkServer := mcpSDK.NewServer(impl, &mcpSDK.ServerOptions{})

		httpOpts := &mcpSDK.StreamableHTTPOptions{}
		err := WithHTTPSessionTimeout(timeout)(httpOpts)
		require.NoError(t, err)

		server := New(sdkServer, WithHTTPOptions(httpOpts))
		assert.NotNil(t, server)
		assert.Equal(t, timeout, server.httpOpts.SessionTimeout)
	})
}
