package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/suite"
)

// CliPromptTestSuite tests the cli_prompt example
type CliPromptTestSuite struct {
	testutil.ExampleTestSuite
}

func (s *CliPromptTestSuite) SetupSuite() {
	// Get project root - we're in examples/cli_prompt
	_, b, _, _ := runtime.Caller(0)
	exampleDir := filepath.Dir(b)
	s.ProjectRoot = filepath.Join(exampleDir, "..", "..")
	s.ExampleName = "cli_prompt"

	// Call parent SetupSuite
	s.ExampleTestSuite.SetupSuite()
}

func TestCliPromptSuite(t *testing.T) {
	suite.Run(t, new(CliPromptTestSuite))
}

func (s *CliPromptTestSuite) TestFunctionality() {
	ctx := s.T().Context()

	// Create server using the same logic as main()
	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("prompt-server"),
			mcpio.WithPrompt("greeter", "Generates a friendly greeting", greeterPrompt),
			mcpio.WithTypedPrompt("document_writer", "Generates writing prompts with typed arguments", documentPrompt),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		// Test greeter prompt (map-based)
		s.Run("GreeterWithName", func() {
			result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
				Name:      "greeter",
				Arguments: map[string]string{"name": "Alice"},
			})
			s.Require().NoError(err)
			s.Require().Len(result.Messages, 2)

			s.Equal("system", string(result.Messages[0].Role))
			s.Equal("You are a helpful assistant.", result.Messages[0].Content.(*mcp.TextContent).Text)
			s.Equal("user", string(result.Messages[1].Role))
			s.Equal("Create a friendly greeting for Alice", result.Messages[1].Content.(*mcp.TextContent).Text)
		})

		s.Run("GreeterWithoutName", func() {
			result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
				Name:      "greeter",
				Arguments: map[string]string{},
			})
			s.Require().NoError(err)
			s.Require().Len(result.Messages, 2)

			s.Equal("system", string(result.Messages[0].Role))
			s.Equal("user", string(result.Messages[1].Role))
			s.Equal("Create a friendly greeting for World", result.Messages[1].Content.(*mcp.TextContent).Text)
		})

		// Test document_writer prompt (typed) with length
		s.Run("DocumentWriterTypedPrompt", func() {
			result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
				Name: "document_writer",
				Arguments: map[string]string{
					"documentType": "blog post",
					"topic":        "artificial intelligence",
					"tone":         "conversational",
					"length":       "500",
				},
			})
			s.Require().NoError(err)
			s.Require().Len(result.Messages, 2)
			s.NotEmpty(result.Description)

			s.Equal("system", string(result.Messages[0].Role))
			s.Equal("You are an expert writer who creates high-quality content.", result.Messages[0].Content.(*mcp.TextContent).Text)
			s.Equal("user", string(result.Messages[1].Role))
			s.Contains(result.Messages[1].Content.(*mcp.TextContent).Text, "Write a blog post about artificial intelligence using a conversational tone")
			s.Contains(result.Messages[1].Content.(*mcp.TextContent).Text, "approximately 500 words")
			s.Equal("Generated writing prompt based on user specifications", result.Description)
		})

		s.Run("DocumentWriterWithoutOptionalLength", func() {
			result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
				Name: "document_writer",
				Arguments: map[string]string{
					"documentType": "essay",
					"topic":        "climate change",
					"tone":         "formal",
				},
			})
			s.Require().NoError(err)
			s.Require().Len(result.Messages, 2)

			s.Equal("user", string(result.Messages[1].Role))
			userContent := result.Messages[1].Content.(*mcp.TextContent).Text
			s.Contains(userContent, "Write a essay about climate change using a formal tone")
			s.NotContains(userContent, "approximately") // No length specified
		})
	})
}

func (s *CliPromptTestSuite) TestListing() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("prompt-server"),
			mcpio.WithPrompt("greeter", "Generates a friendly greeting", greeterPrompt),
			mcpio.WithTypedPrompt("document_writer", "Generates writing prompts with typed arguments", documentPrompt),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		result, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
		s.Require().NoError(err)
		s.Require().Len(result.Prompts, 2)

		// Create a map for easier lookup
		promptMap := make(map[string]*mcp.Prompt)
		for _, prompt := range result.Prompts {
			promptMap[prompt.Name] = prompt
		}

		// Check greeter prompt
		greeter := promptMap["greeter"]
		s.Require().NotNil(greeter)
		s.Equal("greeter", greeter.Name)
		s.Equal("Generates a friendly greeting", greeter.Description)

		// Check document_writer prompt
		docWriter := promptMap["document_writer"]
		s.Require().NotNil(docWriter)
		s.Equal("document_writer", docWriter.Name)
		s.Equal("Generates writing prompts with typed arguments", docWriter.Description)

		// Typed prompts should have arguments defined
		s.NotNil(docWriter.Arguments)
		s.NotEmpty(docWriter.Arguments)

		// Check that required fields are marked as required
		argMap := make(map[string]*mcp.PromptArgument)
		for _, arg := range docWriter.Arguments {
			argMap[arg.Name] = arg
		}

		// documentType, topic, tone should be required; length should be optional
		s.True(argMap["documentType"].Required)
		s.True(argMap["topic"].Required)
		s.True(argMap["tone"].Required)

		// length field may or may not be present depending on struct tag omitempty
		if lengthArg, exists := argMap["length"]; exists {
			s.False(lengthArg.Required)
		}
	})
}

func (s *CliPromptTestSuite) TestErrorHandling() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("prompt-server"),
			mcpio.WithPrompt("greeter", "Generates a friendly greeting", greeterPrompt),
			mcpio.WithTypedPrompt("document_writer", "Generates writing prompts with typed arguments", documentPrompt),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		// Test calling non-existent prompt
		s.Run("UnknownPrompt", func() {
			_, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
				Name:      "non_existent",
				Arguments: map[string]string{},
			})
			s.Require().Error(err)
			s.Contains(err.Error(), "unknown prompt")
		})

		// Test typed prompt with missing fields - these should succeed with empty string values
		testCases := []struct {
			name             string
			args             map[string]string
			expectedContains []string
		}{
			{
				name: "MissingDocumentType",
				args: map[string]string{
					"topic": "test topic",
					"tone":  "casual",
				},
				expectedContains: []string{"Write a  about test topic using a casual tone"},
			},
			{
				name: "MissingTopic",
				args: map[string]string{
					"documentType": "blog post",
					"tone":         "casual",
				},
				expectedContains: []string{"Write a blog post about  using a casual tone"},
			},
			{
				name: "MissingTone",
				args: map[string]string{
					"documentType": "blog post",
					"topic":        "test topic",
				},
				expectedContains: []string{"Write a blog post about test topic using a  tone"},
			},
			{
				name: "AllRequiredFieldsPresent",
				args: map[string]string{
					"documentType": "blog post",
					"topic":        "test topic",
					"tone":         "casual",
				},
				expectedContains: []string{"Write a blog post about test topic using a casual tone"},
			},
			{
				name:             "EmptyArguments",
				args:             map[string]string{},
				expectedContains: []string{"Write a  about  using a  tone"},
			},
		}

		for _, tc := range testCases {
			s.Run("DocumentWriter"+tc.name, func() {
				result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
					Name:      "document_writer",
					Arguments: tc.args,
				})

				s.Require().NoError(err)
				s.Require().Len(result.Messages, 2)
				s.Equal("system", string(result.Messages[0].Role))
				s.Equal("user", string(result.Messages[1].Role))

				// Check that the user message contains expected content
				userContent := result.Messages[1].Content.(*mcp.TextContent).Text
				for _, expected := range tc.expectedContains {
					s.Contains(userContent, expected)
				}
			})
		}
	})
}

func (s *CliPromptTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()

	// Verify binary was created
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}
