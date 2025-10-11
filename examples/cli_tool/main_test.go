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

// CliToolTestSuite tests the cli_tool example
type CliToolTestSuite struct {
	testutil.ExampleTestSuite
}

func (s *CliToolTestSuite) SetupSuite() {
	// Get project root - we're in examples/cli_tool
	_, b, _, _ := runtime.Caller(0)
	exampleDir := filepath.Dir(b)
	s.ProjectRoot = filepath.Join(exampleDir, "..", "..")
	s.ExampleName = "cli_tool"

	// Call parent SetupSuite
	s.ExampleTestSuite.SetupSuite()
}

func TestCliToolSuite(t *testing.T) {
	suite.Run(t, new(CliToolTestSuite))
}

func (s *CliToolTestSuite) TestFunctionality() {
	ctx := s.T().Context()

	// Create server using the same logic as main()
	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("text-processor"),
			mcpio.WithTool("to_upper", "Convert text to uppercase", toUpperCase),
			mcpio.WithTool("to_lower", "Convert text to lowercase", toLowerCase),
			mcpio.WithTool("reverse", "Reverse text", reverseText),
			mcpio.WithTool("count", "Count words, characters, or lines", countText),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		// Test all tools
		testCases := []struct {
			name     string
			toolName string
			args     map[string]any
			expected map[string]any
		}{
			{
				name:     "ToUpper",
				toolName: "to_upper",
				args:     map[string]any{"text": "hello world"},
				expected: map[string]any{"result": "HELLO WORLD"},
			},
			{
				name:     "ToLower",
				toolName: "to_lower",
				args:     map[string]any{"text": "HELLO WORLD"},
				expected: map[string]any{"result": "hello world"},
			},
			{
				name:     "Reverse",
				toolName: "reverse",
				args:     map[string]any{"text": "hello"},
				expected: map[string]any{"result": "olleh"},
			},
			{
				name:     "CountWords",
				toolName: "count",
				args:     map[string]any{"text": "hello world test", "type": "words"},
				expected: map[string]any{"count": float64(3), "type": "words"},
			},
			{
				name:     "CountChars",
				toolName: "count",
				args:     map[string]any{"text": "hello", "type": "chars"},
				expected: map[string]any{"count": float64(5), "type": "chars"},
			},
			{
				name:     "CountLines",
				toolName: "count",
				args:     map[string]any{"text": "line1\nline2\nline3", "type": "lines"},
				expected: map[string]any{"count": float64(3), "type": "lines"},
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				result, err := session.CallTool(ctx, &mcp.CallToolParams{
					Name:      tc.toolName,
					Arguments: tc.args,
				})
				s.Require().NoError(err)
				s.Require().False(result.IsError)

				// Use StructuredContent if available (structured output)
				if result.StructuredContent != nil {
					resultMap, ok := result.StructuredContent.(map[string]any)
					s.Require().True(ok, "StructuredContent should be a map")
					for key, expectedValue := range tc.expected {
						s.Equal(expectedValue, resultMap[key])
					}
				} else {
					// Fallback to parsing Content as TextContent
					s.Require().Len(result.Content, 1)
					textContent, ok := result.Content[0].(*mcp.TextContent)
					s.Require().True(ok)

					// The text should contain JSON that we can validate
					s.Contains(textContent.Text, tc.name)
				}
			})
		}
	})
}

func (s *CliToolTestSuite) TestErrorHandling() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("text-processor"),
			mcpio.WithTool("count", "Count words, characters, or lines", countText),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		s.Run("InvalidCountType", func() {
			result, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      "count",
				Arguments: map[string]any{"text": "hello", "type": "invalid"},
			})
			s.Require().NoError(err)
			// Should return error content, not protocol error
			s.Require().True(result.IsError)
			s.Require().Len(result.Content, 1)
		})

		s.Run("UnknownTool", func() {
			_, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      "unknown_tool",
				Arguments: map[string]any{},
			})
			s.Require().Error(err)
			s.Contains(err.Error(), "unknown tool")
		})
	})
}

func (s *CliToolTestSuite) TestListing() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("text-processor"),
			mcpio.WithTool("to_upper", "Convert text to uppercase", toUpperCase),
			mcpio.WithTool("to_lower", "Convert text to lowercase", toLowerCase),
			mcpio.WithTool("reverse", "Reverse text", reverseText),
			mcpio.WithTool("count", "Count words, characters, or lines", countText),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		s.Require().NoError(err)
		s.Require().Len(result.Tools, 4)

		// Check tool names and descriptions
		expectedTools := map[string]string{
			"to_upper": "Convert text to uppercase",
			"to_lower": "Convert text to lowercase",
			"reverse":  "Reverse text",
			"count":    "Count words, characters, or lines",
		}

		actualTools := make(map[string]string)
		for _, tool := range result.Tools {
			actualTools[tool.Name] = tool.Description
		}

		s.Equal(expectedTools, actualTools)

		// Verify schemas are present
		for _, tool := range result.Tools {
			s.NotNil(tool.InputSchema, "Tool %s should have input schema", tool.Name)
		}
	})
}

func (s *CliToolTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()

	// Verify binary was created
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}
