package mcpio

import (
	"errors"
	"fmt"
)

// MCP Error Handling Guide
//
// The Model Context Protocol (MCP) has different error patterns:
//
// ## Tools - Dual Error Types
// Tools can return errors that are either visible to LLMs or hidden:
//   - ToolError -> LLM sees the error and can potentially retry/correct
//   - Regular errors -> Hidden protocol errors for system/security issues
//
// ## Prompts & Resources - Protocol Only
// Prompts and resources only support protocol-level errors (never visible to LLMs).
//
// ## Example Tool Function
//   func apiTool(ctx context.Context, input APIInput) (APIOutput, error) {
//       // Protocol error (hidden from LLM) - security/system issue
//       if !isAuthorized(ctx) {
//           return APIOutput{}, fmt.Errorf("unauthorized access")
//       }
//
//       // LLM-visible error - user can correct this
//       if input.Query == "" {
//           return APIOutput{}, mcpio.NewToolError("Query parameter is required")
//       }
//
//       return APIOutput{Result: processQuery(input.Query)}, nil
//   }

// ToolError represents a user-facing tool execution error that becomes visible to LLMs.
// When returned from tool functions, the MCP SDK automatically wraps these in CallToolResult
// with IsError=true, allowing LLMs to see the error and potentially retry or self-correct.
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

// Empty string validation errors
var (
	ErrEmptyName             = errors.New("name cannot be empty")
	ErrEmptyVersion          = errors.New("version cannot be empty")
	ErrEmptyToolName         = errors.New("tool name cannot be empty")
	ErrEmptyPromptName       = errors.New("prompt name cannot be empty")
	ErrEmptyResourceURI      = errors.New("resource URI cannot be empty")
	ErrEmptyResourceTemplate = errors.New("resource template cannot be empty")
)

// Nil value validation errors
var (
	ErrNilSchema   = errors.New("schema cannot be nil")
	ErrNilFunction = errors.New("function cannot be nil")
	ErrNilServer   = errors.New("server cannot be nil")
)

// Runtime and operational errors
var (
	ErrDuplicateTool    = errors.New("tool already registered")
	ErrInvalidOperation = errors.New("invalid operation")
	ErrInvalidJSON      = errors.New("tool returned invalid JSON")
)
