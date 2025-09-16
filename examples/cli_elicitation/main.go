package main

import (
	"context"
	"fmt"
	"log"

	mcpio "github.com/robbyt/mcp-io"
)

// UserConfig represents basic configuration that we'll elicit from the user
type UserConfig struct {
	Name        string `json:"name"        jsonschema:"description:Your name"`
	Email       string `json:"email"       jsonschema:"format:email,description:Your email address"`
	Environment string `json:"environment" jsonschema:"description:Target environment,enum:development,enum:staging,enum:production"`
}

// setupApplication demonstrates basic elicitation with a single step
func setupApplication(ctx context.Context, capability mcpio.ElicitationCapability, input struct{}) (map[string]any, error) {
	// Elicit user configuration
	result, err := mcpio.ElicitTypedResult[UserConfig](ctx, capability, "Please provide your application configuration:")
	if err != nil {
		return nil, fmt.Errorf("failed to elicit configuration: %w", err)
	}

	// Handle user response
	if !result.IsAccepted() {
		return map[string]any{
			"status": "cancelled",
			"reason": fmt.Sprintf("User %s the configuration", result.Action),
		}, nil
	}

	// Return the configuration
	return map[string]any{
		"status": "configured",
		"config": result.GetContent(),
		"message": fmt.Sprintf("Successfully configured application for %s environment",
			result.GetContent()["environment"]),
	}, nil
}

// interactivePrompt demonstrates elicitation from within a prompt
func interactivePrompt(ctx context.Context, capability mcpio.ElicitationCapability, args map[string]any) (*mcpio.PromptResult, error) {
	documentType := "general"
	if dt, ok := args["document_type"].(string); ok {
		documentType = dt
	}

	// Elicit specific requirements for the document
	result, err := mcpio.ElicitSimple(ctx, capability,
		fmt.Sprintf("What specific requirements do you have for your %s document?", documentType),
		"requirements", "Describe your specific needs and requirements")
	if err != nil {
		return nil, fmt.Errorf("failed to elicit requirements: %w", err)
	}

	requirements := "No specific requirements provided"
	if result.Action == "accept" {
		if content := result.GetContent(); content != nil {
			if req, ok := content["requirements"].(string); ok && req != "" {
				requirements = req
			}
		}
	}

	// Generate the prompt based on the elicited requirements
	systemMessage := fmt.Sprintf(`You are a professional document writer. Create a %s document that meets these specific requirements: %s

Focus on clarity, professionalism, and meeting the user's stated needs.`, documentType, requirements)

	userMessage := fmt.Sprintf("Please create a %s document incorporating the requirements I specified.", documentType)

	return &mcpio.PromptResult{
		Description: fmt.Sprintf("Interactive %s document generation with custom requirements", documentType),
		Messages: []mcpio.PromptMessage{
			{Role: "system", Content: systemMessage},
			{Role: "user", Content: userMessage},
		},
	}, nil
}

func main() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("elicitation-demo"),
		mcpio.WithVersion("1.0.0"),

		// Session-aware tools that can elicit information
		mcpio.WithSessionTool("setup_application", "Interactive application setup with configuration elicitation", setupApplication),

		// Session-aware prompt that can elicit requirements
		mcpio.WithSessionPrompt("interactive_document", "Generate documents with elicited requirements", interactivePrompt),
	)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	// Run the server on stdio transport
	if err := handler.ServeStdio(context.Background(), nil, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
