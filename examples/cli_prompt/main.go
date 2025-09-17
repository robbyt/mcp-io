package main

import (
	"context"
	"fmt"
	"log"
	"os"

	mcpio "github.com/robbyt/mcp-io"
)

func greeterPrompt(ctx context.Context, args map[string]any) (*mcpio.PromptResult, error) {
	name, _ := args["name"].(string)
	if name == "" {
		name = "World"
	}
	return &mcpio.PromptResult{
		Messages: []mcpio.PromptMessage{
			{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
			{
				Role:    "user",
				Content: "Create a friendly greeting for " + name,
			},
		},
	}, nil
}

// DocumentPromptArgs demonstrates typed prompt arguments with schema generation
type DocumentPromptArgs struct {
	DocumentType string `json:"documentType"     jsonschema:"description:Type of document to generate"`
	Topic        string `json:"topic"            jsonschema:"description:Main topic or subject"`
	Tone         string `json:"tone"             jsonschema:"description:Writing tone (formal, casual, etc)"`
	Length       int    `json:"length,omitempty" jsonschema:"description:Target length in words,minimum:50,maximum:2000"`
}

// documentPrompt demonstrates a typed prompt function with automatic schema generation
func documentPrompt(ctx context.Context, args DocumentPromptArgs) (*mcpio.PromptResult, error) {
	// Use typed arguments directly - no casting needed!
	content := fmt.Sprintf("Write a %s about %s using a %s tone", args.DocumentType, args.Topic, args.Tone)
	if args.Length > 0 {
		content += fmt.Sprintf(" in approximately %d words", args.Length)
	}

	return &mcpio.PromptResult{
		Messages: []mcpio.PromptMessage{
			{
				Role:    "system",
				Content: "You are an expert writer who creates high-quality content.",
			},
			{
				Role:    "user",
				Content: content,
			},
		},
		Description: "Generated writing prompt based on user specifications",
	}, nil
}

func main() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("prompt-server"),
		mcpio.WithPrompt("greeter", "Generates a friendly greeting", greeterPrompt),
		mcpio.WithTypedPrompt("document_writer", "Generates writing prompts with typed arguments", documentPrompt),
	)
	if err != nil {
		log.Fatal("Failed to create MCP handler:", err)
	}

	if err := handler.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
		log.Fatal("Failed to serve MCP via stdio:", err)
	}
}
