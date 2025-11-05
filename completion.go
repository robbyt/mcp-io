package mcpio

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP completion reference types
const (
	RefTypePrompt   = "ref/prompt"
	RefTypeResource = "ref/resource"
)

// CompletionFunc is the function signature for user-defined completion handlers.
// The function receives context, request metadata, completion reference details,
// and returns completion suggestions with optional metadata.
//
// Example:
//
//	func myCompletionHandler(ctx context.Context, reqCtx mcpio.RequestContext, ref CompletionRef) (*CompletionResult, error) {
//	    if ref.Type == mcpio.RefTypePrompt && ref.Name == "greet" {
//	        return &CompletionResult{
//	            Values: []string{"Hello", "Hi", "Greetings"},
//	        }, nil
//	    }
//	    return nil, mcpio.NewCompletionError("unsupported reference type")
//	}
type CompletionFunc func(context.Context, RequestContext, CompletionRef) (*CompletionResult, error)

// CompletionRef contains information about what is being completed.
// This maps to mcp.CompleteRequest.Params.Ref with user-friendly field names.
type CompletionRef struct {
	// Type indicates what kind of reference this is (e.g., "ref/prompt", "ref/resource")
	Type string

	// Name is the name of the prompt being completed (used with "ref/prompt")
	Name string

	// URI is the resource URI being completed (used with "ref/resource")
	URI string

	// Argument is the specific argument name being completed within a prompt
	Argument string

	// Value is the current input value for the argument being completed
	Value string

	// Context contains previously-resolved arguments for dependency-aware completions.
	// For example, if completing a "style" argument and the user has already selected
	// language="Spanish", Context would contain {"language": "Spanish"}.
	Context map[string]string
}

// CompletionResult represents completion suggestions returned by a completion handler.
// This provides a simplified, opinionated interface compared to the raw SDK types.
type CompletionResult struct {
	// Values contains the completion suggestions (required, max recommended: 100)
	Values []string

	// HasMore indicates if additional completions are available beyond this set (optional)
	// Default: false (all completions provided)
	HasMore bool

	// Total indicates the total number of completions available (optional)
	// Use 0 or omit if unknown. When HasMore is true, Total > len(Values)
	Total int

	// Meta is protocol-reserved metadata for extensibility (optional)
	Meta map[string]any
}

// Validate ensures the CompletionResult is well-formed before returning to the SDK
func (r *CompletionResult) Validate() error {
	if len(r.Values) == 0 {
		return ErrNoCompletions
	}
	if r.Total > 0 && r.Total < len(r.Values) {
		return ErrInvalidTotal
	}
	return nil
}

// getCompletionIdentifier extracts the appropriate identifier from a completion reference.
// Returns the prompt name for RefTypePrompt, resource URI for RefTypeResource, or the type as fallback.
func getCompletionIdentifier(ref *mcp.CompleteReference) string {
	switch ref.Type {
	case RefTypePrompt:
		return ref.Name
	case RefTypeResource:
		return ref.URI
	default:
		return ref.Type
	}
}

// createCompletionHandler wraps a user-provided CompletionFunc to match the SDK's CompletionHandler signature.
// This adapter follows the same pattern as createToolHandler, createPromptHandler, createResourceHandler.
//
// The adapter:
//   - Extracts completion reference data from the MCP request
//   - Creates a RequestContext with session and metadata
//   - Calls the user's completion function
//   - Validates the result
//   - Converts to SDK format
func createCompletionHandler(fn CompletionFunc) func(context.Context, *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	return func(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
		// Create request context with all MCP metadata
		// Use prompt name or resource URI as identifier based on reference type
		reqCtx := newRequestContext(getCompletionIdentifier(req.Params.Ref), req.Params, req.Session, req.Extra)

		// Extract reference information from SDK request
		ref := CompletionRef{
			Type:     req.Params.Ref.Type,
			Name:     req.Params.Ref.Name,
			URI:      req.Params.Ref.URI,
			Argument: req.Params.Argument.Name,
			Value:    req.Params.Argument.Value,
		}

		// Extract context if present (nil-safe)
		if req.Params.Context != nil {
			ref.Context = req.Params.Context.Arguments
		}

		// Execute user function with request context
		result, err := fn(ctx, reqCtx, ref)
		if err != nil {
			// Check if it's a user-facing completion error
			var compErr *CompletionError
			if errors.As(err, &compErr) {
				// Return as protocol error - completions don't have isError field like tools
				return nil, err
			}
			// Protocol-level error - return as-is
			return nil, err
		}

		// Validate the result before converting to SDK format
		if err := result.Validate(); err != nil {
			return nil, fmt.Errorf("invalid completion result: %w", err)
		}

		// Convert to SDK format
		total := result.Total
		if total == 0 {
			// If total not provided, default to number of values returned (no more available)
			total = len(result.Values)
		}

		return &mcp.CompleteResult{
			Meta: result.Meta,
			Completion: mcp.CompletionResultDetails{
				Values:  result.Values,
				Total:   total,
				HasMore: result.HasMore,
			},
		}, nil
	}
}
