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

func TestNotifyProgress(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(context.Background(), 0.5, 1.0)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("Success_ZeroProgress", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(context.Background(), 0.0, 1.0)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("Success_FullProgress", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(context.Background(), 1.0, 1.0)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedErr := assert.AnError
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(expectedErr)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(context.Background(), 0.5, 1.0)

		require.Error(t, err)
		require.ErrorIs(t, err, expectedErr)
		mockSession.AssertExpectations(t)
	})
}
