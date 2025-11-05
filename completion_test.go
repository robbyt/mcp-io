package mcpio_test

import (
	"context"
	"testing"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionResult_Validate(t *testing.T) {
	tests := []struct {
		name    string
		result  *mcpio.CompletionResult
		wantErr error
	}{
		{
			name: "valid with values only",
			result: &mcpio.CompletionResult{
				Values: []string{"a", "b", "c"},
			},
			wantErr: nil,
		},
		{
			name: "valid with total matching length",
			result: &mcpio.CompletionResult{
				Values: []string{"a", "b"},
				Total:  2,
			},
			wantErr: nil,
		},
		{
			name: "valid with hasmore and higher total",
			result: &mcpio.CompletionResult{
				Values:  []string{"a", "b"},
				Total:   10,
				HasMore: true,
			},
			wantErr: nil,
		},
		{
			name: "invalid - no values",
			result: &mcpio.CompletionResult{
				Values: []string{},
			},
			wantErr: mcpio.ErrNoCompletions,
		},
		{
			name: "invalid - total less than values",
			result: &mcpio.CompletionResult{
				Values: []string{"a", "b", "c"},
				Total:  2,
			},
			wantErr: mcpio.ErrInvalidTotal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWithCompletion(t *testing.T) {
	t.Run("valid completion handler", func(t *testing.T) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("test"),
			mcpio.WithCompletion(func(ctx context.Context, reqCtx mcpio.RequestContext, ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
				return &mcpio.CompletionResult{Values: []string{"test"}}, nil
			}),
		)
		require.NoError(t, err)
		require.NotNil(t, handler)
	})

	t.Run("nil completion handler", func(t *testing.T) {
		_, err := mcpio.NewHandler(
			mcpio.WithName("test"),
			mcpio.WithCompletion(nil),
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, mcpio.ErrNilValue)
	})
}

func TestCompletionError(t *testing.T) {
	err := mcpio.NewCompletionError("test error message")
	require.Error(t, err)
	assert.Equal(t, "test error message", err.Error())
}
