//go:build integration
// +build integration

package mcpio

import (
	"context"
	"log/slog"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration test types
type UserPreferences struct {
	Theme         string `json:"theme"         jsonschema:"UI theme preference"`
	Language      string `json:"language"      jsonschema:"Preferred language"`
	Notifications bool   `json:"notifications" jsonschema:"Enable notifications"`
}

type APIConfiguration struct {
	Endpoint string `json:"endpoint" jsonschema:"API endpoint URL"`
	Timeout  int    `json:"timeout"  jsonschema:"Request timeout in seconds"`
	Retries  int    `json:"retries"  jsonschema:"Number of retry attempts"`
}

// MockSessionCapability for testing context-based elicitation
type MockSessionCapability struct {
	ElicitResponses []*mcp.ElicitResult
	CallIndex       int
}

func (m *MockSessionCapability) Elicit(ctx context.Context, message string, requestedSchema any) (*mcp.ElicitResult, error) {
	if m.CallIndex >= len(m.ElicitResponses) {
		return &mcp.ElicitResult{Action: "cancel"}, nil
	}
	result := m.ElicitResponses[m.CallIndex]
	m.CallIndex++
	return result, nil
}

func (m *MockSessionCapability) CreateMessage(ctx context.Context, messages []*capabilities.Message, maxTokens int) (*capabilities.MessageResult, error) {
	return nil, nil
}

func (m *MockSessionCapability) CreateMessageRaw(ctx context.Context, params *mcp.CreateMessageParams) (*mcp.CreateMessageResult, error) {
	return nil, nil
}

func (m *MockSessionCapability) ListRoots(ctx context.Context) ([]*capabilities.Root, error) {
	return nil, nil
}

func (m *MockSessionCapability) Log(ctx context.Context, level capabilities.LogLevel, message string, data map[string]any) error {
	return nil
}

func (m *MockSessionCapability) Logger() *slog.Logger {
	return nil
}

func (m *MockSessionCapability) NotifyProgress(ctx context.Context, progress, total float64) error {
	return nil
}

func (m *MockSessionCapability) SessionID() string {
	return "test-session"
}

func (m *MockSessionCapability) ClientCapabilities() *capabilities.ClientCapabilities {
	return &capabilities.ClientCapabilities{
		Elicitation: &capabilities.ElicitationCapabilities{},
	}
}

func (m *MockSessionCapability) SupportsElicitation() bool {
	return true
}

func (m *MockSessionCapability) SupportsSampling() bool {
	return false
}

func (m *MockSessionCapability) Wait() error {
	return nil
}

func (m *MockSessionCapability) Close() error {
	return nil
}

func TestToolWithContextElicitation(t *testing.T) {
	// Create a mock session with predefined responses
	mockSession := &MockSessionCapability{
		ElicitResponses: []*mcp.ElicitResult{
			{
				Action: "accept",
				Content: map[string]any{
					"theme":         "dark",
					"language":      "en",
					"notifications": true,
				},
			},
		},
	}

	// Define a tool that uses context-based elicitation
	setupUserPreferences := func(ctx context.Context, _ struct{}) (map[string]any, error) {
		result, err := ElicitTyped[UserPreferences](ctx, "Please configure your preferences:")
		if err != nil {
			return nil, err
		}

		if !result.IsAccepted() {
			return map[string]any{"status": "cancelled"}, nil
		}

		// Parse the preferences
		var prefs UserPreferences
		if err := result.DecodeContent(&prefs); err != nil {
			return nil, err
		}

		return map[string]any{
			"status": "configured",
			"preferences": map[string]any{
				"theme":         prefs.Theme,
				"language":      prefs.Language,
				"notifications": prefs.Notifications,
			},
		}, nil
	}

	// Create handler with regular tool (session is injected automatically)
	handler, err := NewHandler(
		WithName("test-elicitation"),
		WithVersion("1.0.0"),
		WithTool("setup_preferences", "Setup user preferences", setupUserPreferences),
	)
	require.NoError(t, err)
	assert.NotNil(t, handler)

	// Test direct elicitation call with mock session injected in context
	ctx := injectSession(context.Background(), mockSession)
	result, err := ElicitTyped[UserPreferences](ctx, "Test message")
	require.NoError(t, err)
	assert.True(t, result.IsAccepted())
	assert.Equal(t, "dark", result.GetContent()["theme"])
	assert.Equal(t, "en", result.GetContent()["language"])
	assert.Equal(t, true, result.GetContent()["notifications"])
}

func TestMultiStepElicitation(t *testing.T) {
	// Create a mock session with multiple responses
	mockSession := &MockSessionCapability{
		ElicitResponses: []*mcp.ElicitResult{
			{
				Action: "accept",
				Content: map[string]any{
					"endpoint": "https://api.example.com",
					"timeout":  float64(30),
					"retries":  float64(3),
				},
			},
			{
				Action: "accept",
				Content: map[string]any{
					"theme":         "light",
					"language":      "es",
					"notifications": false,
				},
			},
			{
				Action: "accept",
				Content: map[string]any{
					"confirm": "yes",
				},
			},
		},
	}

	ctx := injectSession(context.Background(), mockSession)

	// First elicitation - API config
	apiResult, err := ElicitTyped[APIConfiguration](ctx, "Configure API settings:")
	require.NoError(t, err)
	assert.True(t, apiResult.IsAccepted())

	var apiConfig APIConfiguration
	err = apiResult.DecodeContent(&apiConfig)
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com", apiConfig.Endpoint)
	assert.Equal(t, 30, apiConfig.Timeout)
	assert.Equal(t, 3, apiConfig.Retries)

	// Second elicitation - User preferences
	prefResult, err := ElicitTyped[UserPreferences](ctx, "Configure preferences:")
	require.NoError(t, err)
	assert.True(t, prefResult.IsAccepted())

	var prefs UserPreferences
	err = prefResult.DecodeContent(&prefs)
	require.NoError(t, err)
	assert.Equal(t, "light", prefs.Theme)
	assert.Equal(t, "es", prefs.Language)
	assert.False(t, prefs.Notifications)

	// Third elicitation - Simple confirmation
	confirmResult, err := ElicitSimple(ctx, "Proceed?", "confirm", "Type yes to proceed")
	require.NoError(t, err)
	assert.True(t, confirmResult.IsAccepted())
	assert.Equal(t, "yes", confirmResult.GetContent()["confirm"])
}

func TestElicitationDecline(t *testing.T) {
	mockSession := &MockSessionCapability{
		ElicitResponses: []*mcp.ElicitResult{
			{Action: "decline"},
		},
	}

	ctx := injectSession(context.Background(), mockSession)
	result, err := ElicitTyped[UserPreferences](ctx, "Configure preferences:")
	require.NoError(t, err)
	assert.True(t, result.IsDeclined())
	assert.Nil(t, result.GetContent())
}

func TestElicitationCancel(t *testing.T) {
	mockSession := &MockSessionCapability{
		ElicitResponses: []*mcp.ElicitResult{
			{Action: "cancel"},
		},
	}

	ctx := injectSession(context.Background(), mockSession)
	result, err := ElicitSimple(ctx, "Enter name:", "name", "Your name")
	require.NoError(t, err)
	assert.True(t, result.IsCancelled())
	assert.Nil(t, result.GetContent())
}

func TestElicitationWithoutSession(t *testing.T) {
	// Test that elicitation fails gracefully when no session is in context
	ctx := context.Background()
	result, err := ElicitTyped[UserPreferences](ctx, "Test message")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNoSession)
	assert.Nil(t, result)
}
