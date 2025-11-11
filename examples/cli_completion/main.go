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
	case mcpio.RefTypePrompt:
		return handlePromptCompletion(ref)
	case mcpio.RefTypeResource:
		return handleResourceCompletion(ref)
	default:
		return nil, mcpio.NewCompletionError(fmt.Sprintf("unsupported reference type: %s", ref.Type))
	}
}

func handlePromptCompletion(ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
	// Example: Complete language argument for "greet" prompt
	if ref.Name == "greet" && ref.Argument == "language" {
		// Filter based on current input value
		if ref.Value != "" {
			filtered := []string{}
			allLangs := []string{"English", "Spanish", "French", "German", "Italian", "Portuguese"}
			for _, lang := range allLangs {
				if strings.HasPrefix(strings.ToLower(lang), strings.ToLower(ref.Value)) {
					filtered = append(filtered, lang)
				}
			}
			return &mcpio.CompletionResult{Values: filtered}, nil
		}

		return &mcpio.CompletionResult{
			Values:  []string{"English", "Spanish", "French", "German", "Italian", "Portuguese"},
			HasMore: false,
		}, nil
	}

	// Example: Context-aware completion for "style" argument
	// Provide different style options based on previously-selected language
	if ref.Name == "greet" && ref.Argument == "style" {
		if lang, ok := ref.Context["language"]; ok {
			switch lang {
			case "Spanish":
				return &mcpio.CompletionResult{
					Values: []string{"formal", "informal"},
				}, nil
			case "French":
				return &mcpio.CompletionResult{
					Values: []string{"formel", "informel"},
				}, nil
			default:
				return &mcpio.CompletionResult{
					Values: []string{"casual", "professional"},
				}, nil
			}
		}

		// No context available, provide generic styles
		return &mcpio.CompletionResult{
			Values: []string{"casual", "professional", "formal"},
		}, nil
	}

	return &mcpio.CompletionResult{
		Values: []string{}, // No suggestions
	}, nil
}

func handleResourceCompletion(ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
	// Example: Complete resource URIs using the URI field
	// In a real implementation, you might query a filesystem or database

	// Use URI field to determine what to complete
	if strings.HasPrefix(ref.URI, "file:///data") {
		// Filter based on current input value if present
		if ref.Value != "" && strings.Contains(ref.Value, "user") {
			return &mcpio.CompletionResult{
				Values: []string{"file:///data/users.json"},
			}, nil
		}

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
			{Name: "style", Description: "Style of greeting", Required: false},
		}, func(ctx context.Context, reqCtx mcpio.RequestContext, args map[string]any) (*mcpio.PromptResult, error) {
			lang := args["language"].(string)
			style, _ := args["style"].(string)
			prompt := fmt.Sprintf("Say hello in %s", lang)
			if style != "" {
				prompt += fmt.Sprintf(" with a %s style", style)
			}
			return &mcpio.PromptResult{
				Messages: []mcpio.PromptMessage{
					{Role: "user", Content: prompt},
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
		if err := handler.Run(context.Background()); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}
