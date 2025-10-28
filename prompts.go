package mcpio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PromptFunc is the function signature for user-defined prompt handlers.
// The function receives a context and a map of arguments, and returns a PromptResult with an optional error.
type PromptFunc func(context.Context, map[string]any) (*PromptResult, error)

// TypedPromptFunc is the function signature for typed prompts with automatic schema generation.
// The function receives a context and typed arguments, and returns a PromptResult with an optional error.
// Schema generation is handled automatically based on the TArgs type.
//
// Example:
//
//	type DocumentArgs struct {
//		DocumentType string `json:"document_type" jsonschema:"description:Type of document"`
//		Topic        string `json:"topic" jsonschema:"description:Main topic"`
//	}
//	func myPrompt(ctx context.Context, args DocumentArgs) (*PromptResult, error) { ... }
type TypedPromptFunc[TArgs any] func(context.Context, TArgs) (*PromptResult, error)

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

// createPromptHandler wraps a user function to match MCP PromptHandler signature
func createPromptHandler(fn PromptFunc) mcp.PromptHandler {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Inject session into context so the user function can access it
		ctx = withMCPSession(ctx, req.Session)

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

// createTypedPromptHandler wraps a typed prompt function to match MCP PromptHandler signature
func createTypedPromptHandler[TArgs any](fn TypedPromptFunc[TArgs]) mcp.PromptHandler {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Inject session into context so the user function can access it
		ctx = withMCPSession(ctx, req.Session)

		// Convert MCP request arguments to typed struct
		var args TArgs
		if req.Params.Arguments != nil {
			// Marshal the arguments map to JSON, then unmarshal into typed struct
			argBytes, err := json.Marshal(req.Params.Arguments)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal prompt arguments: %w", err)
			}

			if err := json.Unmarshal(argBytes, &args); err != nil {
				return nil, fmt.Errorf("into %T: %w", args, errors.Join(ErrUnmarshalContent, err))
			}
		}

		// Execute user function with typed arguments
		result, err := fn(ctx, args)
		if err != nil {
			// Return all errors as protocol-level errors
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

// convertToJSONSchema converts any schema type to *jsonschema.Schema for
// internal processing that requires accessing specific schema fields like
// Properties and Required.
func convertToJSONSchema(schema any) (*jsonschema.Schema, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema cannot be nil")
	}

	// If it's already a *jsonschema.Schema, use it directly
	if s, ok := schema.(*jsonschema.Schema); ok {
		return s, nil
	}

	// Convert other types by marshaling to JSON and unmarshaling to *jsonschema.Schema
	jsonBytes, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema to JSON: %w", err)
	}

	var result *jsonschema.Schema
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema from JSON: %w", err)
	}

	return result, nil
}

// schemaToPromptArguments converts a JSON schema to MCP prompt arguments
func schemaToPromptArguments(s any) ([]*mcp.PromptArgument, error) {
	if s == nil {
		return nil, ErrNilValue
	}

	// Convert to *jsonschema.Schema to access Properties and Required fields
	jsonSchemaObj, err := convertToJSONSchema(s)
	if err != nil {
		return nil, err
	}

	if jsonSchemaObj.Properties == nil {
		return nil, ErrNoSchemaProperties
	}

	var args []*mcp.PromptArgument
	requiredMap := make(map[string]bool)

	// Create a map of required fields
	for _, field := range jsonSchemaObj.Required {
		requiredMap[field] = true
	}

	// Extract and sort property keys for deterministic ordering
	keys := slices.Sorted(maps.Keys(jsonSchemaObj.Properties))

	// Convert schema properties to prompt arguments in sorted order
	for _, name := range keys {
		propSchema := jsonSchemaObj.Properties[name]
		arg := &mcp.PromptArgument{
			Name:        name,
			Description: propSchema.Description,
			Required:    requiredMap[name],
		}
		args = append(args, arg)
	}

	return args, nil
}
