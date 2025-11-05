package mcpio

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CompletionFunc is the function signature for user-defined completion handlers.
// The function receives context, request metadata, completion reference details,
// and returns completion suggestions with optional metadata.
//
// Example:
//
//	func myCompletionHandler(ctx context.Context, reqCtx mcpio.RequestContext, ref CompletionRef) (*CompletionResult, error) {
//	    if ref.Type == "ref/prompt" && ref.Name == "greet" {
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

	// Name is the name of the prompt or resource being completed (optional)
	Name string

	// Argument is the specific argument name being completed within a prompt (optional)
	Argument string
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
		// Use empty string as identifier since completions aren't named resources
		reqCtx := newRequestContext("", req.Params, req.Session, req.Extra)

		// Extract reference information from SDK request
		ref := CompletionRef{
			Type:     req.Params.Ref.Type,
			Name:     req.Params.Ref.Name,
			Argument: req.Params.Argument.Name,
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
			Completion: mcp.CompletionResultDetails{
				Values:  result.Values,
				Total:   total,
				HasMore: result.HasMore,
			},
		}, nil
	}
}
