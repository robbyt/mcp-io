package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	mcpio "github.com/robbyt/mcp-io"
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
func toUpper(ctx context.Context, toolCtx mcpio.ToolContext, input TextInput) (TextOutput, error) {
	return TextOutput{Result: strings.ToUpper(input.Text)}, nil
}

func main() {
	listen := flag.String("listen", "", "HTTP listen address (e.g., ':8080'). If empty, uses stdio transport")
	flag.Parse()

	handler, err := mcpio.NewHandler(
		mcpio.WithName("simple-text-processor"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),
	)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	if *listen != "" {
		// HTTP transport
		http.Handle("/mcp", handler)
		log.Printf("MCP HTTP server listening on %s/mcp", *listen)
		log.Fatal(http.ListenAndServe(*listen, nil))
	} else {
		// Stdio transport
		if err := handler.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}
