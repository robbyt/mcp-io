package main

import (
	"context"
	"log"
	"os"
	"strings"

	mcpio "github.com/robbyt/go-mcpio"
)

// Input schema
type TextInput struct {
	Text string `json:"text" jsonschema:"Text to convert to uppercase"`
}

// Output schema
type TextOutput struct {
	Result string `json:"result" jsonschema:"Uppercase text"`
}

// Tool function
func toUpper(ctx context.Context, input TextInput) (TextOutput, error) {
	return TextOutput{Result: strings.ToUpper(input.Text)}, nil
}

func main() {
	handler, err := mcpio.New(
		mcpio.WithName("simple-text-processor"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),
	)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	if err := handler.ServeStdio(os.Stdin, os.Stdout); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
