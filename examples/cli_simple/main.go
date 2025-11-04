package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"strings"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/mcpwrapper"
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
func toUpper(ctx context.Context, toolCtx mcpio.RequestContext, input TextInput) (TextOutput, error) {
	return TextOutput{Result: strings.ToUpper(input.Text)}, nil
}

func main() {
	listen := flag.String("listen", "", "HTTP listen address (e.g., ':8080'). If empty, uses stdio transport")
	flag.Parse()

	handler, err := mcpio.NewHandler(
		mcpio.WithName("simple-text-processor"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),
		// WithStateless enables stateless mode for CLI testing tools like the MCP Inspector.
		// Stateless mode allows single POST requests without session establishment.
		// For production with full MCP clients (Claude Desktop, SDK-based clients), you can
		// remove this option to use stateful sessions with Mcp-Session-Id headers.
		mcpio.WithHTTPTransport(mcpwrapper.WithStateless()),
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
		if err := handler.ServeStdio(context.Background()); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}
