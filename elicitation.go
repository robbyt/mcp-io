package mcpio

import (
	"context"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ElicitationCapability provides access to elicitation functionality.
// This interface allows handlers to request additional information from users
// through the MCP client during runtime execution.
type ElicitationCapability interface {
	// Elicit sends an elicitation request to the client asking for user input.
	// The message parameter is presented to the user, and requestedSchema defines
	// the structure of the expected response using a restricted subset of JSON Schema.
	//
	// Returns an ElicitResult containing the user's action ("accept", "decline", or "cancel")
	// and optionally the submitted form data when action is "accept".
	Elicit(ctx context.Context, message string, requestedSchema *jsonschema.Schema) (*mcp.ElicitResult, error)
}

// sessionElicitationCapability implements ElicitationCapability using a ServerSession
type sessionElicitationCapability struct {
	session *mcp.ServerSession
}

// Elicit implements ElicitationCapability interface
func (s *sessionElicitationCapability) Elicit(ctx context.Context, message string, requestedSchema *jsonschema.Schema) (*mcp.ElicitResult, error) {
	params := &mcp.ElicitParams{
		Message:         message,
		RequestedSchema: requestedSchema,
	}
	return s.session.Elicit(ctx, params)
}

// GetElicitationCapability extracts the elicitation capability from a ServerSession.
// This is the primary way to access elicitation functionality from within handlers.
// The session is automatically available in session-aware tool/prompt/resource handlers.
//
// Example in a session-aware tool:
//
//	func setupTool(ctx context.Context, capability ElicitationCapability, input struct{}) (map[string]any, error) {
//	    // The capability parameter provides access to elicitation
//	    result, err := ElicitTyped[Config](ctx, capability, "Enter setup configuration:")
//	    if err != nil {
//	        return nil, err
//	    }
//	    // Process result...
//	}
//
// Example in a regular tool (manual extraction):
//
//	func regularTool(ctx context.Context, req *mcp.CallToolRequest, input ToolInput) (*mcp.CallToolResult, ToolOutput, error) {
//	    capability := GetElicitationCapability(req.Session)
//	    result, err := ElicitSimple(ctx, capability, "Confirm action?", "confirm", "Type 'yes' to proceed")
//	    // Handle result...
//	}
//
// Note: Session-aware handlers (WithSessionTool, WithSessionPrompt, WithSessionResource)
// automatically provide the capability parameter, making manual extraction unnecessary.
func GetElicitationCapability(session *mcp.ServerSession) ElicitationCapability {
	return &sessionElicitationCapability{session: session}
}

// ElicitTyped sends an elicitation request with automatic schema generation from a Go struct.
// This is a convenience function that automatically generates the JSON schema from the provided
// type parameter T, similar to how typed tools work. The schema is generated from struct tags,
// allowing you to specify validation rules, descriptions, and constraints.
//
// Basic Example:
//
//	type UserConfig struct {
//	    Name     string `json:"name" jsonschema:"description=Your full name"`
//	    Email    string `json:"email" jsonschema:"format=email,description=Your email address"`
//	    EnableUI bool   `json:"enableUI" jsonschema:"description=Enable graphical interface"`
//	}
//
//	result, err := mcpio.ElicitTyped[UserConfig](ctx, capability, "Please provide your configuration:")
//	if err != nil {
//	    return nil, fmt.Errorf("elicitation failed: %w", err)
//	}
//
//	// Handle all possible user responses
//	switch result.Action {
//	case "accept":
//	    // Parse the user's input
//	    var config UserConfig
//	    data, _ := json.Marshal(result.Content)
//	    json.Unmarshal(data, &config)
//	    // Use config...
//	case "decline":
//	    return map[string]any{"status": "declined"}, nil
//	case "cancel":
//	    return map[string]any{"status": "cancelled"}, nil
//	}
//
// For multi-step workflows with conditional elicitation, see examples/http_multistep/
//
// Schema Tags:
// Use jsonschema struct tags for validation:
//   - description: Human-readable field description
//   - minimum/maximum: Numeric constraints
//   - enum: Allowed values (enum=option1,enum=option2)
//   - format: String format (email, uri, etc.)
func ElicitTyped[T any](ctx context.Context, capability ElicitationCapability, message string) (*mcp.ElicitResult, error) {
	// Generate schema from type T
	schema, err := GenerateSchema[T]()
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema for elicitation: %w", err)
	}
	return capability.Elicit(ctx, message, schema)
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

// WrapElicitResult wraps an MCP ElicitResult with convenience methods.
func WrapElicitResult(result *mcp.ElicitResult) *ElicitationResult {
	return &ElicitationResult{ElicitResult: result}
}

// ElicitTypedResult sends an elicitation request and wraps the result with convenience methods.
func ElicitTypedResult[T any](ctx context.Context, capability ElicitationCapability, message string) (*ElicitationResult, error) {
	result, err := ElicitTyped[T](ctx, capability, message)
	if err != nil {
		return nil, err
	}
	return WrapElicitResult(result), nil
}

// ElicitSimple is a convenience function for simple string input elicitation.
// This creates a schema for a single string field with the specified name and description.
// It's ideal for quick confirmations, simple text inputs, or collecting single values.
//
// Basic Example:
//
//	result, err := ElicitSimple(ctx, capability,
//	    "Enter your username:", "username", "Your account username")
//	if err != nil {
//	    return nil, err
//	}
//
//	if result.IsAccepted() {
//	    username := result.GetContent()["username"].(string)
//	    // Use username...
//	}
//
// Confirmation Example:
//
//	result, err := ElicitSimple(ctx, capability,
//	    "Delete all files? This cannot be undone.", "confirm", "Type 'DELETE' to confirm")
//	if err != nil {
//	    return nil, err
//	}
//
//	if result.IsAccepted() {
//	    if confirmation := result.GetContent()["confirm"].(string); confirmation == "DELETE" {
//	        // Proceed with deletion
//	    } else {
//	        return map[string]any{"status": "cancelled", "reason": "confirmation mismatch"}, nil
//	    }
//	} else {
//	    return map[string]any{"status": "cancelled", "reason": result.Action}, nil
//	}
//
// The fieldName parameter becomes the key in the returned Content map.
// The description parameter guides the user on what to enter.
func ElicitSimple(ctx context.Context, capability ElicitationCapability, message, fieldName, description string) (*ElicitationResult, error) {
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

	result, err := capability.Elicit(ctx, message, schema)
	if err != nil {
		return nil, err
	}
	return WrapElicitResult(result), nil
}
