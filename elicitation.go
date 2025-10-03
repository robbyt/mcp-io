package mcpio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/schema"
)

// ElicitationCapability provides access to elicitation functionality.
// This interface is automatically satisfied by SessionCapability.
// Use GetSession(ctx) to access elicitation in your handlers.
type ElicitationCapability interface {
	// Elicit sends an elicitation request to the client asking for user input.
	// The message parameter is presented to the user, and requestedSchema defines
	// the structure of the expected response using a restricted subset of JSON Schema.
	// The requestedSchema can be any value that JSON-marshals to valid JSON schema,
	// including *jsonschema.Schema, json.RawMessage, or map[string]any.
	//
	// Returns an ElicitResult containing the user's action ("accept", "decline", or "cancel")
	// and optionally the submitted form data when action is "accept".
	Elicit(ctx context.Context, message string, requestedSchema any) (*mcp.ElicitResult, error)
}

// ElicitTyped sends an elicitation request with automatic schema generation from a Go struct.
// The session is automatically extracted from the context.
// The schema is generated from struct tags, allowing you to specify validation rules,
// descriptions, and constraints.
//
// Example:
//
//	type UserConfig struct {
//	    Name     string `json:"name" jsonschema:"Your full name"`
//	    Email    string `json:"email" jsonschema:"format=email"`
//	    EnableUI bool   `json:"enableUI" jsonschema:"Enable graphical interface"`
//	}
//
//	func myTool(ctx context.Context, input MyInput) (MyOutput, error) {
//	    result, err := mcpio.ElicitTyped[UserConfig](ctx, "Please provide your configuration:")
//	    if err != nil {
//	        return MyOutput{}, err
//	    }
//
//	    if result.IsAccepted() {
//	        var config UserConfig
//	        if err := result.DecodeContent(&config); err != nil {
//	            return MyOutput{}, err
//	        }
//	        // Use config...
//	    }
//	    return MyOutput{}, nil
//	}
//
// Schema Tags:
// Use jsonschema struct tags for validation:
//   - description: Human-readable field description
//   - minimum/maximum: Numeric constraints
//   - enum: Allowed values (enum=option1,enum=option2)
//   - format: String format (email, uri, etc.)
func ElicitTyped[T any](ctx context.Context, message string) (*ElicitationResult, error) {
	session := GetSession(ctx)
	if session == nil {
		return nil, ErrNoSession
	}
	if !session.SupportsElicitation() {
		return nil, ErrElicitationNotSupported
	}

	// Generate schema from type T
	s, err := schema.New[T]()
	if err != nil {
		return nil, errors.Join(ErrSchemaGeneration, err)
	}

	result, err := session.Elicit(ctx, message, s)
	if err != nil {
		return nil, err
	}
	return WrapElicitResult(result), nil
}

// ElicitationResult provides typed access to elicitation results.
// This wraps the raw MCP ElicitResult with convenience methods.
type ElicitationResult struct {
	*mcp.ElicitResult
}

// IsAccepted returns true if the user accepted the elicitation.
//
// Example:
//
//	result, err := ElicitTyped[Config](ctx, capability, "Enter configuration:")
//	if err != nil {
//	    return nil, err
//	}
//
//	if result.IsAccepted() {
//	    // User provided data - process result.Content
//	    data := result.GetContent()
//	    // Convert to your struct...
//	}
func (r *ElicitationResult) IsAccepted() bool {
	return r.Action == "accept"
}

// IsDeclined returns true if the user explicitly declined the elicitation.
//
// Example:
//
//	result, err := ElicitSimple(ctx, capability, "Continue?", "confirm", "Type yes/no")
//	if err != nil {
//	    return nil, err
//	}
//
//	if result.IsDeclined() {
//	    return map[string]any{"status": "user_declined"}, nil
//	}
func (r *ElicitationResult) IsDeclined() bool {
	return r.Action == "decline"
}

// IsCancelled returns true if the user cancelled/dismissed the elicitation.
//
// Example:
//
//	result, err := ElicitTyped[Config](ctx, capability, "Enter config:")
//	if err != nil {
//	    return nil, err
//	}
//
//	if result.IsCancelled() {
//	    return map[string]any{"status": "cancelled"}, nil
//	}
func (r *ElicitationResult) IsCancelled() bool {
	return r.Action == "cancel"
}

// GetContent returns the submitted form data when the action is "accept".
// Returns nil if the user declined or cancelled.
//
// Example:
//
//	result, err := ElicitSimple(ctx, capability, "Name:", "name", "Your name")
//	if err != nil {
//	    return nil, err
//	}
//
//	if content := result.GetContent(); content != nil {
//	    name := content["name"].(string)
//	    // Use name...
//	} else {
//	    // User declined or cancelled
//	}
func (r *ElicitationResult) GetContent() map[string]any {
	if r.IsAccepted() {
		return r.Content
	}
	return nil
}

// DecodeContent decodes the elicitation content into the target struct.
// This eliminates the need for manual JSON marshaling/unmarshaling when
// working with typed elicitation results.
//
// Example usage:
//
//	type UserConfig struct {
//	    Name  string `json:"name"`
//	    Email string `json:"email"`
//	}
//
//	result, err := ElicitTypedResult[UserConfig](ctx, capability, "Enter config:")
//	if err != nil {
//	    return err
//	}
//
//	if result.IsAccepted() {
//	    var config UserConfig
//	    if err := result.DecodeContent(&config); err != nil {
//	        return fmt.Errorf("failed to decode config: %w", err)
//	    }
//	    // Use config...
//	}
//
// The method returns an error if:
//   - The elicitation was not accepted (content is nil)
//   - JSON marshaling/unmarshaling fails
//   - The target is not a valid pointer
func (r *ElicitationResult) DecodeContent(target any) error {
	content := r.GetContent()
	if content == nil {
		return ErrNoContent
	}

	data, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal elicitation content: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return errors.Join(ErrUnmarshalContent, err)
	}

	return nil
}

// WrapElicitResult wraps an MCP ElicitResult with convenience methods.
func WrapElicitResult(result *mcp.ElicitResult) *ElicitationResult {
	return &ElicitationResult{ElicitResult: result}
}

// ElicitSimple is a convenience function for simple string input elicitation.
// The session is automatically extracted from the context.
// This creates a schema for a single string field with the specified name and description.
// It's ideal for quick confirmations, simple text inputs, or collecting single values.
//
// Example:
//
//	func myTool(ctx context.Context, input MyInput) (MyOutput, error) {
//	    result, err := mcpio.ElicitSimple(ctx, "Enter your username:", "username", "Your account username")
//	    if err != nil {
//	        return MyOutput{}, err
//	    }
//
//	    if result.IsAccepted() {
//	        username := result.GetContent()["username"].(string)
//	        // Use username...
//	    }
//	    return MyOutput{}, nil
//	}
//
// Confirmation Example:
//
//	result, err := mcpio.ElicitSimple(ctx,
//	    "Delete all files? This cannot be undone.", "confirm", "Type 'DELETE' to confirm")
//	if err != nil {
//	    return MyOutput{}, err
//	}
//
//	if result.IsAccepted() {
//	    if confirmation := result.GetContent()["confirm"].(string); confirmation == "DELETE" {
//	        // Proceed with deletion
//	    }
//	}
//
// The fieldName parameter becomes the key in the returned Content map.
// The description parameter guides the user on what to enter.
func ElicitSimple(ctx context.Context, message, fieldName, description string) (*ElicitationResult, error) {
	session := GetSession(ctx)
	if session == nil {
		return nil, ErrNoSession
	}
	if !session.SupportsElicitation() {
		return nil, ErrElicitationNotSupported
	}

	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			fieldName: {
				Type:        "string",
				Description: description,
			},
		},
		Required: []string{fieldName},
	}

	result, err := session.Elicit(ctx, message, schema)
	if err != nil {
		return nil, err
	}
	return WrapElicitResult(result), nil
}
