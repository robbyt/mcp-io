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

// MockSamplingSession is a mock session that supports sampling
type MockSamplingSession struct {
	SamplingResponses []*capabilities.MessageResult
	CallIndex         int
	SamplingSupport   bool
}

func (m *MockSamplingSession) Elicit(ctx context.Context, message string, requestedSchema any) (*mcp.ElicitResult, error) {
	return nil, ErrElicitationNotSupported
}

func (m *MockSamplingSession) CreateMessage(ctx context.Context, messages []*capabilities.Message, maxTokens int) (*capabilities.MessageResult, error) {
	if m.CallIndex >= len(m.SamplingResponses) {
		return &capabilities.MessageResult{
			Role:    "assistant",
			Content: capabilities.TextContent{Text: "Default response"},
		}, nil
	}
	result := m.SamplingResponses[m.CallIndex]
	m.CallIndex++
	return result, nil
}

func (m *MockSamplingSession) CreateMessageRaw(ctx context.Context, params *mcp.CreateMessageParams) (*mcp.CreateMessageResult, error) {
	return nil, nil
}

func (m *MockSamplingSession) ListRoots(ctx context.Context) ([]*capabilities.Root, error) {
	return nil, nil
}

func (m *MockSamplingSession) Log(ctx context.Context, level capabilities.LogLevel, message string, data map[string]any) error {
	return nil
}

func (m *MockSamplingSession) Logger() *slog.Logger {
	return slog.Default()
}

func (m *MockSamplingSession) NotifyProgress(ctx context.Context, progress, total float64) error {
	return nil
}

func (m *MockSamplingSession) SessionID() string {
	return "test-session-sampling"
}

func (m *MockSamplingSession) ClientCapabilities() *capabilities.ClientCapabilities {
	caps := &capabilities.ClientCapabilities{}
	if m.SamplingSupport {
		caps.Sampling = &capabilities.SamplingCapabilities{}
	}
	return caps
}

func (m *MockSamplingSession) SupportsElicitation() bool {
	return false
}

func (m *MockSamplingSession) SupportsSampling() bool {
	return m.SamplingSupport
}

func (m *MockSamplingSession) Wait() error {
	return nil
}

func (m *MockSamplingSession) Close() error {
	return nil
}

// Test CreateMessage with sampling support
func TestCreateMessageWithSampling(t *testing.T) {
	t.Parallel()

	mockSession := &MockSamplingSession{
		SamplingSupport: true,
		SamplingResponses: []*capabilities.MessageResult{
			{
				Role: "assistant",
				Content: capabilities.TextContent{
					Text: "This is an AI-generated analysis of the code provided.",
				},
			},
		},
	}

	ctx := InjectSessionForTesting(context.Background(), mockSession)

	result, err := CreateMessage(ctx, []*capabilities.Message{{
		Role:    "user",
		Content: "Analyze this code: func main() {}",
	}}, 1000)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "assistant", result.Role)
	assert.Contains(t, result.Content.Text, "AI-generated")
}

// Test CreateMessage without sampling support
func TestCreateMessageWithoutSampling(t *testing.T) {
	t.Parallel()

	mockSession := &MockSamplingSession{
		SamplingSupport: false,
	}

	ctx := InjectSessionForTesting(context.Background(), mockSession)

	result, err := CreateMessage(ctx, []*capabilities.Message{{
		Role:    "user",
		Content: "Test message",
	}}, 1000)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrSamplingNotSupported)
	assert.Nil(t, result)
}

// Test CreateMessage without session
func TestCreateMessageWithoutSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	result, err := CreateMessage(ctx, []*capabilities.Message{{
		Role:    "user",
		Content: "Test message",
	}}, 1000)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrNoSession)
	assert.Nil(t, result)
}

// Test multi-turn conversation with sampling
func TestMultiTurnSampling(t *testing.T) {
	t.Parallel()

	mockSession := &MockSamplingSession{
		SamplingSupport: true,
		SamplingResponses: []*capabilities.MessageResult{
			{
				Role:    "assistant",
				Content: capabilities.TextContent{Text: "First response"},
			},
			{
				Role:    "assistant",
				Content: capabilities.TextContent{Text: "Second response"},
			},
			{
				Role:    "assistant",
				Content: capabilities.TextContent{Text: "Third response"},
			},
		},
	}

	ctx := InjectSessionForTesting(context.Background(), mockSession)

	// First message
	result1, err := CreateMessage(ctx, []*capabilities.Message{{
		Role:    "user",
		Content: "First question",
	}}, 1000)
	require.NoError(t, err)
	assert.Equal(t, "First response", result1.Content.Text)

	// Second message
	result2, err := CreateMessage(ctx, []*capabilities.Message{{
		Role:    "user",
		Content: "Second question",
	}}, 1000)
	require.NoError(t, err)
	assert.Equal(t, "Second response", result2.Content.Text)

	// Third message
	result3, err := CreateMessage(ctx, []*capabilities.Message{{
		Role:    "user",
		Content: "Third question",
	}}, 1000)
	require.NoError(t, err)
	assert.Equal(t, "Third response", result3.Content.Text)
}

// Test SupportsSampling capability check
func TestSupportsSamplingCheck(t *testing.T) {
	t.Parallel()

	t.Run("SamplingSupported", func(t *testing.T) {
		mockSession := &MockSamplingSession{
			SamplingSupport: true,
		}

		ctx := InjectSessionForTesting(context.Background(), mockSession)
		session := GetSession(ctx)

		require.NotNil(t, session)
		assert.True(t, session.SupportsSampling())
	})

	t.Run("SamplingNotSupported", func(t *testing.T) {
		mockSession := &MockSamplingSession{
			SamplingSupport: false,
		}

		ctx := InjectSessionForTesting(context.Background(), mockSession)
		session := GetSession(ctx)

		require.NotNil(t, session)
		assert.False(t, session.SupportsSampling())
	})
}

// Test sampling in a tool function
func TestSamplingInTool(t *testing.T) {
	t.Parallel()

	type AnalyzeInput struct {
		Code string `json:"code"`
	}

	type AnalyzeOutput struct {
		Analysis     string `json:"analysis"`
		SamplingUsed bool   `json:"samplingUsed"`
	}

	analyzeTool := func(ctx context.Context, input AnalyzeInput) (AnalyzeOutput, error) {
		session := GetSession(ctx)
		if session == nil || !session.SupportsSampling() {
			return AnalyzeOutput{
				Analysis:     "Sampling not available",
				SamplingUsed: false,
			}, nil
		}

		result, err := CreateMessage(ctx, []*capabilities.Message{{
			Role:    "user",
			Content: "Analyze: " + input.Code,
		}}, 2000)
		if err != nil {
			return AnalyzeOutput{}, err
		}

		return AnalyzeOutput{
			Analysis:     result.Content.Text,
			SamplingUsed: true,
		}, nil
	}

	mockSession := &MockSamplingSession{
		SamplingSupport: true,
		SamplingResponses: []*capabilities.MessageResult{
			{
				Role:    "assistant",
				Content: capabilities.TextContent{Text: "Code analysis: Looks good!"},
			},
		},
	}

	ctx := InjectSessionForTesting(context.Background(), mockSession)

	output, err := analyzeTool(ctx, AnalyzeInput{Code: "func test() {}"})
	require.NoError(t, err)
	assert.True(t, output.SamplingUsed)
	assert.Contains(t, output.Analysis, "Code analysis")
}

// Test that session is actually injected for typed tools (PR #1 bug fix)
func TestTypedToolSessionInjection(t *testing.T) {
	t.Parallel()

	type Input struct {
		Text string `json:"text"`
	}
	type Output struct {
		SessionID string `json:"sessionId"`
	}

	handler, err := NewHandler(
		WithName("session-test"),
		WithTool("typed_tool", "Test session injection", func(ctx context.Context, input Input) (Output, error) {
			session := GetSession(ctx)
			require.NotNil(t, session, "Session should be injected for typed tools")
			return Output{SessionID: session.SessionID()}, nil
		}),
	)
	require.NoError(t, err)

	// Create in-memory transport
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()

	go func() {
		_ = handler.GetServer().Run(ctx, serverTransport) //nolint:errcheck
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = session.Close() }() //nolint:errcheck

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "typed_tool",
		Arguments: map[string]any{"text": "test"},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
}

// Test that session is injected for raw tools (PR #1 bug fix)
func TestRawToolSessionInjection(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(
		WithName("session-test"),
		WithRawTool("raw_tool", "Test session injection",
			`{"type":"object"}`,
			func(ctx context.Context, input []byte) ([]byte, error) {
				session := GetSession(ctx)
				require.NotNil(t, session, "Session should be injected for raw tools")
				return []byte(`{"session_id":"` + session.SessionID() + `"}`), nil
			},
		),
	)
	require.NoError(t, err)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()

	go func() {
		_ = handler.GetServer().Run(ctx, serverTransport) //nolint:errcheck
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = session.Close() }() //nolint:errcheck

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "raw_tool",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
}

// Test that session is injected for prompts (PR #1 bug fix)
func TestPromptSessionInjection(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(
		WithName("session-test"),
		WithPrompt("test_prompt", "Test session injection",
			func(ctx context.Context, args map[string]any) (*PromptResult, error) {
				session := GetSession(ctx)
				require.NotNil(t, session, "Session should be injected for prompts")
				return &PromptResult{
					Messages: []PromptMessage{{Role: "user", Content: "Session: " + session.SessionID()}},
				}, nil
			},
		),
	)
	require.NoError(t, err)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()

	go func() {
		_ = handler.GetServer().Run(ctx, serverTransport) //nolint:errcheck
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = session.Close() }() //nolint:errcheck

	result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{Name: "test_prompt"})
	require.NoError(t, err)
	require.Len(t, result.Messages, 1)
}

// Test that session is injected for typed prompts (PR #1 bug fix)
func TestTypedPromptSessionInjection(t *testing.T) {
	t.Parallel()

	type PromptArgs struct {
		Topic string `json:"topic"`
	}

	handler, err := NewHandler(
		WithName("session-test"),
		WithTypedPrompt("typed_prompt", "Test session injection",
			func(ctx context.Context, args PromptArgs) (*PromptResult, error) {
				session := GetSession(ctx)
				require.NotNil(t, session, "Session should be injected for typed prompts")
				return &PromptResult{
					Messages: []PromptMessage{{Role: "user", Content: "Session: " + session.SessionID()}},
				}, nil
			},
		),
	)
	require.NoError(t, err)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()

	go func() {
		_ = handler.GetServer().Run(ctx, serverTransport) //nolint:errcheck
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = session.Close() }() //nolint:errcheck

	result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name:      "typed_prompt",
		Arguments: map[string]string{"topic": "test"},
	})
	require.NoError(t, err)
	require.Len(t, result.Messages, 1)
}

// Test that session is injected for resources (PR #1 bug fix)
func TestResourceSessionInjection(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(
		WithName("session-test"),
		WithResource("test://resource", "Test session injection",
			func(ctx context.Context, uri string) (*ResourceContent, error) {
				session := GetSession(ctx)
				require.NotNil(t, session, "Session should be injected for resources")
				return &ResourceContent{
					Content:  []byte("Session: " + session.SessionID()),
					MIMEType: "text/plain",
				}, nil
			},
		),
	)
	require.NoError(t, err)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()

	go func() {
		_ = handler.GetServer().Run(ctx, serverTransport) //nolint:errcheck
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = session.Close() }() //nolint:errcheck

	result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "test://resource"})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
}
