package mcpio_test

import (
	"log/slog"
	"testing"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetSession(t *testing.T) {
	t.Parallel()

	t.Run("WithSession", func(t *testing.T) {
		mockSession := new(testutil.MockSessionCapability)
		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)

		session := mcpio.GetSession(ctx)
		assert.NotNil(t, session)
		assert.Equal(t, mockSession, session)
	})

	t.Run("WithoutSession", func(t *testing.T) {
		ctx := t.Context()
		session := mcpio.GetSession(ctx)
		assert.Nil(t, session)
	})
}

func TestGetSessionID(t *testing.T) {
	t.Parallel()

	t.Run("WithSession", func(t *testing.T) {
		mockSession := new(testutil.MockSessionCapability)
		mockSession.On("SessionID").Return("test-session-123")

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)
		sessionID := mcpio.GetSessionID(ctx)

		assert.Equal(t, "test-session-123", sessionID)
		mockSession.AssertExpectations(t)
	})

	t.Run("WithoutSession", func(t *testing.T) {
		ctx := t.Context()
		sessionID := mcpio.GetSessionID(ctx)
		assert.Empty(t, sessionID)
	})
}

func TestCreateMessage(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := new(testutil.MockSessionCapability)
		mockSession.On("SupportsSampling").Return(true)
		mockSession.On("CreateMessage", mock.Anything, mock.Anything, 2000).Return(
			&capabilities.MessageResult{
				Role:    "assistant",
				Content: capabilities.TextContent{Text: "Analysis result"},
			}, nil)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)

		result, err := mcpio.CreateMessage(ctx, []*capabilities.Message{{
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
		ctx := t.Context()
		result, err := mcpio.CreateMessage(ctx, []*capabilities.Message{{
			Role:    "user",
			Content: "Test",
		}}, 1000)

		require.Error(t, err)
		require.ErrorIs(t, err, mcpio.ErrNoSession)
		assert.Nil(t, result)
	})

	t.Run("SamplingNotSupported", func(t *testing.T) {
		mockSession := new(testutil.MockSessionCapability)
		mockSession.On("SupportsSampling").Return(false)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)
		result, err := mcpio.CreateMessage(ctx, []*capabilities.Message{{
			Role:    "user",
			Content: "Test",
		}}, 1000)

		require.Error(t, err)
		require.ErrorIs(t, err, mcpio.ErrSamplingNotSupported)
		assert.Nil(t, result)
		mockSession.AssertExpectations(t)
	})
}

func TestListRoots(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := new(testutil.MockSessionCapability)
		expectedRoots := []*capabilities.Root{
			{URI: "file:///project", Name: "project"},
			{URI: "file:///src", Name: "src"},
		}
		mockSession.On("ListRoots", mock.Anything).Return(expectedRoots, nil)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)
		roots, err := mcpio.ListRoots(ctx)

		require.NoError(t, err)
		assert.Equal(t, expectedRoots, roots)
		mockSession.AssertExpectations(t)
	})

	t.Run("NoSession", func(t *testing.T) {
		ctx := t.Context()
		roots, err := mcpio.ListRoots(ctx)

		require.Error(t, err)
		require.ErrorIs(t, err, mcpio.ErrNoSession)
		assert.Nil(t, roots)
	})
}

func TestNotifyProgress(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := new(testutil.MockSessionCapability)
		mockSession.On("NotifyProgress", mock.Anything, 0.5, 1.0).Return(nil)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)
		err := mcpio.NotifyProgress(ctx, 0.5, 1.0)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("NoSession", func(t *testing.T) {
		ctx := t.Context()
		err := mcpio.NotifyProgress(ctx, 0.5, 1.0)

		require.Error(t, err)
		assert.ErrorIs(t, err, mcpio.ErrNoSession)
	})
}

func TestLogInfo(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := new(testutil.MockSessionCapability)
		mockSession.On("Log", mock.Anything, capabilities.LogLevelInfo, "test message", mock.Anything).Return(nil)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)
		err := mcpio.LogInfo(ctx, "test message", map[string]any{"key": "value"})

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("NoSession", func(t *testing.T) {
		ctx := t.Context()
		err := mcpio.LogInfo(ctx, "test message", nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, mcpio.ErrNoSession)
	})
}

func TestLogWarn(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := new(testutil.MockSessionCapability)
		mockSession.On("Log", mock.Anything, capabilities.LogLevelWarning, "warning", mock.Anything).Return(nil)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)
		err := mcpio.LogWarn(ctx, "warning", nil)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("NoSession", func(t *testing.T) {
		ctx := t.Context()
		err := mcpio.LogWarn(ctx, "warning", nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, mcpio.ErrNoSession)
	})
}

func TestLogError(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := new(testutil.MockSessionCapability)
		mockSession.On("Log", mock.Anything, capabilities.LogLevelError, "error", mock.Anything).Return(nil)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)
		err := mcpio.LogError(ctx, "error", nil)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("NoSession", func(t *testing.T) {
		ctx := t.Context()
		err := mcpio.LogError(ctx, "error", nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, mcpio.ErrNoSession)
	})
}

func TestLogDebug(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := new(testutil.MockSessionCapability)
		mockSession.On("Log", mock.Anything, capabilities.LogLevelDebug, "debug", mock.Anything).Return(nil)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)
		err := mcpio.LogDebug(ctx, "debug", nil)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("NoSession", func(t *testing.T) {
		ctx := t.Context()
		err := mcpio.LogDebug(ctx, "debug", nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, mcpio.ErrNoSession)
	})
}

func TestGetLogger(t *testing.T) {
	t.Parallel()

	t.Run("WithSession", func(t *testing.T) {
		mockSession := new(testutil.MockSessionCapability)
		expectedLogger := slog.Default()
		mockSession.On("Logger").Return(expectedLogger)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)
		logger := mcpio.GetLogger(ctx)

		assert.NotNil(t, logger)
		assert.Equal(t, expectedLogger, logger)
		mockSession.AssertExpectations(t)
	})

	t.Run("WithoutSession", func(t *testing.T) {
		ctx := t.Context()
		logger := mcpio.GetLogger(ctx)
		assert.Nil(t, logger)
	})
}

func TestInjectSessionForTesting(t *testing.T) {
	t.Parallel()

	mockSession := new(testutil.MockSessionCapability)
	ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)

	retrieved := mcpio.GetSession(ctx)
	assert.Equal(t, mockSession, retrieved)
}

func TestMultiTurnSampling(t *testing.T) {
	t.Parallel()

	mockSession := new(testutil.MockSessionCapability)
	mockSession.On("SupportsSampling").Return(true)
	mockSession.On("CreateMessage", mock.Anything, mock.Anything, 1000).Return(
		&capabilities.MessageResult{
			Role:    "assistant",
			Content: capabilities.TextContent{Text: "First response"},
		}, nil).Once()
	mockSession.On("CreateMessage", mock.Anything, mock.Anything, 1000).Return(
		&capabilities.MessageResult{
			Role:    "assistant",
			Content: capabilities.TextContent{Text: "Second response"},
		}, nil).Once()
	mockSession.On("CreateMessage", mock.Anything, mock.Anything, 1000).Return(
		&capabilities.MessageResult{
			Role:    "assistant",
			Content: capabilities.TextContent{Text: "Third response"},
		}, nil).Once()

	ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)

	result1, err := mcpio.CreateMessage(ctx, []*capabilities.Message{{
		Role:    "user",
		Content: "First question",
	}}, 1000)
	require.NoError(t, err)
	assert.Equal(t, "First response", result1.Content.Text)

	result2, err := mcpio.CreateMessage(ctx, []*capabilities.Message{{
		Role:    "user",
		Content: "Second question",
	}}, 1000)
	require.NoError(t, err)
	assert.Equal(t, "Second response", result2.Content.Text)

	result3, err := mcpio.CreateMessage(ctx, []*capabilities.Message{{
		Role:    "user",
		Content: "Third question",
	}}, 1000)
	require.NoError(t, err)
	assert.Equal(t, "Third response", result3.Content.Text)

	mockSession.AssertExpectations(t)
}

func TestSupportsSamplingCheck(t *testing.T) {
	t.Parallel()

	t.Run("SamplingSupported", func(t *testing.T) {
		mockSession := new(testutil.MockSessionCapability)
		mockSession.On("SupportsSampling").Return(true)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)
		session := mcpio.GetSession(ctx)

		require.NotNil(t, session)
		assert.True(t, session.SupportsSampling())
		mockSession.AssertExpectations(t)
	})

	t.Run("SamplingNotSupported", func(t *testing.T) {
		mockSession := new(testutil.MockSessionCapability)
		mockSession.On("SupportsSampling").Return(false)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession)
		session := mcpio.GetSession(ctx)

		require.NotNil(t, session)
		assert.False(t, session.SupportsSampling())
		mockSession.AssertExpectations(t)
	})
}
