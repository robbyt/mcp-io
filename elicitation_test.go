package mcpio

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for elicitation
type TestConfig struct {
	Name  string `json:"name"  jsonschema:"Test name"`
	Value int    `json:"value" jsonschema:"Test value"`
}

// TestMockServerSession implements the serverSession interface for testing
type TestMockServerSession struct {
	ElicitFunc func(ctx context.Context, params *mcp.ElicitParams) (*mcp.ElicitResult, error)
}

func (m *TestMockServerSession) InitializeParams() *mcp.InitializeParams {
	return &mcp.InitializeParams{
		Capabilities: &mcp.ClientCapabilities{
			Elicitation: &mcp.ElicitationCapabilities{},
		},
	}
}

func (m *TestMockServerSession) Elicit(ctx context.Context, params *mcp.ElicitParams) (*mcp.ElicitResult, error) {
	if m.ElicitFunc != nil {
		return m.ElicitFunc(ctx, params)
	}
	return &mcp.ElicitResult{Action: "accept", Content: map[string]any{}}, nil
}

func (m *TestMockServerSession) Log(ctx context.Context, params *mcp.LoggingMessageParams) error {
	return nil
}

func (m *TestMockServerSession) NotifyProgress(ctx context.Context, params *mcp.ProgressNotificationParams) error {
	return nil
}

func (m *TestMockServerSession) ListRoots(ctx context.Context, params *mcp.ListRootsParams) (*mcp.ListRootsResult, error) {
	return nil, nil
}

func (m *TestMockServerSession) CreateMessage(ctx context.Context, params *mcp.CreateMessageParams) (*mcp.CreateMessageResult, error) {
	return nil, nil
}

func (m *TestMockServerSession) ID() string {
	return "test-session"
}

func (m *TestMockServerSession) Wait() error {
	return nil
}

func (m *TestMockServerSession) Close() error {
	return nil
}

func TestElicitTyped_Success(t *testing.T) {
	mockServerSession := &TestMockServerSession{
		ElicitFunc: func(ctx context.Context, params *mcp.ElicitParams) (*mcp.ElicitResult, error) {
			return &mcp.ElicitResult{
				Action: "accept",
				Content: map[string]any{
					"name":  "test",
					"value": float64(42), // JSON numbers are float64
				},
			}, nil
		},
	}

	session := capabilities.NewSession(mockServerSession)
	ctx := capabilities.WithSession(t.Context(), session)
	result, err := ElicitTyped[TestConfig](ctx, "Test message")

	require.NoError(t, err)
	assert.True(t, result.IsAccepted())
	assert.Equal(t, "test", result.GetContent()["name"])
	assert.InDelta(t, 42.0, result.GetContent()["value"], 0.0001)
}

func TestElicitTyped_SchemaGeneration(t *testing.T) {
	var capturedSchema any
	mockServerSession := &TestMockServerSession{
		ElicitFunc: func(ctx context.Context, params *mcp.ElicitParams) (*mcp.ElicitResult, error) {
			capturedSchema = params.RequestedSchema
			return &mcp.ElicitResult{Action: "decline"}, nil
		},
	}

	session := capabilities.NewSession(mockServerSession)
	ctx := capabilities.WithSession(t.Context(), session)
	result, err := ElicitTyped[TestConfig](ctx, "Test message")

	require.NoError(t, err)
	assert.True(t, result.IsDeclined())
	assert.NotNil(t, capturedSchema)
}

func TestElicitTyped_NoSession(t *testing.T) {
	ctx := t.Context()
	result, err := ElicitTyped[TestConfig](ctx, "Test message")

	require.Error(t, err)
	require.ErrorIs(t, err, ErrNoSession)
	assert.Nil(t, result)
}

func TestElicitTyped_ElicitationNotSupported(t *testing.T) {
	// Create a mock server session that doesn't support elicitation
	mockServerSession := &TestMockNoElicitation{}
	session := capabilities.NewSession(mockServerSession)

	ctx := capabilities.WithSession(t.Context(), session)
	result, err := ElicitTyped[TestConfig](ctx, "Test message")

	require.Error(t, err)
	require.ErrorIs(t, err, ErrElicitationNotSupported)
	assert.Nil(t, result)
}

// TestMockNoElicitation is a mock server session that doesn't support elicitation
type TestMockNoElicitation struct {
	TestMockServerSession
}

func (m *TestMockNoElicitation) InitializeParams() *mcp.InitializeParams {
	// Return params without Elicitation capability
	return &mcp.InitializeParams{
		Capabilities: &mcp.ClientCapabilities{},
	}
}

func TestElicitationResult_IsAccepted(t *testing.T) {
	result := &ElicitationResult{ElicitResult: &mcp.ElicitResult{Action: "accept"}}
	assert.True(t, result.IsAccepted())

	result = &ElicitationResult{ElicitResult: &mcp.ElicitResult{Action: "decline"}}
	assert.False(t, result.IsAccepted())
}

func TestElicitationResult_IsDeclined(t *testing.T) {
	result := &ElicitationResult{ElicitResult: &mcp.ElicitResult{Action: "decline"}}
	assert.True(t, result.IsDeclined())

	result = &ElicitationResult{ElicitResult: &mcp.ElicitResult{Action: "accept"}}
	assert.False(t, result.IsDeclined())
}

func TestElicitationResult_IsCancelled(t *testing.T) {
	result := &ElicitationResult{ElicitResult: &mcp.ElicitResult{Action: "cancel"}}
	assert.True(t, result.IsCancelled())

	result = &ElicitationResult{ElicitResult: &mcp.ElicitResult{Action: "accept"}}
	assert.False(t, result.IsCancelled())
}

func TestElicitationResult_GetContent(t *testing.T) {
	content := map[string]any{"key": "value"}
	result := &ElicitationResult{ElicitResult: &mcp.ElicitResult{
		Action:  "accept",
		Content: content,
	}}
	assert.Equal(t, content, result.GetContent())

	result = &ElicitationResult{ElicitResult: &mcp.ElicitResult{Action: "decline"}}
	assert.Nil(t, result.GetContent())
}

func TestElicitationResult_DecodeContent(t *testing.T) {
	result := &ElicitationResult{ElicitResult: &mcp.ElicitResult{
		Action: "accept",
		Content: map[string]any{
			"name":  "test",
			"value": float64(42),
		},
	}}

	var config TestConfig
	err := result.DecodeContent(&config)
	require.NoError(t, err)
	assert.Equal(t, "test", config.Name)
	assert.Equal(t, 42, config.Value)
}

func TestElicitationResult_DecodeContent_NoContent(t *testing.T) {
	result := &ElicitationResult{ElicitResult: &mcp.ElicitResult{Action: "decline"}}

	var config TestConfig
	err := result.DecodeContent(&config)
	assert.ErrorIs(t, err, ErrNoContent)
}

func TestElicitSimple_Success(t *testing.T) {
	mockServerSession := &TestMockServerSession{
		ElicitFunc: func(ctx context.Context, params *mcp.ElicitParams) (*mcp.ElicitResult, error) {
			return &mcp.ElicitResult{
				Action:  "accept",
				Content: map[string]any{"username": "testuser"},
			}, nil
		},
	}

	session := capabilities.NewSession(mockServerSession)
	ctx := capabilities.WithSession(t.Context(), session)
	result, err := ElicitSimple(ctx, "Enter username:", "username", "Your username")

	require.NoError(t, err)
	assert.True(t, result.IsAccepted())
	assert.Equal(t, "testuser", result.GetContent()["username"])
}

func TestElicitSimple_Decline(t *testing.T) {
	mockServerSession := &TestMockServerSession{
		ElicitFunc: func(ctx context.Context, params *mcp.ElicitParams) (*mcp.ElicitResult, error) {
			return &mcp.ElicitResult{Action: "decline"}, nil
		},
	}

	session := capabilities.NewSession(mockServerSession)
	ctx := capabilities.WithSession(t.Context(), session)
	result, err := ElicitSimple(ctx, "Enter username:", "username", "Your username")

	require.NoError(t, err)
	assert.True(t, result.IsDeclined())
	assert.Nil(t, result.GetContent())
}

func TestElicitSimple_NoSession(t *testing.T) {
	ctx := t.Context()
	result, err := ElicitSimple(ctx, "Enter username:", "username", "Your username")

	require.Error(t, err)
	require.ErrorIs(t, err, ErrNoSession)
	assert.Nil(t, result)
}

func TestWrapElicitResult(t *testing.T) {
	mcpResult := &mcp.ElicitResult{
		Action:  "accept",
		Content: map[string]any{"test": "data"},
	}

	wrapped := WrapElicitResult(mcpResult)
	assert.NotNil(t, wrapped)
	assert.Equal(t, mcpResult, wrapped.ElicitResult)
	assert.True(t, wrapped.IsAccepted())
}

func TestMultiStepElicitation(t *testing.T) {
	t.Parallel()

	callCount := 0
	mockServerSession := &TestMockServerSession{
		ElicitFunc: func(ctx context.Context, params *mcp.ElicitParams) (*mcp.ElicitResult, error) {
			callCount++
			switch callCount {
			case 1:
				return &mcp.ElicitResult{
					Action: "accept",
					Content: map[string]any{
						"name":  "config1",
						"value": float64(100),
					},
				}, nil
			case 2:
				return &mcp.ElicitResult{
					Action: "accept",
					Content: map[string]any{
						"name":  "config2",
						"value": float64(200),
					},
				}, nil
			case 3:
				return &mcp.ElicitResult{
					Action:  "accept",
					Content: map[string]any{"confirm": "yes"},
				}, nil
			default:
				return &mcp.ElicitResult{Action: "cancel"}, nil
			}
		},
	}

	session := capabilities.NewSession(mockServerSession)
	ctx := capabilities.WithSession(t.Context(), session)

	result1, err := ElicitTyped[TestConfig](ctx, "First config:")
	require.NoError(t, err)
	assert.True(t, result1.IsAccepted())

	var config1 TestConfig
	err = result1.DecodeContent(&config1)
	require.NoError(t, err)
	assert.Equal(t, "config1", config1.Name)
	assert.Equal(t, 100, config1.Value)

	result2, err := ElicitTyped[TestConfig](ctx, "Second config:")
	require.NoError(t, err)
	assert.True(t, result2.IsAccepted())

	var config2 TestConfig
	err = result2.DecodeContent(&config2)
	require.NoError(t, err)
	assert.Equal(t, "config2", config2.Name)
	assert.Equal(t, 200, config2.Value)

	result3, err := ElicitSimple(ctx, "Proceed?", "confirm", "Type yes to proceed")
	require.NoError(t, err)
	assert.True(t, result3.IsAccepted())
	assert.Equal(t, "yes", result3.GetContent()["confirm"])
}
