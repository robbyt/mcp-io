// This example demonstrates mcp-io's type-safe tool registration.
// Compare with the SDK's direct tool registration pattern.
//
// SDK pattern:
//   server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "v0.0.1"}, nil)
//   mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, handler)
//
// mcp-io pattern:
//   handler, err := mcpio.NewHandler(
//     mcpio.WithName("greeter"),
//     mcpio.WithVersion("v0.0.1"),
//     mcpio.WithTool("greet", "say hi", handler),
//   )

package sdk_test

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/mcpwrapper"
)

// Example_tools demonstrates mcp-io's functional options API and type-safe tool registration.
//
// Key differences from the SDK:
//   - Type-safe input/output with automatic schema generation
//   - Functional options pattern for configuration
//   - Simplified function signatures using Go generics
//   - Error handling instead of panics
func Example_tools() {
	ctx := context.Background()

	// Define input schema with jsonschema tags for automatic schema generation
	type SayHiInput struct {
		Name string `json:"name" jsonschema:"Person's name to greet"`
	}

	// Define output schema - mcp-io automatically generates JSON schemas from these structs
	type SayHiOutput struct {
		Greeting string `json:"greeting" jsonschema:"The greeting message"`
	}

	// Tool function using mcp-io's simplified signature
	// SDK version:    func(ctx, *mcp.CallToolRequest, SayHiInput) (*mcp.CallToolResult, any, error)
	// mcp-io version: func(ctx, mcpio.RequestContext, SayHiInput) (SayHiOutput, error)
	sayHi := func(ctx context.Context, toolCtx mcpio.RequestContext, input SayHiInput) (SayHiOutput, error) {
		return SayHiOutput{Greeting: "Hi " + input.Name}, nil
	}

	// Create handler using functional options API
	handler, err := mcpio.NewHandler(
		mcpio.WithName("greeter"),
		mcpio.WithVersion("v0.0.1"),
		mcpio.WithTool("greet", "say hi", sayHi),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Set up in-memory transport for testing
	// In production, use handler.ServeHTTP() or handler.ServeStdio()
	sdkServer, ok := handler.GetServer().Unwrap().(*mcp.Server)
	if !ok {
		log.Fatal("failed to unwrap SDK server")
	}
	wrappedServer, clientTransport := mcpwrapper.NewInMemoryServer(sdkServer)

	// Run the server
	go func() {
		if runErr := wrappedServer.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			log.Printf("server run error: %v", runErr)
		}
	}()

	// Create client and connect
	client := mcp.NewClient(&mcp.Implementation{Name: "client"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}

	// Extract and print the result
	if len(res.Content) > 0 {
		fmt.Println(res.Content[0].(*mcp.TextContent).Text)
	}

	// Output: {"greeting":"Hi user"}
}
