package main

import (
	"context"
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

func main() {
	handler, err := mcpio.NewPromptHandler(
		mcpio.WithName("prompt-server"),
		mcpio.WithPrompt("greeter", "Generates a friendly greeting", greeterPrompt),
	)
	if err != nil {
		log.Fatal("Failed to create MCP handler:", err)
	}

	if err := handler.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
		log.Fatal("Failed to serve MCP via stdio:", err)
	}
}
