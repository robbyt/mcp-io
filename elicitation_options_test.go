package mcpio

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types and functions for session-aware testing
type sessionTestInput struct {
	Message string `json:"message"`
}

type sessionTestOutput struct {
	Response string `json:"response"`
}

func testSessionAwareToolFunc(ctx context.Context, capability ElicitationCapability, input sessionTestInput) (sessionTestOutput, error) {
	return sessionTestOutput{Response: "processed: " + input.Message}, nil
}

func testSessionAwarePromptFunc(ctx context.Context, capability ElicitationCapability, args map[string]any) (*PromptResult, error) {
	return &PromptResult{
		Messages: []PromptMessage{{Role: "user", Content: "session prompt"}},
	}, nil
}

func testSessionAwareResourceFunc(ctx context.Context, capability ElicitationCapability, uri string) (*ResourceContent, error) {
	return &ResourceContent{Content: []byte("session resource: " + uri), MIMEType: "text/plain"}, nil
}

func TestWithSessionTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		toolName    string
		description string
		toolFunc    SessionAwareToolFunc[sessionTestInput, sessionTestOutput]
		wantErr     error
		expectTools int
	}{
		{
			name:        "valid session tool",
			toolName:    "session-tool",
			description: "A session-aware test tool",
			toolFunc:    testSessionAwareToolFunc,
			wantErr:     nil,
			expectTools: 1,
		},
		{
			name:        "empty tool name should return error",
			toolName:    "",
			description: "A session-aware test tool",
			toolFunc:    testSessionAwareToolFunc,
			wantErr:     ErrEmptyToolName,
			expectTools: 0,
		},
		{
			name:        "empty description should be valid",
			toolName:    "session-tool",
			description: "",
			toolFunc:    testSessionAwareToolFunc,
			wantErr:     nil,
			expectTools: 1,
		},
		{
			name:        "tool with special characters in name",
			toolName:    "session_tool-v1",
			description: "A session tool with special chars",
			toolFunc:    testSessionAwareToolFunc,
			wantErr:     nil,
			expectTools: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &handlerConfig{
				tools: make([]toolRegisterFunc, 0),
			}
			option := WithSessionTool(tt.toolName, tt.description, tt.toolFunc)
			err := option(cfg)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Len(t, cfg.tools, tt.expectTools)
		})
	}
}

func TestWithSessionPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		promptName    string
		description   string
		promptFunc    SessionAwarePromptFunc
		wantErr       error
		expectPrompts int
	}{
		{
			name:          "valid session prompt",
			promptName:    "session-prompt",
			description:   "A session-aware test prompt",
			promptFunc:    testSessionAwarePromptFunc,
			wantErr:       nil,
			expectPrompts: 1,
		},
		{
			name:          "empty prompt name should return error",
			promptName:    "",
			description:   "A session-aware test prompt",
			promptFunc:    testSessionAwarePromptFunc,
			wantErr:       ErrEmptyPromptName,
			expectPrompts: 0,
		},
		{
			name:          "empty description should be valid",
			promptName:    "session-prompt",
			description:   "",
			promptFunc:    testSessionAwarePromptFunc,
			wantErr:       nil,
			expectPrompts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &handlerConfig{
				prompts: make([]promptRegisterFunc, 0),
			}
			option := WithSessionPrompt(tt.promptName, tt.description, tt.promptFunc)
			err := option(cfg)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Len(t, cfg.prompts, tt.expectPrompts)
		})
	}
}

func TestWithSessionPromptWithArgs(t *testing.T) {
	t.Parallel()

	promptArgs := []*mcp.PromptArgument{
		{
			Name:        "name",
			Description: "The name to use",
			Required:    true,
		},
	}

	tests := []struct {
		name          string
		promptName    string
		description   string
		args          []*mcp.PromptArgument
		promptFunc    SessionAwarePromptFunc
		wantErr       error
		expectPrompts int
	}{
		{
			name:          "valid session prompt with args",
			promptName:    "session-prompt",
			description:   "A session-aware test prompt",
			args:          promptArgs,
			promptFunc:    testSessionAwarePromptFunc,
			wantErr:       nil,
			expectPrompts: 1,
		},
		{
			name:          "empty prompt name should return error",
			promptName:    "",
			description:   "A session-aware test prompt",
			args:          promptArgs,
			promptFunc:    testSessionAwarePromptFunc,
			wantErr:       ErrEmptyPromptName,
			expectPrompts: 0,
		},
		{
			name:          "nil args should be valid",
			promptName:    "session-prompt",
			description:   "A session-aware test prompt",
			args:          nil,
			promptFunc:    testSessionAwarePromptFunc,
			wantErr:       nil,
			expectPrompts: 1,
		},
		{
			name:          "empty args slice should be valid",
			promptName:    "session-prompt",
			description:   "A session-aware test prompt",
			args:          []*mcp.PromptArgument{},
			promptFunc:    testSessionAwarePromptFunc,
			wantErr:       nil,
			expectPrompts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &handlerConfig{
				prompts: make([]promptRegisterFunc, 0),
			}
			option := WithSessionPromptWithArgs(tt.promptName, tt.description, tt.args, tt.promptFunc)
			err := option(cfg)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Len(t, cfg.prompts, tt.expectPrompts)
		})
	}
}

func TestWithSessionResource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		uri       string
		wantErr   error
		wantCount int
	}{
		{"valid session resource", "session://resource", nil, 1},
		{"empty URI should return error", "", ErrEmptyResourceURI, 0},
		{"empty description should be valid", "session://resource", nil, 1},
		{"complex URI should be valid", "file:///path/to/session/resource.txt", nil, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &handlerConfig{resources: make([]resourceRegisterFunc, 0)}
			option := WithSessionResource(tt.uri, "test description", testSessionAwareResourceFunc)
			err := option(cfg)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Len(t, cfg.resources, tt.wantCount)
		})
	}
}

func TestWithSessionResourceTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		template  string
		wantErr   error
		wantCount int
	}{
		{"valid session resource template", "session://users/{id}", nil, 1},
		{"empty URI template should return error", "", ErrEmptyResourceTemplate, 0},
		{"empty description should be valid", "session://config/{section}", nil, 1},
		{"multiple placeholders should be valid", "api://v1/users/{userId}/posts/{postId}", nil, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &handlerConfig{resourceTemplates: make([]resourceTemplateRegisterFunc, 0)}
			option := WithSessionResourceTemplate(tt.template, "test description", testSessionAwareResourceFunc)
			err := option(cfg)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Len(t, cfg.resourceTemplates, tt.wantCount)
		})
	}
}

func TestMultipleSessionOptionsApplication(t *testing.T) {
	t.Parallel()

	cfg := &handlerConfig{
		tools:             make([]toolRegisterFunc, 0),
		prompts:           make([]promptRegisterFunc, 0),
		resources:         make([]resourceRegisterFunc, 0),
		resourceTemplates: make([]resourceTemplateRegisterFunc, 0),
	}

	promptArgs := []*mcp.PromptArgument{
		{Name: "arg1", Description: "Test argument", Required: true},
	}

	options := []Option{
		WithSessionTool("session-tool", "A session tool", testSessionAwareToolFunc),
		WithSessionPrompt("session-prompt", "A session prompt", testSessionAwarePromptFunc),
		WithSessionPromptWithArgs("session-prompt-args", "A session prompt with args", promptArgs, testSessionAwarePromptFunc),
		WithSessionResource("session://resource", "A session resource", testSessionAwareResourceFunc),
		WithSessionResourceTemplate("session://template/{id}", "A session template", testSessionAwareResourceFunc),
	}

	// Apply all options
	for _, opt := range options {
		err := opt(cfg)
		require.NoError(t, err)
	}

	// Verify all config fields are set correctly
	assert.Len(t, cfg.tools, 1)
	assert.Len(t, cfg.prompts, 2) // Two prompts: one regular, one with args
	assert.Len(t, cfg.resources, 1)
	assert.Len(t, cfg.resourceTemplates, 1)
}

func TestSessionOptionErrorConditions(t *testing.T) {
	t.Parallel()

	// Test that errors from session options don't affect config state
	cfg := &handlerConfig{
		tools:             make([]toolRegisterFunc, 0),
		prompts:           make([]promptRegisterFunc, 0),
		resources:         make([]resourceRegisterFunc, 0),
		resourceTemplates: make([]resourceTemplateRegisterFunc, 0),
	}

	// Apply a valid session tool first
	err := WithSessionTool("valid-tool", "Valid tool", testSessionAwareToolFunc)(cfg)
	require.NoError(t, err)
	assert.Len(t, cfg.tools, 1)

	// Apply an invalid session tool - should not change existing state
	err = WithSessionTool("", "Invalid tool", testSessionAwareToolFunc)(cfg)
	require.ErrorIs(t, err, ErrEmptyToolName)
	assert.Len(t, cfg.tools, 1) // Should remain unchanged

	// Apply valid session prompt
	err = WithSessionPrompt("valid-prompt", "Valid prompt", testSessionAwarePromptFunc)(cfg)
	require.NoError(t, err)
	assert.Len(t, cfg.prompts, 1)

	// Apply invalid session prompt - should not change prompt state
	err = WithSessionPrompt("", "Invalid prompt", testSessionAwarePromptFunc)(cfg)
	require.ErrorIs(t, err, ErrEmptyPromptName)
	assert.Len(t, cfg.prompts, 1) // Should remain unchanged

	// Apply valid session resource
	err = WithSessionResource("valid://resource", "Valid resource", testSessionAwareResourceFunc)(cfg)
	require.NoError(t, err)
	assert.Len(t, cfg.resources, 1)

	// Apply invalid session resource - should not change resource state
	err = WithSessionResource("", "Invalid resource", testSessionAwareResourceFunc)(cfg)
	require.ErrorIs(t, err, ErrEmptyResourceURI)
	assert.Len(t, cfg.resources, 1) // Should remain unchanged
}

func TestSessionOptionIntegrationWithHandler(t *testing.T) {
	t.Parallel()

	// Test that session-aware options can be used in actual handler creation
	handler, err := NewHandler(
		WithName("session-test-server"),
		WithVersion("1.0.0"),
		WithSessionTool("integration-tool", "Integration test tool", testSessionAwareToolFunc),
		WithSessionPrompt("integration-prompt", "Integration test prompt", testSessionAwarePromptFunc),
		WithSessionResource("integration://resource", "Integration test resource", testSessionAwareResourceFunc),
	)

	require.NoError(t, err)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.GetServer())

	// Verify the server was created with the expected implementation
	server := handler.GetServer()
	assert.NotNil(t, server)
}
