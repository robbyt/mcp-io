// This example demonstrates the mcp-io equivalent of the basic MCP SDK example.
// It shows how mcp-io's functional options API and type-safe tool registration
// simplifies server creation compared to the SDK's direct API.
//
// Original SDK example:
// https://github.com/modelcontextprotocol/go-sdk/tree/main/examples/basic

package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/mcpwrapper"
)

// SayHiInput defines the input schema for the greet tool.
// The jsonschema tags provide descriptions for the LLM.
type SayHiInput struct {
	Name string `json:"name" jsonschema:"Person's name to greet"`
}

// SayHiOutput defines the output schema for the greet tool.
// mcp-io automatically generates schemas from these structs.
type SayHiOutput struct {
	Greeting string `json:"greeting" jsonschema:"The greeting message"`
}

// SayHi is the tool function using mcp-io's simplified signature.
// Compare with SDK version:
//
//	SDK:    func(ctx, *mcp.CallToolRequest, SayHiInput) (*mcp.CallToolResult, any, error)
//	mcp-io: func(ctx, mcpio.RequestContext, SayHiInput) (SayHiOutput, error)
//
// Benefits:
//   - No manual CallToolResult construction
//   - Typed output instead of 'any'
//   - Cleaner signature with RequestContext interface
func SayHi(ctx context.Context, toolCtx mcpio.RequestContext, input SayHiInput) (SayHiOutput, error) {
	return SayHiOutput{
		Greeting: "Hi " + input.Name,
	}, nil
}

// runGreeter creates an MCP server with a greeting tool, calls it via in-memory transport,
// and returns the greeting result. This function encapsulates the full round-trip for testing.
func runGreeter(ctx context.Context) (string, error) {
	// Create handler using mcp-io's functional options API
	// Compare with SDK:
	//   server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "v0.0.1"}, nil)
	//          mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)
	//   mcp-io: Single NewHandler call with functional options
	handler, err := mcpio.NewHandler(
		mcpio.WithName("greeter"),
		mcpio.WithVersion("v0.0.1"),
		mcpio.WithTool("greet", "say hi", SayHi),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create handler: %w", err)
	}

	// Set up in-memory transport using mcp-io's wrapper pattern
	// In production, you'd use handler.ServeHTTP() or handler.ServeStdio()
	sdkServer, ok := handler.GetServer().Unwrap().(*mcp.Server)
	if !ok {
		return "", fmt.Errorf("failed to unwrap SDK server")
	}
	wrappedServer, clientTransport := mcpwrapper.NewInMemoryServer(sdkServer)

	// Run the server in a goroutine
	go func() {
		if runErr := wrappedServer.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			log.Printf("server run error: %v", runErr)
		}
	}()

	// Connect client using the returned transport
	client := mcp.NewClient(&mcp.Implementation{Name: "client"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return "", fmt.Errorf("failed to connect client: %w", err)
	}
	defer func() {
		if closeErr := clientSession.Close(); closeErr != nil {
			log.Printf("error closing client session: %v", closeErr)
		}
	}()

	// Call the tool
	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "greet",
		Arguments: map[string]any{"name": "user"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to call tool: %w", err)
	}

	// Extract the greeting from the result
	// The result contains both Content (JSON text) and StructuredContent (typed data)
	if len(res.Content) > 0 {
		return res.Content[0].(*mcp.TextContent).Text, nil
	}

	return "", fmt.Errorf("no content in result")
}

func main() {
	ctx := context.Background()

	output, err := runGreeter(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(output)
}
