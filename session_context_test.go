package mcpio_test

import (
	"testing"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetSession(t *testing.T) {
	t.Parallel()

	t.Run("WithSession", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		ctx := testutil.WithSession(t.Context(), mockSession.Session)

		session := mcpio.GetSession(ctx)
		assert.NotNil(t, session)
		assert.Equal(t, mockSession.Session, session)
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
		mockSession := testutil.NewMockSession()
		mockSession.On("ID").Return("test-session-123")

		ctx := testutil.WithSession(t.Context(), mockSession.Session)
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

func TestLogInfo(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("Log", mock.Anything, mock.Anything).Return(nil)

		ctx := testutil.WithSession(t.Context(), mockSession.Session)
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
		mockSession := testutil.NewMockSession()
		mockSession.On("Log", mock.Anything, mock.Anything).Return(nil)

		ctx := testutil.WithSession(t.Context(), mockSession.Session)
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
		mockSession := testutil.NewMockSession()
		mockSession.On("Log", mock.Anything, mock.Anything).Return(nil)

		ctx := testutil.WithSession(t.Context(), mockSession.Session)
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
		mockSession := testutil.NewMockSession()
		mockSession.On("Log", mock.Anything, mock.Anything).Return(nil)

		ctx := testutil.WithSession(t.Context(), mockSession.Session)
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
		mockSession := testutil.NewMockSession()

		ctx := testutil.WithSession(t.Context(), mockSession.Session)
		logger := mcpio.GetLogger(ctx)

		assert.NotNil(t, logger)
	})

	t.Run("WithoutSession", func(t *testing.T) {
		ctx := t.Context()
		logger := mcpio.GetLogger(ctx)
		assert.Nil(t, logger)
	})
}

func TestWithSession(t *testing.T) {
	t.Parallel()

	mockSession := testutil.NewMockSession()
	ctx := testutil.WithSession(t.Context(), mockSession.Session)

	retrieved := mcpio.GetSession(ctx)
	assert.Equal(t, mockSession.Session, retrieved)
}

func TestSupportsSamplingCheck(t *testing.T) {
	t.Parallel()

	t.Run("SamplingSupported", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.SetupSampling()

		ctx := testutil.WithSession(t.Context(), mockSession.Session)
		session := mcpio.GetSession(ctx)

		require.NotNil(t, session)
		assert.True(t, session.SupportsSampling())
		mockSession.AssertExpectations(t)
	})

	t.Run("SamplingNotSupported", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.SetupNoCapabilities()

		ctx := testutil.WithSession(t.Context(), mockSession.Session)
		session := mcpio.GetSession(ctx)

		require.NotNil(t, session)
		assert.False(t, session.SupportsSampling())
		mockSession.AssertExpectations(t)
	})
}
