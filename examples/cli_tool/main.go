package main

import (
	"context"
	"log"
	"os"
	"strings"

	mcpio "github.com/robbyt/go-mcpio"
)

// Text processing input/output types
type TextInput struct {
	Text string `json:"text" jsonschema:"Text to process"`
}

type TextOutput struct {
	Result string `json:"result" jsonschema:"Processed text"`
}

// Count input/output types
type CountInput struct {
	Text string `json:"text" jsonschema:"Text to count"`
	Type string `json:"type" jsonschema:"What to count: words, chars, or lines"`
}

type CountOutput struct {
	Count int    `json:"count" jsonschema:"The count result"`
	Type  string `json:"type"  jsonschema:"What was counted"`
}

// Text processing tools
func toUpperCase(ctx context.Context, input TextInput) (TextOutput, error) {
	return TextOutput{Result: strings.ToUpper(input.Text)}, nil
}

func toLowerCase(ctx context.Context, input TextInput) (TextOutput, error) {
	return TextOutput{Result: strings.ToLower(input.Text)}, nil
}

func reverseText(ctx context.Context, input TextInput) (TextOutput, error) {
	runes := []rune(input.Text)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return TextOutput{Result: string(runes)}, nil
}

func countText(ctx context.Context, input CountInput) (CountOutput, error) {
	var count int

	switch input.Type {
	case "words":
		words := strings.Fields(input.Text)
		count = len(words)
	case "chars":
		count = len([]rune(input.Text))
	case "lines":
		lines := strings.Split(input.Text, "\n")
		count = len(lines)
	default:
		return CountOutput{}, mcpio.ValidationError("unsupported count type: " + input.Type)
	}

	return CountOutput{Count: count, Type: input.Type}, nil
}

func main() {
	// Create MCP handler for CLI use
	handler, err := mcpio.New(
		mcpio.WithName("text-processor"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("to_upper", "Convert text to uppercase", toUpperCase),
		mcpio.WithTool("to_lower", "Convert text to lowercase", toLowerCase),
		mcpio.WithTool("reverse", "Reverse text", reverseText),
		mcpio.WithTool("count", "Count words, characters, or lines", countText),
	)
	if err != nil {
		log.Fatal("Failed to create MCP handler:", err)
	}

	// Serve via stdio (standard for CLI MCP tools)
	if err := handler.ServeStdio(os.Stdin, os.Stdout); err != nil {
		log.Fatal("Failed to serve MCP via stdio:", err)
	}
}
