// This example demonstrates mcp-io's ElicitTyped for server-side elicitation.
// Compare with the SDK's manual schema construction pattern.
//
// SDK pattern:
//   schema := &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{...}}
//   result, err := serverSession.Elicit(ctx, &mcp.ElicitParams{Message: "...", RequestedSchema: schema})
//   // Manual result.Action checking and map[string]any type assertions
//
// mcp-io pattern:
//   result, err := mcpio.ElicitTyped[ServerConfig](ctx, elicitor, "...")
//   // Automatic schema generation, type-safe DecodeContent(), and helper methods

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

// Example_elicitation demonstrates mcp-io's type-safe elicitation with automatic schema generation.
//
// Key differences from the SDK:
//   - Automatic JSON schema generation from Go structs
//   - Type-safe DecodeContent() instead of untyped map[string]any
//   - Helper methods: IsAccepted(), IsDeclined(), IsCancelled()
//   - Cleaner tool function signatures
func Example_elicitation() {
	ctx := context.Background()

	// Define configuration structure for elicitation
	// mcp-io automatically generates JSON schema from this struct
	type ServerConfig struct {
		ServerEndpoint string `json:"serverEndpoint" jsonschema:"required,description:Server endpoint URL"`
		MaxRetries     int    `json:"maxRetries"     jsonschema:"minimum:1,maximum:10,description:Maximum retry attempts"`
		EnableLogs     bool   `json:"enableLogs"     jsonschema:"description:Enable debug logging"`
	}

	// Define tool input (empty in this example)
	type ConfigRequest struct{}

	// Define tool output
	type ConfigResponse struct {
		Endpoint    string `json:"endpoint"    jsonschema:"description:Configured server endpoint"`
		MaxRetries  int    `json:"maxRetries"  jsonschema:"description:Configured max retries"`
		LogsEnabled bool   `json:"logsEnabled" jsonschema:"description:Debug logging enabled"`
	}

	// Tool function that uses elicitation
	configureTool := func(ctx context.Context, toolCtx mcpio.RequestContext, input ConfigRequest) (ConfigResponse, error) {
		// Use mcp-io's elicitation API with automatic schema generation
		elicitor := mcpio.NewElicitor(toolCtx)
		result, err := mcpio.ElicitTyped[ServerConfig](ctx, elicitor, "Please provide your configuration settings")
		if err != nil {
			return ConfigResponse{}, fmt.Errorf("elicitation failed: %w", err)
		}

		// Use helper methods instead of manual string comparison
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

	// Create handler with configure tool
	handler, err := mcpio.NewHandler(
		mcpio.WithName("config-server"),
		mcpio.WithVersion("v1.0.0"),
		mcpio.WithTool("configure", "Request user configuration via elicitation", configureTool),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Set up in-memory transport
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

	// Create client with elicitation handler
	// Note: Never use elicitation for sensitive data like API keys or passwords
	client := mcp.NewClient(&mcp.Implementation{Name: "config-client", Version: "v1.0.0"}, &mcp.ClientOptions{
		ElicitationHandler: func(ctx context.Context, request *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			// In a real application, this would prompt the user for input
			fmt.Printf("Server requests: %s\n", request.Params.Message)

			return &mcp.ElicitResult{
				Action: mcpio.ElicitActionAccept,
				Content: map[string]any{
					"serverEndpoint": "https://api.example.com",
					"maxRetries":     float64(3), // JSON numbers are float64
					"enableLogs":     true,
				},
			}, nil
		},
	})

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}

	// Extract the configuration from the structured result
	if res.StructuredContent != nil {
		config, ok := res.StructuredContent.(map[string]any)
		if ok {
			fmt.Printf("Configuration received: Endpoint: %v, Max Retries: %.0f, Logs: %v\n",
				config["endpoint"],
				config["maxRetries"],
				config["logsEnabled"])
		}
	}

	// Output:
	// Server requests: Please provide your configuration settings
	// Configuration received: Endpoint: https://api.example.com, Max Retries: 3, Logs: true
}
