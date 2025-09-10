package mcpio

import (
	"errors"
	"fmt"
)

// ToolError represents a tool execution error that should be returned to the client
// as part of the CallToolResult with IsError: true. This allows LLMs to see the error
// and potentially retry or self-correct.
type ToolError struct {
	Message string
	Code    string // Optional error code for categorization
}

func (e *ToolError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("[%s] %s", e.Code, e.Message)
	}
	return e.Message
}

// NewToolError creates a new tool error with the given message
func NewToolError(message string) *ToolError {
	return &ToolError{Message: message}
}

// NewToolErrorWithCode creates a new tool error with message and code
func NewToolErrorWithCode(message, code string) *ToolError {
	return &ToolError{Message: message, Code: code}
}

// ValidationError is a convenience function for creating validation tool errors
func ValidationError(message string) *ToolError {
	return &ToolError{Message: message, Code: "VALIDATION_ERROR"}
}

// ProcessingError is a convenience function for creating processing tool errors
func ProcessingError(message string) *ToolError {
	return &ToolError{Message: message, Code: "PROCESSING_ERROR"}
}

// Sentinel errors for configuration validation
var (
	ErrEmptyName        = errors.New("name cannot be empty")
	ErrEmptyVersion     = errors.New("version cannot be empty")
	ErrEmptyToolName    = errors.New("tool name cannot be empty")
	ErrNilSchema        = errors.New("schema cannot be nil")
	ErrNilFunction      = errors.New("function cannot be nil")
	ErrNilServer        = errors.New("server cannot be nil")
	ErrNilEvaluator     = errors.New("evaluator cannot be nil")
	ErrDuplicateTool    = errors.New("tool already registered")
	ErrInvalidOperation = errors.New("invalid operation")
	ErrInvalidJSON      = errors.New("tool returned invalid JSON")
)
