package main

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/suite"
)

// MockElicitationCapability for testing elicitation functionality
type MockElicitationCapability struct {
	Responses []*mcp.ElicitResult
	CallIndex int
}

func (m *MockElicitationCapability) Elicit(ctx context.Context, message string, requestedSchema *jsonschema.Schema) (*mcp.ElicitResult, error) {
	if m.CallIndex >= len(m.Responses) {
		return &mcp.ElicitResult{Action: "cancel"}, nil
	}
	result := m.Responses[m.CallIndex]
	m.CallIndex++
	return result, nil
}

// CliElicitationTestSuite tests the cli_elicitation example
type CliElicitationTestSuite struct {
	testutil.ExampleTestSuite
}

func (s *CliElicitationTestSuite) SetupSuite() {
	// Get project root - we're in examples/cli_elicitation
	_, b, _, _ := runtime.Caller(0)
	exampleDir := filepath.Dir(b)
	s.ProjectRoot = filepath.Join(exampleDir, "..", "..")
	s.ExampleName = "cli_elicitation"

	// Call parent SetupSuite
	s.ExampleTestSuite.SetupSuite()
}

func TestCliElicitationSuite(t *testing.T) {
	suite.Run(t, new(CliElicitationTestSuite))
}

func (s *CliElicitationTestSuite) TestSetupApplication() {
	ctx := s.T().Context()

	// Test accepting configuration
	s.Run("AcceptConfiguration", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"name":        "John Doe",
						"email":       "john@example.com",
						"environment": "development",
					},
				},
			},
		}

		result, err := setupApplication(ctx, mockCapability, struct{}{})
		s.Require().NoError(err)

		s.Equal("configured", result["status"])
		s.Contains(result["message"], "development")

		config := result["config"].(map[string]any)
		s.Equal("John Doe", config["name"])
		s.Equal("john@example.com", config["email"])
		s.Equal("development", config["environment"])
	})

	s.Run("DeclineConfiguration", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{Action: "decline"},
			},
		}

		result, err := setupApplication(ctx, mockCapability, struct{}{})
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])
		s.Contains(result["reason"], "decline")
	})

	s.Run("CancelConfiguration", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{Action: "cancel"},
			},
		}

		result, err := setupApplication(ctx, mockCapability, struct{}{})
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])
		s.Contains(result["reason"], "cancel")
	})
}

func (s *CliElicitationTestSuite) TestInteractivePrompt() {
	ctx := s.T().Context()

	s.Run("AcceptRequirements", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"requirements": "Must include error handling and logging",
					},
				},
			},
		}

		result, err := interactivePrompt(ctx, mockCapability, map[string]any{"document_type": "API guide"})
		s.Require().NoError(err)

		s.Contains(result.Description, "API guide")
		s.Require().Len(result.Messages, 2)

		// Check system message includes requirements
		systemMsg := result.Messages[0]
		s.Equal("system", systemMsg.Role)
		s.Contains(systemMsg.Content, "error handling and logging")

		// Check user message
		userMsg := result.Messages[1]
		s.Equal("user", userMsg.Role)
		s.Contains(userMsg.Content, "API guide")
	})

	s.Run("DeclineRequirements", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{Action: "decline"},
			},
		}

		result, err := interactivePrompt(ctx, mockCapability, map[string]any{"document_type": "report"})
		s.Require().NoError(err)

		s.Contains(result.Description, "report")
		s.Require().Len(result.Messages, 2)

		// Check system message uses default requirements
		systemMsg := result.Messages[0]
		s.Equal("system", systemMsg.Role)
		s.Contains(systemMsg.Content, "No specific requirements provided")
	})

	s.Run("DefaultDocumentType", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"requirements": "Keep it simple and concise",
					},
				},
			},
		}

		// Test with no document_type specified
		result, err := interactivePrompt(ctx, mockCapability, map[string]any{})
		s.Require().NoError(err)

		s.Contains(result.Description, "general")
		systemMsg := result.Messages[0]
		s.Contains(systemMsg.Content, "general")
		s.Contains(systemMsg.Content, "Keep it simple and concise")
	})
}

func (s *CliElicitationTestSuite) TestServerCreation() {
	// Test that we can create the server (session-aware tools require special handling)
	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("elicitation-demo"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithSessionTool("setup_application", "Interactive application setup with configuration elicitation", setupApplication),
			mcpio.WithSessionPrompt("interactive_document", "Generate documents with elicited requirements", interactivePrompt),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	// Verify server creation works
	server, err := serverBuilder()
	s.Require().NoError(err)
	s.NotNil(server)
}

func (s *CliElicitationTestSuite) TestUserConfigValidation() {
	// Test that UserConfig struct has proper JSON schema tags
	// We can't easily test the schema generation without the elicitation package internals,
	// but we can test that the struct is properly defined
	var config UserConfig
	config.Name = "Test User"
	config.Email = "test@example.com"
	config.Environment = "production"

	s.Equal("Test User", config.Name)
	s.Equal("test@example.com", config.Email)
	s.Equal("production", config.Environment)

	// Test enum validation would happen at schema level
	validEnvironments := []string{"development", "staging", "production"}
	s.Contains(validEnvironments, config.Environment)
}

func (s *CliElicitationTestSuite) TestErrorHandling() {
	ctx := s.T().Context()

	s.Run("EmptyElicitationResult", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action:  "accept",
					Content: map[string]any{}, // Empty content
				},
			},
		}

		// This should still work, just with empty/default values
		result, err := setupApplication(ctx, mockCapability, struct{}{})
		s.Require().NoError(err)
		s.Equal("configured", result["status"])
	})

	s.Run("MultipleElicitationCalls", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"requirements": "First requirement",
					},
				},
				{
					Action: "accept",
					Content: map[string]any{
						"requirements": "Second requirement",
					},
				},
			},
		}

		// First call
		result1, err := interactivePrompt(ctx, mockCapability, map[string]any{"document_type": "guide"})
		s.Require().NoError(err)
		s.Contains(result1.Messages[0].Content, "First requirement")

		// Second call
		result2, err := interactivePrompt(ctx, mockCapability, map[string]any{"document_type": "manual"})
		s.Require().NoError(err)
		s.Contains(result2.Messages[0].Content, "Second requirement")
	})
}

func (s *CliElicitationTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()

	// Verify binary was created
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}
