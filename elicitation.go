package mcpio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Elicitation action constants defined by the MCP specification
const (
	ElicitActionAccept  = "accept"
	ElicitActionDecline = "decline"
	ElicitActionCancel  = "cancel"
)

// Elicitor provides elicitation functionality for interactive user input.
// Create via mcpio.NewElicitor(toolCtx) in tool functions.
type Elicitor struct {
	toolCtx RequestContext
}

// NewElicitor creates an Elicitor with the given RequestContext.
// This allows elicitation to access the session and validate elicitation support.
func NewElicitor(toolCtx RequestContext) *Elicitor {
	return &Elicitor{toolCtx: toolCtx}
}

// validateElicitation checks if the RequestContext has a valid session with elicitation support.
func (e *Elicitor) validateElicitation(ctx context.Context) error {
	session := e.toolCtx.GetSession()
	if session == nil {
		return ErrNoSession
	}
	if !session.SupportsElicitation() {
		return ErrElicitationNotSupported
	}
	return nil
}

// ElicitTyped sends an elicitation request with automatic schema generation from a Go struct.
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
//	func myTool(ctx context.Context, toolCtx mcpio.RequestContext, input MyInput) (MyOutput, error) {
//	    elicitor := mcpio.NewElicitor(toolCtx)
//	    result, err := mcpio.ElicitTyped[UserConfig](ctx, elicitor, "Please provide your configuration:")
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
func ElicitTyped[T any](ctx context.Context, elicitor *Elicitor, message string) (*ElicitationResult, error) {
	if elicitor == nil {
		return nil, errors.New("elicitor cannot be nil")
	}
	if err := elicitor.validateElicitation(ctx); err != nil {
		return nil, err
	}

	// Generate schema from type T
	s, err := jsonschema.For[T](nil)
	if err != nil {
		return nil, errors.Join(ErrSchemaGeneration, err)
	}

	session := elicitor.toolCtx.GetSession()
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
	return strings.ToLower(r.Action) == ElicitActionAccept
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
	return strings.ToLower(r.Action) == ElicitActionDecline
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
	return strings.ToLower(r.Action) == ElicitActionCancel
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
func (e *Elicitor) ElicitSimple(ctx context.Context, message, fieldName, description string) (*ElicitationResult, error) {
	if err := e.validateElicitation(ctx); err != nil {
		return nil, err
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

	session := e.toolCtx.GetSession()
	result, err := session.Elicit(ctx, message, schema)
	if err != nil {
		return nil, err
	}
	return WrapElicitResult(result), nil
}
