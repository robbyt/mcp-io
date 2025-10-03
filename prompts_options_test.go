package mcpio

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithPrompt(t *testing.T) {
	t.Parallel()

	testPromptFunc := func(ctx context.Context, args map[string]any) (*PromptResult, error) {
		return &PromptResult{
			Messages: []PromptMessage{{Role: "user", Content: "test"}},
		}, nil
	}

	tests := []struct {
		name          string
		promptName    string
		description   string
		promptFunc    PromptFunc
		wantErr       error
		expectPrompts int
	}{
		{
			name:          "valid prompt",
			promptName:    "test-prompt",
			description:   "A test prompt",
			promptFunc:    testPromptFunc,
			wantErr:       nil,
			expectPrompts: 1,
		},
		{
			name:          "empty prompt name should return error",
			promptName:    "",
			description:   "A test prompt",
			promptFunc:    testPromptFunc,
			wantErr:       ErrEmptyValue,
			expectPrompts: 0,
		},
		{
			name:          "empty description should be valid",
			promptName:    "test-prompt",
			description:   "",
			promptFunc:    testPromptFunc,
			wantErr:       nil,
			expectPrompts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &handlerConfig{
				prompts: make([]promptRegisterFunc, 0),
			}
			option := WithPrompt(tt.promptName, tt.description, tt.promptFunc)
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

func TestWithPromptWithArgs(t *testing.T) {
	t.Parallel()

	testPromptFunc := func(ctx context.Context, args map[string]any) (*PromptResult, error) {
		return &PromptResult{
			Messages: []PromptMessage{{Role: "user", Content: "test"}},
		}, nil
	}

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
		promptFunc    PromptFunc
		wantErr       error
		expectPrompts int
	}{
		{
			name:          "valid prompt with args",
			promptName:    "test-prompt",
			description:   "A test prompt",
			args:          promptArgs,
			promptFunc:    testPromptFunc,
			wantErr:       nil,
			expectPrompts: 1,
		},
		{
			name:          "empty prompt name should return error",
			promptName:    "",
			description:   "A test prompt",
			args:          promptArgs,
			promptFunc:    testPromptFunc,
			wantErr:       ErrEmptyValue,
			expectPrompts: 0,
		},
		{
			name:          "nil args should be valid",
			promptName:    "test-prompt",
			description:   "A test prompt",
			args:          nil,
			promptFunc:    testPromptFunc,
			wantErr:       nil,
			expectPrompts: 1,
		},
		{
			name:          "empty args slice should be valid",
			promptName:    "test-prompt",
			description:   "A test prompt",
			args:          []*mcp.PromptArgument{},
			promptFunc:    testPromptFunc,
			wantErr:       nil,
			expectPrompts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &handlerConfig{
				prompts: make([]promptRegisterFunc, 0),
			}
			option := WithPromptWithArgs(tt.promptName, tt.description, tt.args, tt.promptFunc)
			err := option(cfg)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Len(t, cfg.prompts, tt.expectPrompts)
		})
	}

	// Test that the registration closure works correctly
	t.Run("registration function executes", func(t *testing.T) {
		handler, err := NewHandler(
			WithName("test-server"),
			WithPromptWithArgs("greeting", "Generate greeting with args", promptArgs, testPromptFunc),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)
		assert.NotNil(t, handler.GetServer())
	})
}
