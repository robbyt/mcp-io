package mcpio

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PromptFunc is the function signature for user-defined prompt handlers.
// The function receives a context and a map of arguments, and returns a PromptResult with an optional error.
type PromptFunc func(context.Context, map[string]any) (*PromptResult, error)

// PromptResult represents the result of prompt generation
type PromptResult struct {
	Messages    []PromptMessage `json:"messages"`
	Description string          `json:"description,omitempty"`
}

// PromptMessage represents a single message in a prompt
type PromptMessage struct {
	Role    string `json:"role"` // "user", "assistant", "system"
	Content string `json:"content"`
}

// NewPromptHandler creates a new MCP handler that only supports prompts.
// For mixed resource types, use NewHandler instead.
func NewPromptHandler(opts ...Option) (*Handler, error) {
	// Pre-validate that only prompts are configured
	cfg := &handlerConfig{
		name:              "mcp-server",
		version:           "1.0.0",
		tools:             make([]toolRegisterFunc, 0),
		prompts:           make([]promptRegisterFunc, 0),
		resources:         make([]resourceRegisterFunc, 0),
		resourceTemplates: make([]resourceTemplateRegisterFunc, 0),
	}

	// Apply options to check configuration
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Validate that only prompts are configured
	if len(cfg.tools) > 0 || len(cfg.resources) > 0 || len(cfg.resourceTemplates) > 0 {
		return nil, fmt.Errorf("NewPromptHandler only supports prompts; use NewHandler for mixed resource types")
	}

	// Delegate to the unified constructor
	return NewHandler(opts...)
}

// createPromptHandler wraps a user function to match MCP PromptHandler signature
func createPromptHandler(fn PromptFunc) mcp.PromptHandler {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Convert MCP request to user-friendly args
		args := make(map[string]any)
		if req.Params.Arguments != nil {
			// Convert map[string]string to map[string]any
			for k, v := range req.Params.Arguments {
				args[k] = v
			}
		}

		// Execute user function
		result, err := fn(ctx, args)
		if err != nil {
			// Return all errors as protocol-level errors
			// Prompts and resources follow the same pattern: errors are protocol-level,
			// unlike tools where the SDK wraps errors for LLM visibility
			return nil, err
		}

		// Convert result to MCP format
		messages := make([]*mcp.PromptMessage, len(result.Messages))
		for i, msg := range result.Messages {
			messages[i] = &mcp.PromptMessage{
				Role:    mcp.Role(msg.Role),
				Content: &mcp.TextContent{Text: msg.Content},
			}
		}

		return &mcp.GetPromptResult{
			Messages:    messages,
			Description: result.Description,
		}, nil
	}
}
