package capabilities_test

import (
	"context"
	"testing"

	"github.com/robbyt/mcp-io/capabilities"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLogLevel_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, capabilities.LogLevelDebug, capabilities.LogLevel("debug"))
	assert.Equal(t, capabilities.LogLevelInfo, capabilities.LogLevel("info"))
	assert.Equal(t, capabilities.LogLevelNotice, capabilities.LogLevel("notice"))
	assert.Equal(t, capabilities.LogLevelWarning, capabilities.LogLevel("warning"))
	assert.Equal(t, capabilities.LogLevelError, capabilities.LogLevel("error"))
	assert.Equal(t, capabilities.LogLevelCritical, capabilities.LogLevel("critical"))
	assert.Equal(t, capabilities.LogLevelAlert, capabilities.LogLevel("alert"))
	assert.Equal(t, capabilities.LogLevelEmergency, capabilities.LogLevel("emergency"))
}

func TestLog(t *testing.T) {
	t.Parallel()

	t.Run("Success_WithData", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("Log", mock.Anything, mock.Anything).Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.Log(context.Background(), capabilities.LogLevelInfo, "test message", map[string]any{"key": "value"})

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("Success_NilData", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("Log", mock.Anything, mock.Anything).Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.Log(context.Background(), capabilities.LogLevelInfo, "test message", nil)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("AllLogLevels", func(t *testing.T) {
		levels := []capabilities.LogLevel{
			capabilities.LogLevelDebug,
			capabilities.LogLevelInfo,
			capabilities.LogLevelNotice,
			capabilities.LogLevelWarning,
			capabilities.LogLevelError,
			capabilities.LogLevelCritical,
			capabilities.LogLevelAlert,
			capabilities.LogLevelEmergency,
		}

		for _, level := range levels {
			mockSession := testutil.NewMockSession()
			mockSession.On("Log", mock.Anything, mock.Anything).Return(nil)

			session := capabilities.NewSession(mockSession.MockServerSession)
			err := session.Log(context.Background(), level, "test message", nil)

			require.NoError(t, err)
			mockSession.AssertExpectations(t)
		}
	})

	t.Run("Error", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedErr := assert.AnError
		mockSession.On("Log", mock.Anything, mock.Anything).Return(expectedErr)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.Log(context.Background(), capabilities.LogLevelError, "error message", nil)

		require.Error(t, err)
		require.ErrorIs(t, err, expectedErr)
		mockSession.AssertExpectations(t)
	})
}
