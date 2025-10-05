package mcpio

import (
	"context"
	"log/slog"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockSession is a test mock for capabilities.SessionCapability
type MockSession struct {
	mock.Mock
}

func (m *MockSession) Elicit(ctx context.Context, message string, requestedSchema any) (*mcp.ElicitResult, error) {
	args := m.Called(ctx, message, requestedSchema)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mcp.ElicitResult), args.Error(1)
}

func (m *MockSession) CreateMessage(ctx context.Context, messages []*capabilities.Message, maxTokens int) (*capabilities.MessageResult, error) {
	args := m.Called(ctx, messages, maxTokens)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*capabilities.MessageResult), args.Error(1)
}

func (m *MockSession) CreateMessageRaw(ctx context.Context, params *mcp.CreateMessageParams) (*mcp.CreateMessageResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mcp.CreateMessageResult), args.Error(1)
}

func (m *MockSession) ListRoots(ctx context.Context) ([]*capabilities.Root, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*capabilities.Root), args.Error(1)
}

func (m *MockSession) Log(ctx context.Context, level capabilities.LogLevel, message string, data map[string]any) error {
	args := m.Called(ctx, level, message, data)
	return args.Error(0)
}

func (m *MockSession) Logger() *slog.Logger {
	args := m.Called()
	return args.Get(0).(*slog.Logger)
}

func (m *MockSession) NotifyProgress(ctx context.Context, progress, total float64) error {
	args := m.Called(ctx, progress, total)
	return args.Error(0)
}

func (m *MockSession) SessionID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSession) ClientCapabilities() *capabilities.ClientCapabilities {
	args := m.Called()
	return args.Get(0).(*capabilities.ClientCapabilities)
}

func (m *MockSession) SupportsElicitation() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSession) SupportsSampling() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSession) Wait() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSession) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestGetSession(t *testing.T) {
	t.Parallel()

	t.Run("WithSession", func(t *testing.T) {
		mockSession := new(MockSession)
		ctx := InjectSessionForTesting(context.Background(), mockSession)

		session := GetSession(ctx)
		assert.NotNil(t, session)
		assert.Equal(t, mockSession, session)
	})

	t.Run("WithoutSession", func(t *testing.T) {
		ctx := context.Background()
		session := GetSession(ctx)
		assert.Nil(t, session)
	})
}

func TestGetSessionID(t *testing.T) {
	t.Parallel()

	t.Run("WithSession", func(t *testing.T) {
		mockSession := new(MockSession)
		mockSession.On("SessionID").Return("test-session-123")

		ctx := InjectSessionForTesting(context.Background(), mockSession)
		sessionID := GetSessionID(ctx)

		assert.Equal(t, "test-session-123", sessionID)
		mockSession.AssertExpectations(t)
	})

	t.Run("WithoutSession", func(t *testing.T) {
		ctx := context.Background()
		sessionID := GetSessionID(ctx)
		assert.Empty(t, sessionID)
	})
}

func TestCreateMessage(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := new(MockSession)
		mockSession.On("SupportsSampling").Return(true)
		mockSession.On("CreateMessage", mock.Anything, mock.Anything, 2000).Return(
			&capabilities.MessageResult{
				Role:    "assistant",
				Content: capabilities.TextContent{Text: "Analysis result"},
			}, nil)

		ctx := InjectSessionForTesting(context.Background(), mockSession)

		result, err := CreateMessage(ctx, []*capabilities.Message{{
			Role:    "user",
			Content: "Analyze this code",
		}}, 2000)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "assistant", result.Role)
		assert.Equal(t, "Analysis result", result.Content.Text)
		mockSession.AssertExpectations(t)
	})

	t.Run("NoSession", func(t *testing.T) {
		ctx := context.Background()
		result, err := CreateMessage(ctx, []*capabilities.Message{{
			Role:    "user",
			Content: "Test",
		}}, 1000)

		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoSession)
		assert.Nil(t, result)
	})

	t.Run("SamplingNotSupported", func(t *testing.T) {
		mockSession := new(MockSession)
		mockSession.On("SupportsSampling").Return(false)

		ctx := InjectSessionForTesting(context.Background(), mockSession)
		result, err := CreateMessage(ctx, []*capabilities.Message{{
			Role:    "user",
			Content: "Test",
		}}, 1000)

		require.Error(t, err)
		require.ErrorIs(t, err, ErrSamplingNotSupported)
		assert.Nil(t, result)
		mockSession.AssertExpectations(t)
	})
}

func TestListRoots(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := new(MockSession)
		expectedRoots := []*capabilities.Root{
			{URI: "file:///project", Name: "project"},
			{URI: "file:///src", Name: "src"},
		}
		mockSession.On("ListRoots", mock.Anything).Return(expectedRoots, nil)

		ctx := InjectSessionForTesting(context.Background(), mockSession)
		roots, err := ListRoots(ctx)

		require.NoError(t, err)
		assert.Equal(t, expectedRoots, roots)
		mockSession.AssertExpectations(t)
	})

	t.Run("NoSession", func(t *testing.T) {
		ctx := context.Background()
		roots, err := ListRoots(ctx)

		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoSession)
		assert.Nil(t, roots)
	})
}

func TestNotifyProgress(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := new(MockSession)
		mockSession.On("NotifyProgress", mock.Anything, 0.5, 1.0).Return(nil)

		ctx := InjectSessionForTesting(context.Background(), mockSession)
		err := NotifyProgress(ctx, 0.5, 1.0)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("NoSession", func(t *testing.T) {
		ctx := context.Background()
		err := NotifyProgress(ctx, 0.5, 1.0)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoSession)
	})
}

func TestLogInfo(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := new(MockSession)
		mockSession.On("Log", mock.Anything, capabilities.LogLevelInfo, "test message", mock.Anything).Return(nil)

		ctx := InjectSessionForTesting(context.Background(), mockSession)
		err := LogInfo(ctx, "test message", map[string]any{"key": "value"})

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("NoSession", func(t *testing.T) {
		ctx := context.Background()
		err := LogInfo(ctx, "test message", nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoSession)
	})
}

func TestLogWarn(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := new(MockSession)
		mockSession.On("Log", mock.Anything, capabilities.LogLevelWarning, "warning", mock.Anything).Return(nil)

		ctx := InjectSessionForTesting(context.Background(), mockSession)
		err := LogWarn(ctx, "warning", nil)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("NoSession", func(t *testing.T) {
		ctx := context.Background()
		err := LogWarn(ctx, "warning", nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoSession)
	})
}

func TestLogError(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := new(MockSession)
		mockSession.On("Log", mock.Anything, capabilities.LogLevelError, "error", mock.Anything).Return(nil)

		ctx := InjectSessionForTesting(context.Background(), mockSession)
		err := LogError(ctx, "error", nil)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("NoSession", func(t *testing.T) {
		ctx := context.Background()
		err := LogError(ctx, "error", nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoSession)
	})
}

func TestLogDebug(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := new(MockSession)
		mockSession.On("Log", mock.Anything, capabilities.LogLevelDebug, "debug", mock.Anything).Return(nil)

		ctx := InjectSessionForTesting(context.Background(), mockSession)
		err := LogDebug(ctx, "debug", nil)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("NoSession", func(t *testing.T) {
		ctx := context.Background()
		err := LogDebug(ctx, "debug", nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoSession)
	})
}

func TestGetLogger(t *testing.T) {
	t.Parallel()

	t.Run("WithSession", func(t *testing.T) {
		mockSession := new(MockSession)
		expectedLogger := slog.Default()
		mockSession.On("Logger").Return(expectedLogger)

		ctx := InjectSessionForTesting(context.Background(), mockSession)
		logger := GetLogger(ctx)

		assert.NotNil(t, logger)
		assert.Equal(t, expectedLogger, logger)
		mockSession.AssertExpectations(t)
	})

	t.Run("WithoutSession", func(t *testing.T) {
		ctx := context.Background()
		logger := GetLogger(ctx)
		assert.Nil(t, logger)
	})
}

func TestInjectSessionForTesting(t *testing.T) {
	t.Parallel()

	mockSession := new(MockSession)
	ctx := InjectSessionForTesting(context.Background(), mockSession)

	retrieved := GetSession(ctx)
	assert.Equal(t, mockSession, retrieved)
}
