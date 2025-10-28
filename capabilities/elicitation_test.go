package capabilities_test

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSupportsElicitation(t *testing.T) {
	t.Parallel()

	t.Run("True", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()

		session := capabilities.NewSession(mockSession.MockServerSession)
		supported := session.SupportsElicitation()

		assert.True(t, supported)
		mockSession.AssertExpectations(t)
	})

	t.Run("False", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.SetupNoCapabilities()

		session := capabilities.NewSession(mockSession.MockServerSession)
		supported := session.SupportsElicitation()

		assert.False(t, supported)
		mockSession.AssertExpectations(t)
	})
}

func TestElicit(t *testing.T) {
	t.Parallel()

	t.Run("Success_Accept", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedResult := &mcp.ElicitResult{
			Action:  "accept",
			Content: map[string]any{"field": "value"},
		}
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(expectedResult, nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		result, err := session.Elicit(context.Background(), "Enter data", map[string]any{"type": "object"})

		require.NoError(t, err)
		assert.Equal(t, "accept", result.Action)
		assert.NotNil(t, result.Content)
		mockSession.AssertExpectations(t)
	})

	t.Run("Success_Decline", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedResult := &mcp.ElicitResult{
			Action:  "decline",
			Content: nil,
		}
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(expectedResult, nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		result, err := session.Elicit(context.Background(), "Enter data", map[string]any{"type": "object"})

		require.NoError(t, err)
		assert.Equal(t, "decline", result.Action)
		assert.Nil(t, result.Content)
		mockSession.AssertExpectations(t)
	})

	t.Run("Success_Cancel", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedResult := &mcp.ElicitResult{
			Action:  "cancel",
			Content: nil,
		}
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(expectedResult, nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		result, err := session.Elicit(context.Background(), "Enter data", map[string]any{"type": "object"})

		require.NoError(t, err)
		assert.Equal(t, "cancel", result.Action)
		assert.Nil(t, result.Content)
		mockSession.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedErr := assert.AnError
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(nil, expectedErr)

		session := capabilities.NewSession(mockSession.MockServerSession)
		result, err := session.Elicit(context.Background(), "Enter data", map[string]any{"type": "object"})

		require.Error(t, err)
		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, result)
		mockSession.AssertExpectations(t)
	})
}
