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

func TestSupportsSampling(t *testing.T) {
	t.Parallel()

	t.Run("True", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.SetupSampling()

		session := capabilities.NewSession(mockSession.MockServerSession)
		supported := session.SupportsSampling()

		assert.True(t, supported)
		mockSession.AssertExpectations(t)
	})

	t.Run("False", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		mockSession.SetupNoCapabilities()

		session := capabilities.NewSession(mockSession.MockServerSession)
		supported := session.SupportsSampling()

		assert.False(t, supported)
		mockSession.AssertExpectations(t)
	})
}

func TestCreateMessage(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedResult := &mcp.CreateMessageResult{
			Role:    "assistant",
			Content: &mcp.TextContent{Text: "Analysis result"},
		}
		mockSession.On("CreateMessage", mock.Anything, mock.Anything).Return(expectedResult, nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		result, err := session.CreateMessage(t.Context(), []*capabilities.Message{{
			Role:    "user",
			Content: "Analyze this code",
		}}, 2000)

		require.NoError(t, err)
		assert.Equal(t, "assistant", result.Role)
		assert.Equal(t, "Analysis result", result.Content.Text)
		mockSession.AssertExpectations(t)
	})

	t.Run("Success_MultipleMessages", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedResult := &mcp.CreateMessageResult{
			Role:    "assistant",
			Content: &mcp.TextContent{Text: "Response"},
		}
		mockSession.On("CreateMessage", mock.Anything, mock.Anything).Return(expectedResult, nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		result, err := session.CreateMessage(t.Context(), []*capabilities.Message{
			{Role: "user", Content: "First message"},
			{Role: "assistant", Content: "First response"},
			{Role: "user", Content: "Second message"},
		}, 1000)

		require.NoError(t, err)
		assert.NotNil(t, result)
		mockSession.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedErr := assert.AnError
		mockSession.On("CreateMessage", mock.Anything, mock.Anything).Return(nil, expectedErr)

		session := capabilities.NewSession(mockSession.MockServerSession)
		result, err := session.CreateMessage(t.Context(), []*capabilities.Message{{
			Role:    "user",
			Content: "Test",
		}}, 1000)

		require.Error(t, err)
		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, result)
		mockSession.AssertExpectations(t)
	})
}

func TestCreateMessageRaw(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedResult := &mcp.CreateMessageResult{
			Role:    "assistant",
			Content: &mcp.TextContent{Text: "Raw response"},
		}
		mockSession.On("CreateMessage", mock.Anything, mock.Anything).Return(expectedResult, nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		params := &mcp.CreateMessageParams{
			Messages: []*mcp.SamplingMessage{{
				Role:    "user",
				Content: &mcp.TextContent{Text: "Test"},
			}},
			MaxTokens: 1000,
		}
		result, err := session.CreateMessageRaw(t.Context(), params)

		require.NoError(t, err)
		assert.Equal(t, mcp.Role("assistant"), result.Role)
		assert.Equal(t, "Raw response", result.Content.(*mcp.TextContent).Text)
		mockSession.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedErr := assert.AnError
		mockSession.On("CreateMessage", mock.Anything, mock.Anything).Return(nil, expectedErr)

		session := capabilities.NewSession(mockSession.MockServerSession)
		params := &mcp.CreateMessageParams{
			Messages: []*mcp.SamplingMessage{{
				Role:    "user",
				Content: &mcp.TextContent{Text: "Test"},
			}},
			MaxTokens: 1000,
		}
		result, err := session.CreateMessageRaw(t.Context(), params)

		require.Error(t, err)
		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, result)
		mockSession.AssertExpectations(t)
	})
}
