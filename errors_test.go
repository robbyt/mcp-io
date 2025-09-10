package mcpio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolError_Error(t *testing.T) {
	tests := []struct {
		name     string
		toolErr  *ToolError
		expected string
	}{
		{
			name:     "message only",
			toolErr:  &ToolError{Message: "something went wrong"},
			expected: "something went wrong",
		},
		{
			name:     "message with code",
			toolErr:  &ToolError{Message: "validation failed", Code: "VALIDATION_ERROR"},
			expected: "[VALIDATION_ERROR] validation failed",
		},
		{
			name:     "empty message",
			toolErr:  &ToolError{Message: ""},
			expected: "",
		},
		{
			name:     "empty code",
			toolErr:  &ToolError{Message: "error", Code: ""},
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.toolErr.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewToolError(t *testing.T) {
	message := "test error message"
	err := NewToolError(message)

	require.Error(t, err)
	assert.Equal(t, message, err.Message)
	assert.Empty(t, err.Code)
	assert.Equal(t, message, err.Error())
}

func TestNewToolErrorWithCode(t *testing.T) {
	message := "test error"
	code := "TEST_ERROR"
	err := NewToolErrorWithCode(message, code)

	require.Error(t, err)
	assert.Equal(t, message, err.Message)
	assert.Equal(t, code, err.Code)
	assert.Equal(t, "[TEST_ERROR] test error", err.Error())
}

func TestValidationError(t *testing.T) {
	message := "invalid input"
	err := ValidationError(message)

	require.Error(t, err)
	assert.Equal(t, message, err.Message)
	assert.Equal(t, "VALIDATION_ERROR", err.Code)
	assert.Equal(t, "[VALIDATION_ERROR] invalid input", err.Error())
}

func TestProcessingError(t *testing.T) {
	message := "failed to process"
	err := ProcessingError(message)

	require.Error(t, err)
	assert.Equal(t, message, err.Message)
	assert.Equal(t, "PROCESSING_ERROR", err.Code)
	assert.Equal(t, "[PROCESSING_ERROR] failed to process", err.Error())
}

func TestToolErrorImplementsError(t *testing.T) {
	var err error = &ToolError{Message: "test"}
	require.Error(t, err)
	assert.Equal(t, "test", err.Error())
}
