package mcpio

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionResult_Validate(t *testing.T) {
	tests := []struct {
		name    string
		result  *CompletionResult
		wantErr error
	}{
		{
			name: "valid with values only",
			result: &CompletionResult{
				Values: []string{"a", "b", "c"},
			},
			wantErr: nil,
		},
		{
			name: "valid with total matching length",
			result: &CompletionResult{
				Values: []string{"a", "b"},
				Total:  2,
			},
			wantErr: nil,
		},
		{
			name: "valid with hasmore and higher total",
			result: &CompletionResult{
				Values:  []string{"a", "b"},
				Total:   10,
				HasMore: true,
			},
			wantErr: nil,
		},
		{
			name: "valid with metadata",
			result: &CompletionResult{
				Values: []string{"a", "b"},
				Meta: map[string]any{
					"source": "cache",
					"ttl":    3600,
				},
			},
			wantErr: nil,
		},
		{
			name: "invalid - no values",
			result: &CompletionResult{
				Values: []string{},
			},
			wantErr: ErrNoCompletions,
		},
		{
			name: "invalid - total less than values",
			result: &CompletionResult{
				Values: []string{"a", "b", "c"},
				Total:  2,
			},
			wantErr: ErrInvalidTotal,
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
		handler, err := NewHandler(
			WithName("test"),
			WithCompletion(func(ctx context.Context, reqCtx RequestContext, ref CompletionRef) (*CompletionResult, error) {
				return &CompletionResult{Values: []string{"test"}}, nil
			}),
		)
		require.NoError(t, err)
		require.NotNil(t, handler)
	})

	t.Run("nil completion handler", func(t *testing.T) {
		_, err := NewHandler(
			WithName("test"),
			WithCompletion(nil),
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNilValue)
	})
}

func TestCompletionError(t *testing.T) {
	err := NewCompletionError("test error message")
	require.Error(t, err)
	assert.Equal(t, "test error message", err.Error())
}

func TestGetCompletionIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		ref      *mcp.CompleteReference
		expected string
	}{
		{
			name:     "ref/prompt returns name",
			ref:      &mcp.CompleteReference{Type: "ref/prompt", Name: "greet"},
			expected: "greet",
		},
		{
			name:     "ref/resource returns URI",
			ref:      &mcp.CompleteReference{Type: "ref/resource", URI: "file:///data/"},
			expected: "file:///data/",
		},
		{
			name:     "unknown type returns type",
			ref:      &mcp.CompleteReference{Type: "ref/unknown"},
			expected: "ref/unknown",
		},
		{
			name:     "empty name for prompt returns empty",
			ref:      &mcp.CompleteReference{Type: "ref/prompt", Name: ""},
			expected: "",
		},
		{
			name:     "empty URI for resource returns empty",
			ref:      &mcp.CompleteReference{Type: "ref/resource", URI: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCompletionIdentifier(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}
