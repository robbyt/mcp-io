package capabilities_test

import (
	"testing"

	"github.com/robbyt/mcp-io/capabilities"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionCapability(t *testing.T) {
	t.Parallel()

	t.Run("WithMockSession", func(t *testing.T) {
		mockSession := testutil.NewMockSession()

		session := capabilities.NewSessionCapability(mockSession.MockServerSession)

		assert.NotNil(t, session)
		assert.NotNil(t, session.Logger())
	})
}

func TestSession_Logger(t *testing.T) {
	t.Parallel()

	t.Run("WithMockSession", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		session := capabilities.NewSessionCapability(mockSession.MockServerSession)

		logger := session.Logger()

		assert.NotNil(t, logger)
	})
}

func TestSession_SessionID(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsID", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("ID").Return("test-session-123")

		session := capabilities.NewSessionCapability(mockSession.MockServerSession)
		sessionID := session.SessionID()

		assert.Equal(t, "test-session-123", sessionID)
		mockSession.AssertExpectations(t)
	})
}

func TestSession_Wait(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("Wait").Return(nil)

		session := capabilities.NewSessionCapability(mockSession.MockServerSession)
		err := session.Wait()

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedErr := assert.AnError
		mockSession.On("Wait").Return(expectedErr)

		session := capabilities.NewSessionCapability(mockSession.MockServerSession)
		err := session.Wait()

		require.ErrorIs(t, err, expectedErr)
		mockSession.AssertExpectations(t)
	})
}

func TestSession_Close(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.On("Close").Return(nil)

		session := capabilities.NewSessionCapability(mockSession.MockServerSession)
		err := session.Close()

		require.NoError(t, err)
		mockSession.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedErr := assert.AnError
		mockSession.On("Close").Return(expectedErr)

		session := capabilities.NewSessionCapability(mockSession.MockServerSession)
		err := session.Close()

		require.ErrorIs(t, err, expectedErr)
		mockSession.AssertExpectations(t)
	})
}
