// This example demonstrates mcp-io's completion handler registration.
// Compare with the SDK's completion handler pattern.
//
// SDK pattern:
//
//	myCompletionHandler := func(_ context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
//	    var suggestions []string
//	    switch req.Params.Ref.Type {
//	    case "ref/prompt":
//	        suggestions = []string{"suggestion1", "suggestion2", "suggestion3"}
//	    case "ref/resource":
//	        suggestions = []string{"suggestion4", "suggestion5", "suggestion6"}
//	    default:
//	        return nil, fmt.Errorf("unrecognized content type %s", req.Params.Ref.Type)
//	    }
//	    return &mcp.CompleteResult{
//	        Completion: mcp.CompletionResultDetails{
//	            HasMore: false,
//	            Total:   len(suggestions),
//	            Values:  suggestions,
//	        },
//	    }, nil
//	}
//	server := mcp.NewServer(&mcp.Implementation{Name: "server"}, &mcp.ServerOptions{
//	    CompletionHandler: myCompletionHandler,
//	})
//
// mcp-io pattern:
//
//	handler, err := mcpio.NewHandler(
//	    mcpio.WithName("server"),
//	    mcpio.WithCompletion(func(ctx context.Context, reqCtx mcpio.RequestContext, ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
//	        var suggestions []string
//	        switch ref.Type {
//	        case "ref/prompt":
//	            suggestions = []string{"suggestion1", "suggestion2", "suggestion3"}
//	        case "ref/resource":
//	            suggestions = []string{"suggestion4", "suggestion5", "suggestion6"}
//	        default:
//	            return nil, mcpio.NewCompletionError(fmt.Sprintf("unrecognized content type %s", ref.Type))
//	        }
//	        return &mcpio.CompletionResult{Values: suggestions}, nil
//	    }),
//	)
package sdk_test

import (
	"context"
	"fmt"
	"log"

	mcpio "github.com/robbyt/mcp-io"
)

// Example_completion demonstrates mcp-io's completion handler registration.
// Compare with SDK: https://github.com/modelcontextprotocol/go-sdk/blob/main/examples/server/completion/main.go
func Example_completion() {
	// Define completion handler logic using mcp-io's simplified types
	// SDK version: func(_ context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error)
	// mcp-io version: func(context.Context, mcpio.RequestContext, mcpio.CompletionRef) (*mcpio.CompletionResult, error)
	myCompletionHandler := func(_ context.Context, _ mcpio.RequestContext, ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
		// In a real application, you'd implement actual completion logic here.
		// For this example, we return a fixed set of suggestions.
		var suggestions []string

		// Direct field access - no nested structs
		// SDK uses: req.Params.Ref.Type
		// mcp-io uses: ref.Type
		switch ref.Type {
		case "ref/prompt":
			suggestions = []string{"suggestion1", "suggestion2", "suggestion3"}
		case "ref/resource":
			suggestions = []string{"suggestion4", "suggestion5", "suggestion6"}
		default:
			// SDK uses: return nil, fmt.Errorf(...)
			// mcp-io provides typed error: mcpio.NewCompletionError()
			return nil, mcpio.NewCompletionError(fmt.Sprintf("unrecognized content type %s", ref.Type))
		}

		// Simple result construction
		// SDK requires: &mcp.CompleteResult{Completion: mcp.CompletionResultDetails{...}}
		// mcp-io uses: &mcpio.CompletionResult{...}
		return &mcpio.CompletionResult{
			Values:  suggestions,
			HasMore: false,
			Total:   len(suggestions),
		}, nil
	}

	// Create handler using functional options
	// SDK: mcp.NewServer(&mcp.Implementation{Name: "server"}, &mcp.ServerOptions{CompletionHandler: ...})
	// mcp-io: mcpio.NewHandler(mcpio.WithName(...), mcpio.WithCompletion(...))
	handler, err := mcpio.NewHandler(
		mcpio.WithName("server"),
		mcpio.WithCompletion(myCompletionHandler),
	)
	if err != nil {
		log.Fatal(err)
	}

	// No server running, just showing the configuration
	log.Println("MCP handler created with completion support (but not running).")
	log.Println("This example demonstrates configuration, not live interaction.")

	// Verify handler was created
	if handler != nil {
		fmt.Println("Handler configured successfully")
	}

	// Output:
	// Handler configured successfully
}
