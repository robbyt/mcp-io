package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
)

// Completion handler that provides suggestions for prompts and resources
func myCompletionHandler(ctx context.Context, reqCtx mcpio.RequestContext, ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
	session := reqCtx.GetSession()
	session.Logger().Info(fmt.Sprintf("Completion requested for type=%s, name=%s, argument=%s",
		ref.Type, ref.Name, ref.Argument))

	switch ref.Type {
	case "ref/prompt":
		return handlePromptCompletion(ref)
	case "ref/resource":
		return handleResourceCompletion(ref)
	default:
		return nil, mcpio.NewCompletionError(fmt.Sprintf("unsupported reference type: %s", ref.Type))
	}
}

func handlePromptCompletion(ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
	// Example: Complete language argument for "greet" prompt
	if ref.Name == "greet" && ref.Argument == "language" {
		return &mcpio.CompletionResult{
			Values:  []string{"English", "Spanish", "French", "German", "Italian", "Portuguese"},
			HasMore: false,
		}, nil
	}

	// Example: Complete style argument for "summarize" prompt
	if ref.Name == "summarize" && ref.Argument == "style" {
		return &mcpio.CompletionResult{
			Values:  []string{"concise", "detailed", "technical", "casual"},
			HasMore: false,
		}, nil
	}

	return &mcpio.CompletionResult{
		Values: []string{}, // No suggestions
	}, nil
}

func handleResourceCompletion(ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
	// Example: Complete file paths based on partial input
	// In a real implementation, you might query a filesystem or database

	if strings.HasPrefix(ref.Name, "file:///data") {
		return &mcpio.CompletionResult{
			Values: []string{
				"file:///data/users.json",
				"file:///data/products.json",
				"file:///data/orders.json",
			},
			HasMore: false,
		}, nil
	}

	// Default: suggest some common resource URIs
	return &mcpio.CompletionResult{
		Values: []string{
			"file:///data/",
			"http://api.example.com/",
		},
		HasMore: true,
		Total:   10, // Indicate more completions available
	}, nil
}

func main() {
	listen := flag.String("listen", "", "HTTP listen address (e.g., ':8080'). If empty, uses stdio")
	flag.Parse()

	handler, err := mcpio.NewHandler(
		mcpio.WithName("completion-demo"),
		mcpio.WithVersion("1.0.0"),

		// Add some prompts that use completion
		mcpio.WithPromptWithArgs("greet", "Generate a greeting", []*mcp.PromptArgument{
			{Name: "language", Description: "Language for greeting", Required: true},
		}, func(ctx context.Context, reqCtx mcpio.RequestContext, args map[string]any) (*mcpio.PromptResult, error) {
			lang := args["language"].(string)
			return &mcpio.PromptResult{
				Messages: []mcpio.PromptMessage{
					{Role: "user", Content: fmt.Sprintf("Say hello in %s", lang)},
				},
			}, nil
		}),

		// Register completion handler
		mcpio.WithCompletion(myCompletionHandler),
	)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	if *listen != "" {
		// HTTP mode
		http.Handle("/mcp", handler)
		log.Printf("Completion demo server listening on %s/mcp", *listen)
		log.Fatal(http.ListenAndServe(*listen, nil))
	} else {
		// Stdio mode
		if err := handler.ServeStdio(context.Background()); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}
