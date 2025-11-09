package capabilities_test

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/robbyt/mcp-io/capabilities/sampling"
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
		result, err := session.CreateMessage(t.Context(), []*sampling.Message{{
			Role:    "user",
			Content: "Analyze this code",
		}}, sampling.WithMaxTokens(2000))

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
		result, err := session.CreateMessage(t.Context(), []*sampling.Message{
			{Role: "user", Content: "First message"},
			{Role: "assistant", Content: "First response"},
			{Role: "user", Content: "Second message"},
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
		mockSession.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedErr := assert.AnError
		mockSession.On("CreateMessage", mock.Anything, mock.Anything).Return(nil, expectedErr)

		session := capabilities.NewSession(mockSession.MockServerSession)
		result, err := session.CreateMessage(t.Context(), []*sampling.Message{{
			Role:    "user",
			Content: "Test",
		}})

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

func TestCreateMessageWithOptions(t *testing.T) {
	t.Parallel()

	//nolint:dupl // Similar structure but testing different options
	t.Run("WithSystemPrompt", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedResult := &mcp.CreateMessageResult{
			Role:    "assistant",
			Content: &mcp.TextContent{Text: "Response with system prompt"},
		}

		// Verify the system prompt is passed
		mockSession.On("CreateMessage", mock.Anything, mock.MatchedBy(func(params *mcp.CreateMessageParams) bool {
			return params.SystemPrompt == "You are a helpful assistant" &&
				params.MaxTokens == 100
		})).Return(expectedResult, nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		messages := []*sampling.Message{
			{Role: "user", Content: "Hello"},
		}

		result, err := session.CreateMessage(t.Context(), messages,
			sampling.WithMaxTokens(100),
			sampling.WithSystemPrompt("You are a helpful assistant"))

		require.NoError(t, err)
		assert.Equal(t, "assistant", result.Role)
		assert.Equal(t, "Response with system prompt", result.Content.Text)
		mockSession.AssertExpectations(t)
	})

	//nolint:dupl // Similar structure but testing different options
	t.Run("WithTemperature", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedResult := &mcp.CreateMessageResult{
			Role:    "assistant",
			Content: &mcp.TextContent{Text: "Response with temperature"},
		}

		// Verify the temperature is passed
		mockSession.On("CreateMessage", mock.Anything, mock.MatchedBy(func(params *mcp.CreateMessageParams) bool {
			return params.Temperature == 0.7 &&
				params.MaxTokens == 100
		})).Return(expectedResult, nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		messages := []*sampling.Message{
			{Role: "user", Content: "Hello"},
		}

		result, err := session.CreateMessage(t.Context(), messages,
			sampling.WithMaxTokens(100),
			sampling.WithTemperature(0.7))

		require.NoError(t, err)
		assert.Equal(t, "assistant", result.Role)
		assert.Equal(t, "Response with temperature", result.Content.Text)
		mockSession.AssertExpectations(t)
	})

	t.Run("WithMultipleOptions", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedResult := &mcp.CreateMessageResult{
			Role:    "assistant",
			Content: &mcp.TextContent{Text: "Response with multiple options"},
		}

		// Verify multiple options are passed correctly
		mockSession.On("CreateMessage", mock.Anything, mock.MatchedBy(func(params *mcp.CreateMessageParams) bool {
			return params.SystemPrompt == "Be helpful" &&
				params.Temperature == 0.8 &&
				len(params.StopSequences) == 2 &&
				params.StopSequences[0] == "END" &&
				params.StopSequences[1] == "STOP" &&
				params.ModelPreferences != nil &&
				params.ModelPreferences.IntelligencePriority == 0.9 &&
				len(params.ModelPreferences.Hints) == 1 &&
				params.ModelPreferences.Hints[0].Name == "claude-3-5-sonnet"
		})).Return(expectedResult, nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		messages := []*sampling.Message{
			{Role: "user", Content: "Hello"},
		}

		result, err := session.CreateMessage(t.Context(), messages,
			sampling.WithMaxTokens(100),
			sampling.WithSystemPrompt("Be helpful"),
			sampling.WithTemperature(0.8),
			sampling.WithStopSequences("END", "STOP"),
			sampling.WithIntelligencePriority(0.9),
			sampling.WithModelHints("claude-3-5-sonnet"))

		require.NoError(t, err)
		assert.Equal(t, "assistant", result.Role)
		assert.Equal(t, "Response with multiple options", result.Content.Text)
		mockSession.AssertExpectations(t)
	})

	t.Run("InvalidTemperature", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		// Should not call CreateMessage due to validation error

		session := capabilities.NewSession(mockSession.MockServerSession)
		messages := []*sampling.Message{
			{Role: "user", Content: "Hello"},
		}

		result, err := session.CreateMessage(t.Context(), messages,
			sampling.WithTemperature(3.0))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "temperature must be between 0.0 and 2.0")
		assert.Nil(t, result)
		mockSession.AssertNotCalled(t, "CreateMessage")
	})

	t.Run("InvalidIncludeContext", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		// Should not call CreateMessage due to validation error

		session := capabilities.NewSession(mockSession.MockServerSession)
		messages := []*sampling.Message{
			{Role: "user", Content: "Hello"},
		}

		result, err := session.CreateMessage(t.Context(), messages,
			sampling.WithIncludeContext("invalidValue"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "includeContext must be")
		assert.Nil(t, result)
		mockSession.AssertNotCalled(t, "CreateMessage")
	})

	t.Run("BackwardCompatibility", func(t *testing.T) {
		// Test that calling without options still works (backward compatibility)
		mockSession := testutil.NewMockSession()
		expectedResult := &mcp.CreateMessageResult{
			Role:    "assistant",
			Content: &mcp.TextContent{Text: "Response without options"},
		}

		// Should create simple params with no extra fields
		mockSession.On("CreateMessage", mock.Anything, mock.MatchedBy(func(params *mcp.CreateMessageParams) bool {
			return params.SystemPrompt == "" &&
				params.Temperature == 0 &&
				params.StopSequences == nil &&
				params.ModelPreferences == nil &&
				params.MaxTokens == 100
		})).Return(expectedResult, nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		messages := []*sampling.Message{
			{Role: "user", Content: "Hello"},
		}

		// Call with explicit maxTokens to verify backward compatibility
		result, err := session.CreateMessage(t.Context(), messages, sampling.WithMaxTokens(100))

		require.NoError(t, err)
		assert.Equal(t, "assistant", result.Role)
		assert.Equal(t, "Response without options", result.Content.Text)
		mockSession.AssertExpectations(t)
	})

	t.Run("DefaultMaxTokens", func(t *testing.T) {
		// Test that default maxTokens of 1000 is used when not specified
		mockSession := testutil.NewMockSession()
		expectedResult := &mcp.CreateMessageResult{
			Role:    "assistant",
			Content: &mcp.TextContent{Text: "Response with default tokens"},
		}

		// Should use default maxTokens of 1000
		mockSession.On("CreateMessage", mock.Anything, mock.MatchedBy(func(params *mcp.CreateMessageParams) bool {
			return params.MaxTokens == 1000
		})).Return(expectedResult, nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		messages := []*sampling.Message{
			{Role: "user", Content: "Hello"},
		}

		result, err := session.CreateMessage(t.Context(), messages)

		require.NoError(t, err)
		assert.Equal(t, "assistant", result.Role)
		assert.Equal(t, "Response with default tokens", result.Content.Text)
		mockSession.AssertExpectations(t)
	})
}
