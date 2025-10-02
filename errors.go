package mcpio

import (
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP Tool Error Handling Guide
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

// Common tool error codes for categorizing user-facing tool errors
const (
	ErrorCodeValidation       = "VALIDATION_ERROR"     // Input validation failures
	ErrorCodeProcessing       = "PROCESSING_ERROR"     // Data processing issues
	ErrorCodeRateLimit        = "RATE_LIMITED"         // API rate limiting
	ErrorCodeInsufficientData = "INSUFFICIENT_DATA"    // Missing required information
	ErrorCodeAuthentication   = "AUTHENTICATION_ERROR" // Authentication failures
	ErrorCodeAuthorization    = "AUTHORIZATION_ERROR"  // Authorization/permission issues
	ErrorCodeNotFound         = "NOT_FOUND"            // Resource or entity not found
	ErrorCodeTimeout          = "TIMEOUT"              // Operation timeout
)

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

// NewToolError creates a new tool error with the given message.
//
// Tool errors are user-facing errors that become visible to LLMs in the CallToolResult.
// When returned from tool functions, the MCP SDK automatically:
//   - Sets CallToolResult.IsError = true
//   - Includes the error message in CallToolResult.Content
//   - Allows the LLM to see the error and potentially retry or self-correct
//
// Use tool errors for validation failures, input problems, or other issues
// that the LLM should be aware of and can potentially fix.
//
// For system-level errors (authentication, network, etc.) that should be
// hidden from the LLM, return regular Go errors instead.
func NewToolError(message string) *ToolError {
	return &ToolError{Message: message}
}

// NewToolErrorWithCode creates a new tool error with message and error code.
//
// The error code provides categorization for different types of tool errors,
// which can be useful for client-side error handling, analytics, and structured
// error responses. Use the ErrorCode* constants for common error types.
//
// Like NewToolError, these become visible to LLMs through CallToolResult.
func NewToolErrorWithCode(message, code string) *ToolError {
	return &ToolError{Message: message, Code: code}
}

// ValidationError is a convenience function for creating validation tool errors
func ValidationError(message string) *ToolError {
	return &ToolError{Message: message, Code: ErrorCodeValidation}
}

// ProcessingError is a convenience function for creating processing tool errors
func ProcessingError(message string) *ToolError {
	return &ToolError{Message: message, Code: ErrorCodeProcessing}
}

// ResourceNotFoundError returns a protocol-level error indicating that a resource
// could not be found. This provides the proper MCP error code (-32002) for resource
// not found conditions.
func ResourceNotFoundError(uri string) error {
	return mcp.ResourceNotFoundError(uri)
}

// Sentinel errors for configuration validation

// Configuration validation errors
var (
	ErrEmptyValue = errors.New("value cannot be empty")
	ErrNilValue   = errors.New("value cannot be nil")
)

// Schema validation errors
var (
	ErrInvalidJSONSchema  = errors.New("invalid JSON schema")
	ErrNoSchemaProperties = errors.New("schema has no properties")
)

// Runtime and operational errors
var (
	ErrInvalidJSON = errors.New("tool returned invalid JSON")
)

// Handler option type compatibility errors
var (
	ErrIncompatibleHandler = errors.New("option not compatible with this handler type")
)
