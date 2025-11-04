// This example demonstrates the mcp-io equivalent of the SDK elicitation example.
// It shows how mcp-io's ElicitTyped simplifies elicitation with automatic schema
// generation from Go structs, compared to the SDK's manual schema construction.
//
// Original SDK example:
// https://github.com/modelcontextprotocol/go-sdk/tree/main/examples/elicitation

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

// ServerConfig defines the configuration structure for elicitation.
// mcp-io automatically generates a JSON schema from this struct.
// Compare with SDK: manual jsonschema.Schema construction with Properties map
type ServerConfig struct {
	ServerEndpoint string `json:"serverEndpoint" jsonschema:"required,description:Server endpoint URL"`
	MaxRetries     int    `json:"maxRetries"     jsonschema:"minimum:1,maximum:10,description:Maximum retry attempts"`
	EnableLogs     bool   `json:"enableLogs"     jsonschema:"description:Enable debug logging"`
}

// ConfigRequest is the input for the configure tool (empty in this example).
type ConfigRequest struct{}

// ConfigResponse is the output containing the received configuration.
type ConfigResponse struct {
	Endpoint    string `json:"endpoint"    jsonschema:"description:Configured server endpoint"`
	MaxRetries  int    `json:"maxRetries"  jsonschema:"description:Configured max retries"`
	LogsEnabled bool   `json:"logsEnabled" jsonschema:"description:Debug logging enabled"`
}

// configureTool demonstrates server-side elicitation using mcp-io's API.
// Compare with SDK version:
//
//	SDK:    serverSession.Elicit(ctx, &mcp.ElicitParams{Message: "...", RequestedSchema: schema})
//	        Manual schema construction and result.Action checking
//	mcp-io: mcpio.ElicitTyped[ServerConfig](ctx, elicitor, "...")
//	        Automatic schema generation and typed result helpers
//
// Benefits:
//   - No manual schema construction - generated from ServerConfig struct
//   - Type-safe DecodeContent() instead of untyped map[string]any
//   - Helper methods: IsAccepted(), IsDeclined(), IsCancelled()
func configureTool(ctx context.Context, toolCtx mcpio.RequestContext, input ConfigRequest) (ConfigResponse, error) {
	// Use mcp-io's elicitation API with automatic schema generation
	elicitor := mcpio.NewElicitor(toolCtx)
	result, err := mcpio.ElicitTyped[ServerConfig](ctx, elicitor, "Please provide your configuration settings")
	if err != nil {
		return ConfigResponse{}, fmt.Errorf("elicitation failed: %w", err)
	}

	// Use mcp-io's helper methods instead of manual string comparison
	if !result.IsAccepted() {
		if result.IsDeclined() {
			return ConfigResponse{}, mcpio.NewToolError("configuration was declined by user")
		}
		return ConfigResponse{}, mcpio.NewToolError("configuration was cancelled by user")
	}

	// Use type-safe DecodeContent() instead of untyped map access
	var config ServerConfig
	if err := result.DecodeContent(&config); err != nil {
		return ConfigResponse{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ConfigResponse{
		Endpoint:    config.ServerEndpoint,
		MaxRetries:  config.MaxRetries,
		LogsEnabled: config.EnableLogs,
	}, nil
}

// runElicitationExample creates an MCP server with elicitation support, calls the configure
// tool via in-memory transport, and returns the formatted result. This function encapsulates
// the full round-trip for testing.
func runElicitationExample(ctx context.Context) (string, error) {
	// Create handler with configure tool
	handler, err := mcpio.NewHandler(
		mcpio.WithName("config-server"),
		mcpio.WithVersion("v1.0.0"),
		mcpio.WithTool("configure", "Request user configuration via elicitation", configureTool),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create handler: %w", err)
	}

	// Set up in-memory transport using mcp-io's wrapper pattern
	sdkServer := handler.GetServer().Unwrap().(*mcp.Server)
	wrappedServer, clientTransport := mcpwrapper.NewInMemoryServer(sdkServer)

	// Run the server in a goroutine
	go func() {
		if runErr := wrappedServer.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			log.Printf("server run error: %v", runErr)
		}
	}()

	// Create client with elicitation handler
	// Note: Never use elicitation for sensitive data like API keys or passwords
	client := mcp.NewClient(&mcp.Implementation{Name: "config-client", Version: "v1.0.0"}, &mcp.ClientOptions{
		ElicitationHandler: func(ctx context.Context, request *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			// In a real application, this would prompt the user for input
			// Here we simulate user providing configuration data
			fmt.Printf("Server requests: %s\n", request.Params.Message)

			return &mcp.ElicitResult{
				Action: mcpio.ElicitActionAccept, // Use mcp-io's constant
				Content: map[string]any{
					"serverEndpoint": "https://api.example.com",
					"maxRetries":     float64(3),
					"enableLogs":     true,
				},
			}, nil
		},
	})

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return "", fmt.Errorf("failed to connect client: %w", err)
	}
	defer func() {
		if closeErr := clientSession.Close(); closeErr != nil {
			log.Printf("error closing client session: %v", closeErr)
		}
	}()

	// Call the configure tool which will trigger elicitation
	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "configure",
		Arguments: map[string]any{},
	})
	if err != nil {
		return "", fmt.Errorf("failed to call tool: %w", err)
	}

	// Extract the configuration from the structured result
	if res.StructuredContent != nil {
		config, ok := res.StructuredContent.(map[string]any)
		if ok {
			return fmt.Sprintf("Configuration received: Endpoint: %v, Max Retries: %.0f, Logs: %v",
				config["endpoint"],
				config["maxRetries"],
				config["logsEnabled"]), nil
		}
	}

	return "", fmt.Errorf("no structured content in result")
}

func main() {
	ctx := context.Background()

	output, err := runElicitationExample(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(output)

	// Output:
	// Server requests: Please provide your configuration settings
	// Configuration received: Endpoint: https://api.example.com, Max Retries: 3, Logs: true
}
