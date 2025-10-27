package mcpio_test

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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
		mockSession := testutil.NewMockSession()
		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)

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

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)
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
		mockSession := testutil.NewMockSession()
		mockSession.SetupSampling()
		mockSession.On("CreateMessage", mock.Anything, mock.Anything).Return(
			&mcp.CreateMessageResult{
				Role:    "assistant",
				Content: &mcp.TextContent{Text: "Analysis result"},
			}, nil)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)

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
		mockSession := testutil.NewMockSession()
		mockSession.SetupNoCapabilities()

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)
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
		mockSession := testutil.NewMockSession()
		mockSession.On("ListRoots", mock.Anything, mock.Anything).Return(&mcp.ListRootsResult{
			Roots: []*mcp.Root{
				{URI: "file:///project", Name: "project"},
				{URI: "file:///src", Name: "src"},
			},
		}, nil)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)
		roots, err := mcpio.ListRoots(ctx)

		require.NoError(t, err)
		assert.Len(t, roots, 2)
		assert.Equal(t, "file:///project", roots[0].URI)
		assert.Equal(t, "project", roots[0].Name)
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
		mockSession := testutil.NewMockSession()
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(nil)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)
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
		mockSession := testutil.NewMockSession()
		mockSession.On("Log", mock.Anything, mock.Anything).Return(nil)

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)
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

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)
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

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)
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

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)
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

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)
		logger := mcpio.GetLogger(ctx)

		assert.NotNil(t, logger)
	})

	t.Run("WithoutSession", func(t *testing.T) {
		ctx := t.Context()
		logger := mcpio.GetLogger(ctx)
		assert.Nil(t, logger)
	})
}

func TestInjectSessionForTesting(t *testing.T) {
	t.Parallel()

	mockSession := testutil.NewMockSession()
	ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)

	retrieved := mcpio.GetSession(ctx)
	assert.Equal(t, mockSession.Session, retrieved)
}

func TestMultiTurnSampling(t *testing.T) {
	t.Parallel()

	mockSession := testutil.NewMockSession()
	mockSession.SetupSampling()
	mockSession.On("CreateMessage", mock.Anything, mock.Anything).Return(
		&mcp.CreateMessageResult{
			Role:    "assistant",
			Content: &mcp.TextContent{Text: "First response"},
		}, nil).Once()
	mockSession.On("CreateMessage", mock.Anything, mock.Anything).Return(
		&mcp.CreateMessageResult{
			Role:    "assistant",
			Content: &mcp.TextContent{Text: "Second response"},
		}, nil).Once()
	mockSession.On("CreateMessage", mock.Anything, mock.Anything).Return(
		&mcp.CreateMessageResult{
			Role:    "assistant",
			Content: &mcp.TextContent{Text: "Third response"},
		}, nil).Once()

	ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)

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
		mockSession := testutil.NewMockSession()
		mockSession.SetupSampling()

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)
		session := mcpio.GetSession(ctx)

		require.NotNil(t, session)
		assert.True(t, session.SupportsSampling())
		mockSession.AssertExpectations(t)
	})

	t.Run("SamplingNotSupported", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.SetupNoCapabilities()

		ctx := mcpio.InjectSessionForTesting(t.Context(), mockSession.Session)
		session := mcpio.GetSession(ctx)

		require.NotNil(t, session)
		assert.False(t, session.SupportsSampling())
		mockSession.AssertExpectations(t)
	})
}
