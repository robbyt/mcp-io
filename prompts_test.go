package mcpio

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPromptHandler_Validation(t *testing.T) {
	t.Parallel()

	// Test that NewPromptHandler rejects non-prompt resources
	toolFunc := func(ctx context.Context, input struct{}) (struct{}, error) {
		return struct{}{}, nil
	}

	_, err := NewPromptHandler(
		WithName("invalid"),
		WithTool("test", "test tool", toolFunc),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "only supports prompts")
}
