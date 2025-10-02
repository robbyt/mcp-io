package mcpio

import (
	"context"
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockElicitationCapability is a mock implementation for testing
type MockElicitationCapability struct {
	mock.Mock
}

func (m *MockElicitationCapability) Elicit(ctx context.Context, message string, requestedSchema any) (*mcp.ElicitResult, error) {
	args := m.Called(ctx, message, requestedSchema)
	return args.Get(0).(*mcp.ElicitResult), args.Error(1)
}

// Test types for elicitation
type TestConfig struct {
	Name  string `json:"name"  jsonschema:"Test name"`
	Value int    `json:"value" jsonschema:"Test value"`
}

func TestGetElicitationCapability(t *testing.T) {
	session := &mcp.ServerSession{}
	capability := GetElicitationCapability(session)

	assert.NotNil(t, capability)
	assert.IsType(t, &sessionElicitationCapability{}, capability)
}

func TestElicitTyped_Success(t *testing.T) {
	mockCapability := &MockElicitationCapability{}

	expectedResult := &mcp.ElicitResult{
		Action: "accept",
		Content: map[string]any{
			"name":  "test",
			"value": 42,
		},
	}

	// Setup mock expectation
	mockCapability.On("Elicit", mock.Anything, "Test message", mock.Anything).Return(expectedResult, nil)

	// Call ElicitTyped
	result, err := ElicitTyped[TestConfig](context.Background(), mockCapability, "Test message")

	require.NoError(t, err)
	assert.Equal(t, "accept", result.Action)
	assert.Equal(t, "test", result.Content["name"])
	assert.Equal(t, 42, result.Content["value"])

	mockCapability.AssertExpectations(t)
}

func TestElicitTyped_SchemaGeneration(t *testing.T) {
	mockCapability := &MockElicitationCapability{}

	expectedResult := &mcp.ElicitResult{Action: "decline"}

	// Capture the schema that gets passed to Elicit
	var capturedSchema any
	mockCapability.On("Elicit", mock.Anything, "Test message", mock.Anything).
		Run(func(args mock.Arguments) {
			capturedSchema = args.Get(2)
		}).Return(expectedResult, nil)

	// Call ElicitTyped
	_, err := ElicitTyped[TestConfig](context.Background(), mockCapability, "Test message")

	require.NoError(t, err)
	assert.NotNil(t, capturedSchema)

	// Convert captured schema back to jsonschema.Schema for testing
	jsonSchemaObj, err := schema.ConvertToJSONSchema(capturedSchema)
	require.NoError(t, err)
	assert.Equal(t, "object", jsonSchemaObj.Type)

	// Check that required fields are present
	assert.Contains(t, jsonSchemaObj.Properties, "name")
	assert.Contains(t, jsonSchemaObj.Properties, "value")

	// Check field types
	nameField := jsonSchemaObj.Properties["name"]
	assert.Equal(t, "string", nameField.Type)
	assert.Equal(t, "Test name", nameField.Description)

	valueField := jsonSchemaObj.Properties["value"]
	assert.Equal(t, "integer", valueField.Type)
	assert.Equal(t, "Test value", valueField.Description)

	mockCapability.AssertExpectations(t)
}

func TestElicitationResult_IsAccepted(t *testing.T) {
	result := &ElicitationResult{
		ElicitResult: &mcp.ElicitResult{Action: "accept"},
	}
	assert.True(t, result.IsAccepted())
	assert.False(t, result.IsDeclined())
	assert.False(t, result.IsCancelled())
}

func TestElicitationResult_IsDeclined(t *testing.T) {
	result := &ElicitationResult{
		ElicitResult: &mcp.ElicitResult{Action: "decline"},
	}
	assert.False(t, result.IsAccepted())
	assert.True(t, result.IsDeclined())
	assert.False(t, result.IsCancelled())
}

func TestElicitationResult_IsCancelled(t *testing.T) {
	result := &ElicitationResult{
		ElicitResult: &mcp.ElicitResult{Action: "cancel"},
	}
	assert.False(t, result.IsAccepted())
	assert.False(t, result.IsDeclined())
	assert.True(t, result.IsCancelled())
}

func TestElicitationResult_GetContent_Accept(t *testing.T) {
	content := map[string]any{"key": "value"}
	result := &ElicitationResult{
		ElicitResult: &mcp.ElicitResult{
			Action:  "accept",
			Content: content,
		},
	}

	assert.Equal(t, content, result.GetContent())
}

func TestElicitationResult_GetContent_Decline(t *testing.T) {
	result := &ElicitationResult{
		ElicitResult: &mcp.ElicitResult{Action: "decline"},
	}

	assert.Nil(t, result.GetContent())
}

func TestWrapElicitResult(t *testing.T) {
	original := &mcp.ElicitResult{Action: "accept"}
	wrapped := WrapElicitResult(original)

	assert.Equal(t, original, wrapped.ElicitResult)
	assert.True(t, wrapped.IsAccepted())
}

func TestElicitTypedResult(t *testing.T) {
	mockCapability := &MockElicitationCapability{}

	expectedResult := &mcp.ElicitResult{
		Action:  "accept",
		Content: map[string]any{"name": "test"},
	}

	mockCapability.On("Elicit", mock.Anything, "Test message", mock.Anything).Return(expectedResult, nil)

	result, err := ElicitTypedResult[TestConfig](context.Background(), mockCapability, "Test message")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsAccepted())
	assert.Equal(t, "test", result.GetContent()["name"])

	mockCapability.AssertExpectations(t)
}

func TestElicitSimple_Success(t *testing.T) {
	mockCapability := &MockElicitationCapability{}

	expectedResult := &mcp.ElicitResult{
		Action:  "accept",
		Content: map[string]any{"username": "testuser"},
	}

	// Capture the schema to verify it's constructed correctly
	var capturedSchema any
	mockCapability.On("Elicit", mock.Anything, "Enter username", mock.Anything).
		Run(func(args mock.Arguments) {
			capturedSchema = args.Get(2)
		}).Return(expectedResult, nil)

	result, err := ElicitSimple(context.Background(), mockCapability, "Enter username", "username", "Your username")

	require.NoError(t, err)
	assert.True(t, result.IsAccepted())
	assert.Equal(t, "testuser", result.GetContent()["username"])

	// Convert captured schema back to jsonschema.Schema for testing
	jsonSchemaObj, err := schema.ConvertToJSONSchema(capturedSchema)
	require.NoError(t, err)

	// Verify schema structure
	assert.Equal(t, "object", jsonSchemaObj.Type)
	assert.Contains(t, jsonSchemaObj.Properties, "username")
	assert.Equal(t, []string{"username"}, jsonSchemaObj.Required)

	usernameField := jsonSchemaObj.Properties["username"]
	assert.Equal(t, "string", usernameField.Type)
	assert.Equal(t, "Your username", usernameField.Description)

	mockCapability.AssertExpectations(t)
}

func TestElicitSimple_Decline(t *testing.T) {
	mockCapability := &MockElicitationCapability{}

	expectedResult := &mcp.ElicitResult{Action: "decline"}
	mockCapability.On("Elicit", mock.Anything, "Enter username", mock.Anything).Return(expectedResult, nil)

	result, err := ElicitSimple(context.Background(), mockCapability, "Enter username", "username", "Your username")

	require.NoError(t, err)
	assert.True(t, result.IsDeclined())
	assert.Nil(t, result.GetContent())

	mockCapability.AssertExpectations(t)
}

func TestSessionElicitationCapability_Elicit(t *testing.T) {
	// Test the sessionElicitationCapability struct directly
	mockSession := &mcp.ServerSession{}

	// Create the capability wrapper
	sessionCapability := &sessionElicitationCapability{session: mockSession}

	// Verify the structure is created correctly
	assert.NotNil(t, sessionCapability)
	assert.Equal(t, mockSession, sessionCapability.session)

	// Note: We can't easily test the actual Elicit call without setting up
	// the full MCP server infrastructure, but we can test the wrapper creation
	// and type assertion
	capability := GetElicitationCapability(mockSession)
	assert.NotNil(t, capability)

	// Verify it's the right type
	_, ok := capability.(*sessionElicitationCapability)
	assert.True(t, ok, "Expected sessionElicitationCapability type")
}

func TestElicitTyped_SchemaGenerationError(t *testing.T) {
	mockCapability := &MockElicitationCapability{}

	// Use a type that would cause schema generation to fail
	type BadType struct {
		// Invalid JSON schema type - func types cannot be serialized to JSON schema
		BadField func() `json:"badField"`
	}

	result, err := ElicitTyped[BadType](context.Background(), mockCapability, "test message")

	require.ErrorIs(t, err, ErrSchemaGeneration)
	assert.Nil(t, result)

	mockCapability.AssertNotCalled(t, "Elicit")
}

func TestElicitTypedResult_ElicitError(t *testing.T) {
	mockCapability := &MockElicitationCapability{}
	expectedError := fmt.Errorf("elicit failed")

	mockCapability.On("Elicit", mock.Anything, "test message", mock.Anything).Return((*mcp.ElicitResult)(nil), expectedError)

	result, err := ElicitTypedResult[TestConfig](context.Background(), mockCapability, "test message")

	require.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, result)

	mockCapability.AssertExpectations(t)
}

func TestElicitSimple_ElicitError(t *testing.T) {
	mockCapability := &MockElicitationCapability{}
	expectedError := fmt.Errorf("elicit failed")

	mockCapability.On("Elicit", mock.Anything, "test message", mock.Anything).Return((*mcp.ElicitResult)(nil), expectedError)

	result, err := ElicitSimple(context.Background(), mockCapability, "test message", "fieldName", "description")

	require.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, result)

	mockCapability.AssertExpectations(t)
}

func TestElicitationResult_GetContent_Nil(t *testing.T) {
	// Test GetContent when Content is nil
	result := &ElicitationResult{
		ElicitResult: &mcp.ElicitResult{
			Action:  "accept",
			Content: nil,
		},
	}

	content := result.GetContent()
	assert.Nil(t, content)
}

func TestElicitationResult_AllActions(t *testing.T) {
	// Test all action types with the convenience methods
	tests := []struct {
		action      string
		isAccepted  bool
		isDeclined  bool
		isCancelled bool
		hasContent  bool
	}{
		{"accept", true, false, false, true},
		{"decline", false, true, false, false},
		{"cancel", false, false, true, false},
		{"unknown", false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			content := map[string]any{"key": "value"}
			if !tt.hasContent {
				content = nil
			}

			result := &ElicitationResult{
				ElicitResult: &mcp.ElicitResult{
					Action:  tt.action,
					Content: content,
				},
			}

			assert.Equal(t, tt.isAccepted, result.IsAccepted())
			assert.Equal(t, tt.isDeclined, result.IsDeclined())
			assert.Equal(t, tt.isCancelled, result.IsCancelled())

			if tt.hasContent {
				assert.Equal(t, content, result.GetContent())
			} else {
				assert.Nil(t, result.GetContent())
			}
		})
	}
}

func TestElicitTyped_Success_ComplexType(t *testing.T) {
	// Test with a more complex type structure
	type ComplexType struct {
		Settings struct {
			Debug   bool   `json:"debug"   jsonschema:"Enable debug mode"`
			LogFile string `json:"logFile" jsonschema:"Log file path"`
		} `json:"settings" jsonschema:"Application settings"`
		Metadata map[string]string `json:"metadata" jsonschema:"Additional metadata"`
	}

	mockCapability := &MockElicitationCapability{}

	expectedResult := &mcp.ElicitResult{
		Action: "accept",
		Content: map[string]any{
			"settings": map[string]any{
				"debug":   true,
				"logFile": "/var/log/app.log",
			},
			"metadata": map[string]any{
				"version": "1.0.0",
			},
		},
	}

	mockCapability.On("Elicit", mock.Anything, "Complex config", mock.Anything).Return(expectedResult, nil)

	result, err := ElicitTyped[ComplexType](context.Background(), mockCapability, "Complex config")

	require.NoError(t, err)
	assert.Equal(t, "accept", result.Action)
	assert.NotNil(t, result.Content)

	mockCapability.AssertExpectations(t)
}

// Test types for DecodeContent testing
type SimpleDecodeConfig struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type NestedDecodeConfig struct {
	Basic    SimpleDecodeConfig `json:"basic"`
	Settings map[string]any     `json:"settings"`
	Tags     []string           `json:"tags"`
	Enabled  bool               `json:"enabled"`
}

func TestElicitationResult_DecodeContent_Success(t *testing.T) {
	t.Parallel()

	// Create a mock result with content
	result := &ElicitationResult{
		ElicitResult: &mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"name":  "test-config",
				"value": 42,
			},
		},
	}

	var config SimpleDecodeConfig
	err := result.DecodeContent(&config)

	require.NoError(t, err)
	assert.Equal(t, "test-config", config.Name)
	assert.Equal(t, 42, config.Value)
}

func TestElicitationResult_DecodeContent_NestedStruct(t *testing.T) {
	t.Parallel()

	// Create a mock result with nested content
	result := &ElicitationResult{
		ElicitResult: &mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"basic": map[string]any{
					"name":  "nested-test",
					"value": 123,
				},
				"settings": map[string]any{
					"debug":   true,
					"timeout": 30,
				},
				"tags":    []string{"test", "config"},
				"enabled": true,
			},
		},
	}

	var config NestedDecodeConfig
	err := result.DecodeContent(&config)

	require.NoError(t, err)
	assert.Equal(t, "nested-test", config.Basic.Name)
	assert.Equal(t, 123, config.Basic.Value)
	assert.Equal(t, true, config.Settings["debug"])
	assert.InDelta(t, float64(30), config.Settings["timeout"], 0.001)
	assert.Equal(t, []string{"test", "config"}, config.Tags)
	assert.True(t, config.Enabled)
}

func TestElicitationResult_DecodeContent_NotAccepted(t *testing.T) {
	t.Parallel()

	// Create a mock result that was declined
	result := &ElicitationResult{
		ElicitResult: &mcp.ElicitResult{
			Action:  "decline",
			Content: nil,
		},
	}

	var config SimpleDecodeConfig
	err := result.DecodeContent(&config)

	require.ErrorIs(t, err, ErrNoContent)
}

func TestElicitationResult_DecodeContent_CancelledResult(t *testing.T) {
	t.Parallel()

	// Create a mock result that was cancelled
	result := &ElicitationResult{
		ElicitResult: &mcp.ElicitResult{
			Action:  "cancel",
			Content: nil,
		},
	}

	var config SimpleDecodeConfig
	err := result.DecodeContent(&config)

	require.ErrorIs(t, err, ErrNoContent)
}

func TestElicitationResult_DecodeContent_InvalidJSONUnmarshal(t *testing.T) {
	t.Parallel()

	// Create a mock result with content that can't be unmarshaled to the target type
	result := &ElicitationResult{
		ElicitResult: &mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"name":  "test",
				"value": "not-a-number", // This should fail to unmarshal to int
			},
		},
	}

	var config SimpleDecodeConfig
	err := result.DecodeContent(&config)

	require.ErrorIs(t, err, ErrUnmarshalContent)
}

func TestElicitationResult_DecodeContent_NilTarget(t *testing.T) {
	t.Parallel()

	result := &ElicitationResult{
		ElicitResult: &mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"name":  "test",
				"value": 42,
			},
		},
	}

	err := result.DecodeContent(nil)

	require.ErrorIs(t, err, ErrUnmarshalContent)
}

func TestElicitationResult_DecodeContent_NonPointerTarget(t *testing.T) {
	t.Parallel()

	result := &ElicitationResult{
		ElicitResult: &mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"name":  "test",
				"value": 42,
			},
		},
	}

	// Try to decode into a non-pointer (should fail)
	var config SimpleDecodeConfig
	err := result.DecodeContent(config) // Not &config

	require.ErrorIs(t, err, ErrUnmarshalContent)
}

func TestElicitationResult_DecodeContent_EmptyContent(t *testing.T) {
	t.Parallel()

	// Test with accepted result but empty content
	result := &ElicitationResult{
		ElicitResult: &mcp.ElicitResult{
			Action:  "accept",
			Content: map[string]any{},
		},
	}

	var config SimpleDecodeConfig
	err := result.DecodeContent(&config)

	// Should succeed but config should have zero values
	require.NoError(t, err)
	assert.Empty(t, config.Name)
	assert.Equal(t, 0, config.Value)
}
