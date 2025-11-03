package capabilities_test

import (
	"testing"

	"github.com/robbyt/mcp-io/capabilities"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNotifyProgress(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 0.5, 1.0)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("Success_ZeroProgress", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 0.0, 1.0)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("Success_FullProgress", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 1.0, 1.0)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedErr := assert.AnError
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(expectedErr)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 0.5, 1.0)

		require.Error(t, err)
		require.ErrorIs(t, err, expectedErr)
		mockSession.AssertExpectations(t)
	})
}

func TestNotifyProgressWithOptions(t *testing.T) {
	t.Parallel()

	t.Run("WithToken_Int", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 0.5, 1.0, capabilities.WithProgressToken(123))

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("WithToken_String", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 0.75, 1.0, capabilities.WithProgressToken("progress-token-uuid"))

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("WithMessage", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 5.0, 10.0, capabilities.WithProgressMessage("Processing file 5 of 10"))

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("WithTokenAndMessage", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 5.0, 10.0,
			capabilities.WithProgressToken("token-123"),
			capabilities.WithProgressMessage("Processing file 5 of 10"))

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("WithMeta", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 0.5, 1.0,
			capabilities.WithProgressMeta(map[string]any{"custom": "data"}))

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("AllOptions", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 7.0, 10.0,
			capabilities.WithProgressToken(789),
			capabilities.WithProgressMessage("Almost done"),
			capabilities.WithProgressMeta(map[string]any{"step": "finalize"}))

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})
}
