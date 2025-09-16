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
		handler, err := mcpio.NewPromptHandler(
			mcpio.WithName("prompt-server"),
			mcpio.WithPrompt("greeter", "Generates a friendly greeting", greeterPrompt),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		testCases := []struct {
			name     string
			args     map[string]string
			expected []struct {
				role    string
				content string
			}
		}{
			{
				name: "GreeterWithName",
				args: map[string]string{"name": "Alice"},
				expected: []struct {
					role    string
					content string
				}{
					{"system", "You are a helpful assistant."},
					{"user", "Create a friendly greeting for Alice"},
				},
			},
			{
				name: "GreeterWithoutName",
				args: map[string]string{},
				expected: []struct {
					role    string
					content string
				}{
					{"system", "You are a helpful assistant."},
					{"user", "Create a friendly greeting for World"},
				},
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
					Name:      "greeter",
					Arguments: tc.args,
				})
				s.Require().NoError(err)
				s.Require().Len(result.Messages, len(tc.expected))

				for i, expectedMsg := range tc.expected {
					s.Equal(expectedMsg.role, string(result.Messages[i].Role))
					// For TextContent, extract the text
					if textContent, ok := result.Messages[i].Content.(*mcp.TextContent); ok {
						s.Equal(expectedMsg.content, textContent.Text)
					}
				}
			})
		}
	})
}

func (s *CliPromptTestSuite) TestListing() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewPromptHandler(
			mcpio.WithName("prompt-server"),
			mcpio.WithPrompt("greeter", "Generates a friendly greeting", greeterPrompt),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		result, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
		s.Require().NoError(err)
		s.Require().Len(result.Prompts, 1)

		prompt := result.Prompts[0]
		s.Equal("greeter", prompt.Name)
		s.Equal("Generates a friendly greeting", prompt.Description)
		// Arguments can be nil for prompts without required arguments
	})
}

func (s *CliPromptTestSuite) TestErrorHandling() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewPromptHandler(
			mcpio.WithName("prompt-server"),
			mcpio.WithPrompt("greeter", "Generates a friendly greeting", greeterPrompt),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		// Test calling non-existent prompt
		_, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
			Name:      "non_existent",
			Arguments: map[string]string{},
		})
		s.Require().Error(err)
		s.Contains(err.Error(), "unknown prompt")
	})
}

func (s *CliPromptTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()

	// Verify binary was created
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}
