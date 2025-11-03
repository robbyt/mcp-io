package capabilities_test

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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
		var capturedParams *mcp.ProgressNotificationParams
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedParams = args.Get(1).(*mcp.ProgressNotificationParams)
			}).
			Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 0.5, 1.0)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
		assert.InEpsilon(t, 0.5, capturedParams.Progress, 0.0001)
		assert.InEpsilon(t, 1.0, capturedParams.Total, 0.0001)
		assert.Nil(t, capturedParams.ProgressToken)
		assert.Empty(t, capturedParams.Message)
	})

	t.Run("Success_ZeroProgress", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		var capturedParams *mcp.ProgressNotificationParams
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedParams = args.Get(1).(*mcp.ProgressNotificationParams)
			}).
			Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 0.0, 1.0)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
		assert.InDelta(t, 0.0, capturedParams.Progress, 0.0001)
		assert.InEpsilon(t, 1.0, capturedParams.Total, 0.0001)
	})

	t.Run("Success_FullProgress", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		var capturedParams *mcp.ProgressNotificationParams
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedParams = args.Get(1).(*mcp.ProgressNotificationParams)
			}).
			Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 1.0, 1.0)

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
		assert.InEpsilon(t, 1.0, capturedParams.Progress, 0.0001)
		assert.InEpsilon(t, 1.0, capturedParams.Total, 0.0001)
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
		var capturedParams *mcp.ProgressNotificationParams
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedParams = args.Get(1).(*mcp.ProgressNotificationParams)
			}).
			Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 0.5, 1.0, capabilities.WithProgressToken(123))

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
		assert.InEpsilon(t, 0.5, capturedParams.Progress, 0.0001)
		assert.InEpsilon(t, 1.0, capturedParams.Total, 0.0001)
		assert.Equal(t, 123, capturedParams.ProgressToken)
	})

	t.Run("WithToken_String", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		var capturedParams *mcp.ProgressNotificationParams
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedParams = args.Get(1).(*mcp.ProgressNotificationParams)
			}).
			Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 0.75, 1.0, capabilities.WithProgressToken("progress-token-uuid"))

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
		assert.InEpsilon(t, 0.75, capturedParams.Progress, 0.0001)
		assert.InEpsilon(t, 1.0, capturedParams.Total, 0.0001)
		assert.Equal(t, "progress-token-uuid", capturedParams.ProgressToken)
	})

	t.Run("WithMessage", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		var capturedParams *mcp.ProgressNotificationParams
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedParams = args.Get(1).(*mcp.ProgressNotificationParams)
			}).
			Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 5.0, 10.0, capabilities.WithProgressMessage("Processing file 5 of 10"))

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
		assert.InEpsilon(t, 5.0, capturedParams.Progress, 0.0001)
		assert.InEpsilon(t, 10.0, capturedParams.Total, 0.0001)
		assert.Equal(t, "Processing file 5 of 10", capturedParams.Message)
		assert.Nil(t, capturedParams.ProgressToken)
	})

	t.Run("WithTokenAndMessage", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		var capturedParams *mcp.ProgressNotificationParams
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedParams = args.Get(1).(*mcp.ProgressNotificationParams)
			}).
			Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 5.0, 10.0,
			capabilities.WithProgressToken("token-123"),
			capabilities.WithProgressMessage("Processing file 5 of 10"))

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
		assert.InEpsilon(t, 5.0, capturedParams.Progress, 0.0001)
		assert.InEpsilon(t, 10.0, capturedParams.Total, 0.0001)
		assert.Equal(t, "token-123", capturedParams.ProgressToken)
		assert.Equal(t, "Processing file 5 of 10", capturedParams.Message)
	})

	t.Run("WithMeta", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		var capturedParams *mcp.ProgressNotificationParams
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedParams = args.Get(1).(*mcp.ProgressNotificationParams)
			}).
			Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 0.5, 1.0,
			capabilities.WithProgressMeta(map[string]any{"custom": "data"}))

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
		assert.InEpsilon(t, 0.5, capturedParams.Progress, 0.0001)
		assert.InEpsilon(t, 1.0, capturedParams.Total, 0.0001)
		assert.Equal(t, mcp.Meta{"custom": "data"}, capturedParams.Meta)
		assert.Nil(t, capturedParams.ProgressToken)
	})

	t.Run("AllOptions", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		var capturedParams *mcp.ProgressNotificationParams
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedParams = args.Get(1).(*mcp.ProgressNotificationParams)
			}).
			Return(nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		err := session.NotifyProgress(t.Context(), 7.0, 10.0,
			capabilities.WithProgressToken(789),
			capabilities.WithProgressMessage("Almost done"),
			capabilities.WithProgressMeta(map[string]any{"step": "finalize"}))

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
		assert.InEpsilon(t, 7.0, capturedParams.Progress, 0.0001)
		assert.InEpsilon(t, 10.0, capturedParams.Total, 0.0001)
		assert.Equal(t, 789, capturedParams.ProgressToken)
		assert.Equal(t, "Almost done", capturedParams.Message)
		assert.Equal(t, mcp.Meta{"step": "finalize"}, capturedParams.Meta)
	})
}
