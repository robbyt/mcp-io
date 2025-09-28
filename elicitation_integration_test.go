//go:build integration
// +build integration

package mcpio

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration test types
type UserPreferences struct {
	Theme         string `json:"theme"         jsonschema:"UI theme preference"`
	Language      string `json:"language"      jsonschema:"Preferred language"`
	Notifications bool   `json:"notifications" jsonschema:"Enable notifications"`
}

type APIConfiguration struct {
	Endpoint string `json:"endpoint" jsonschema:"API endpoint URL"`
	Timeout  int    `json:"timeout"  jsonschema:"Request timeout in seconds"`
	Retries  int    `json:"retries"  jsonschema:"Number of retry attempts"`
}

// MockElicitationCapabilityWithResponses for integration testing
type MockElicitationCapabilityWithResponses struct {
	Responses []*mcp.ElicitResult
	CallIndex int
}

func (m *MockElicitationCapabilityWithResponses) Elicit(ctx context.Context, message string, requestedSchema any) (*mcp.ElicitResult, error) {
	if m.CallIndex >= len(m.Responses) {
		return &mcp.ElicitResult{Action: "cancel"}, nil
	}
	result := m.Responses[m.CallIndex]
	m.CallIndex++
	return result, nil
}

func TestSessionAwareToolIntegration(t *testing.T) {
	// Create a mock capability with predefined responses
	mockCapability := &MockElicitationCapabilityWithResponses{
		Responses: []*mcp.ElicitResult{
			{
				Action: "accept",
				Content: map[string]any{
					"theme":         "dark",
					"language":      "en",
					"notifications": true,
				},
			},
		},
	}

	// Define a session-aware tool that elicits user preferences
	setupUserPreferences := func(ctx context.Context, capability ElicitationCapability, input struct{}) (map[string]any, error) {
		result, err := ElicitTypedResult[UserPreferences](ctx, capability, "Please configure your preferences:")
		if err != nil {
			return nil, err
		}

		if !result.IsAccepted() {
			return map[string]any{"status": "cancelled"}, nil
		}

		// Parse the preferences
		var prefs UserPreferences
		if err := result.DecodeContent(&prefs); err != nil {
			return nil, err
		}

		return map[string]any{
			"status": "configured",
			"preferences": map[string]any{
				"theme":         prefs.Theme,
				"language":      prefs.Language,
				"notifications": prefs.Notifications,
			},
		}, nil
	}

	// Create handler with session-aware tool
	handler, err := NewHandler(
		WithName("test-elicitation"),
		WithVersion("1.0.0"),
		WithSessionTool("setup_preferences", "Setup user preferences", setupUserPreferences),
	)
	require.NoError(t, err)

	// Test the handler creation (this validates the function signatures work correctly)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.GetServer())

	// Test direct elicitation call using mock capability
	result, err := ElicitTyped[UserPreferences](context.Background(), mockCapability, "Test message")
	require.NoError(t, err)
	assert.Equal(t, "accept", result.Action)
	assert.Equal(t, "dark", result.Content["theme"])
	assert.Equal(t, "en", result.Content["language"])
	assert.Equal(t, true, result.Content["notifications"])
}

func TestMultiStepElicitationIntegration(t *testing.T) {
	// Create a mock capability with multiple responses
	mockCapability := &MockElicitationCapabilityWithResponses{
		Responses: []*mcp.ElicitResult{
			{
				Action: "accept",
				Content: map[string]any{
					"endpoint": "https://api.example.com",
					"timeout":  30,
					"retries":  3,
				},
			},
			{
				Action: "accept",
				Content: map[string]any{
					"theme":         "light",
					"language":      "es",
					"notifications": false,
				},
			},
			{
				Action: "accept",
				Content: map[string]any{
					"confirm": "yes",
				},
			},
		},
	}

	// Multi-step elicitation workflow
	configureSystem := func(ctx context.Context, capability ElicitationCapability, input struct{}) (map[string]any, error) {
		// Step 1: API Configuration
		apiResult, err := ElicitTyped[APIConfiguration](ctx, capability, "Configure API settings:")
		if err != nil {
			return nil, err
		}

		if apiResult.Action != "accept" {
			return map[string]any{"status": "cancelled", "step": "api"}, nil
		}

		// Step 2: User Preferences
		prefResult, err := ElicitTyped[UserPreferences](ctx, capability, "Configure user preferences:")
		if err != nil {
			return nil, err
		}

		if prefResult.Action != "accept" {
			return map[string]any{"status": "cancelled", "step": "preferences"}, nil
		}

		// Step 3: Confirmation
		confirmResult, err := ElicitSimple(ctx, capability, "Confirm configuration?", "confirm", "Type 'yes' to confirm")
		if err != nil {
			return nil, err
		}

		confirmed := false
		if confirmResult.IsAccepted() {
			if content := confirmResult.GetContent(); content != nil {
				if val, ok := content["confirm"].(string); ok && val == "yes" {
					confirmed = true
				}
			}
		}

		return map[string]any{
			"status":    "completed",
			"confirmed": confirmed,
			"api":       apiResult.Content,
			"prefs":     prefResult.Content,
		}, nil
	}

	// Execute multi-step workflow
	result, err := configureSystem(context.Background(), mockCapability, struct{}{})
	require.NoError(t, err)

	assert.Equal(t, "completed", result["status"])
	assert.Equal(t, true, result["confirmed"])

	// Verify API configuration
	apiConfig := result["api"].(map[string]any)
	assert.Equal(t, "https://api.example.com", apiConfig["endpoint"])
	assert.Equal(t, 30, apiConfig["timeout"])
	assert.Equal(t, 3, apiConfig["retries"])

	// Verify user preferences
	prefs := result["prefs"].(map[string]any)
	assert.Equal(t, "light", prefs["theme"])
	assert.Equal(t, "es", prefs["language"])
	assert.Equal(t, false, prefs["notifications"])
}

func TestElicitationWithDeclineHandling(t *testing.T) {
	// Test scenario where user declines elicitation
	mockCapability := &MockElicitationCapabilityWithResponses{
		Responses: []*mcp.ElicitResult{
			{Action: "decline"},
			{Action: "cancel"},
			{
				Action: "accept",
				Content: map[string]any{
					"endpoint": "https://fallback.example.com",
					"timeout":  15,
					"retries":  1,
				},
			},
		},
	}

	// Tool that handles decline/cancel gracefully
	robustConfiguration := func(ctx context.Context, capability ElicitationCapability, input struct{}) (map[string]any, error) {
		// Try primary configuration
		result, err := ElicitTyped[APIConfiguration](ctx, capability, "Primary API configuration:")
		if err != nil {
			return nil, err
		}

		if result.Action == "decline" {
			// Try alternative simple configuration
			simpleResult, err := ElicitSimple(ctx, capability, "Minimal setup - just endpoint?", "endpoint", "API endpoint URL")
			if err != nil {
				return nil, err
			}

			if simpleResult.Action == "cancel" {
				// Use defaults
				fallbackResult, err := ElicitTyped[APIConfiguration](ctx, capability, "Use default configuration?")
				if err != nil {
					return nil, err
				}

				if fallbackResult.Action == "accept" {
					return map[string]any{
						"status": "fallback",
						"config": fallbackResult.Content,
					}, nil
				}

				return map[string]any{"status": "no_configuration"}, nil
			}

			return map[string]any{
				"status":   "simple",
				"endpoint": simpleResult.GetContent()["endpoint"],
			}, nil
		}

		return map[string]any{
			"status": "full",
			"config": result.Content,
		}, nil
	}

	// Execute with decline handling
	result, err := robustConfiguration(context.Background(), mockCapability, struct{}{})
	require.NoError(t, err)

	assert.Equal(t, "fallback", result["status"])

	config := result["config"].(map[string]any)
	assert.Equal(t, "https://fallback.example.com", config["endpoint"])
	assert.Equal(t, 15, config["timeout"])
	assert.Equal(t, 1, config["retries"])
}

func TestSessionAwarePromptIntegration(t *testing.T) {
	// Test session-aware prompt that elicits requirements
	mockCapability := &MockElicitationCapabilityWithResponses{
		Responses: []*mcp.ElicitResult{
			{
				Action: "accept",
				Content: map[string]any{
					"requirements": "Include error handling and logging",
				},
			},
		},
	}

	// Session-aware prompt function
	dynamicPromptGenerator := func(ctx context.Context, capability ElicitationCapability, args map[string]any) (*PromptResult, error) {
		documentType := "document"
		if dt, ok := args["type"].(string); ok {
			documentType = dt
		}

		// Elicit specific requirements
		result, err := ElicitSimple(ctx, capability,
			"What specific requirements do you have for this "+documentType+"?",
			"requirements", "Describe your requirements")
		if err != nil {
			return nil, err
		}

		requirements := "No specific requirements"
		if result.IsAccepted() {
			if content := result.GetContent(); content != nil {
				if req, ok := content["requirements"].(string); ok && req != "" {
					requirements = req
				}
			}
		}

		return &PromptResult{
			Description: "Custom " + documentType + " with elicited requirements",
			Messages: []PromptMessage{
				{Role: "system", Content: "You are creating a " + documentType + " with these requirements: " + requirements},
				{Role: "user", Content: "Please create the " + documentType},
			},
		}, nil
	}

	// Test the prompt function
	result, err := dynamicPromptGenerator(context.Background(), mockCapability, map[string]any{"type": "API guide"})
	require.NoError(t, err)

	assert.Equal(t, "Custom API guide with elicited requirements", result.Description)
	assert.Len(t, result.Messages, 2)
	assert.Equal(t, "system", result.Messages[0].Role)
	assert.Contains(t, result.Messages[0].Content, "error handling and logging")
	assert.Equal(t, "user", result.Messages[1].Role)
	assert.Contains(t, result.Messages[1].Content, "API guide")
}

func TestElicitationSchemaValidation(t *testing.T) {
	// Test that proper schemas are generated for complex types
	mockCapability := &MockElicitationCapabilityWithResponses{
		Responses: []*mcp.ElicitResult{
			{Action: "accept", Content: map[string]any{}},
		},
	}

	// Test schema generation for complex type
	type ComplexConfig struct {
		Database struct {
			Host     string `json:"host"     jsonschema:"Database host"`
			Port     int    `json:"port"     jsonschema:"Database port"`
			Username string `json:"username" jsonschema:"Database username"`
		} `json:"database" jsonschema:"Database configuration"`
		Features struct {
			EnableCache    bool     `json:"enableCache"    jsonschema:"Enable caching"`
			AllowedOrigins []string `json:"allowedOrigins" jsonschema:"Allowed CORS origins"`
		} `json:"features" jsonschema:"Feature flags"`
	}

	// Generate schema to verify it works with nested structures
	schema, err := schema.New[ComplexConfig]()
	require.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	assert.Contains(t, schema.Properties, "database")
	assert.Contains(t, schema.Properties, "features")

	// Test elicitation with complex type
	result, err := ElicitTyped[ComplexConfig](context.Background(), mockCapability, "Configure complex settings:")
	require.NoError(t, err)
	assert.Equal(t, "accept", result.Action)
}
