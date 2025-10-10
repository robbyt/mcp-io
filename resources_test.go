package mcpio

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResourceHandler_Validation(t *testing.T) {
	t.Parallel()

	// Test that NewResourceHandler rejects non-resource types
	toolFunc := func(ctx context.Context, _ struct{}) (struct{}, error) {
		return struct{}{}, nil
	}

	_, err := NewResourceHandler(
		WithName("invalid"),
		WithTool("test", "test tool", toolFunc),
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrIncompatibleHandler)
}
