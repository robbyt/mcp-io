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

// CliSimpleTestSuite tests the cli_simple example
type CliSimpleTestSuite struct {
	testutil.ExampleTestSuite
}

func (s *CliSimpleTestSuite) SetupSuite() {
	// Get project root - we're in examples/cli_simple
	_, b, _, _ := runtime.Caller(0)
	exampleDir := filepath.Dir(b)
	s.ProjectRoot = filepath.Join(exampleDir, "..", "..")
	s.ExampleName = "cli_simple"

	// Call parent SetupSuite
	s.ExampleTestSuite.SetupSuite()
}

func TestCliSimpleSuite(t *testing.T) {
	suite.Run(t, new(CliSimpleTestSuite))
}

func (s *CliSimpleTestSuite) TestToUpper() {
	ctx := s.T().Context()

	// Create server using the same logic as main()
	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("simple-text-processor"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{"ToUpperTool", "hello world", "HELLO WORLD"},
			{"ToUpperEmptyString", "", ""},
			{"ToUpperSpecialCharacters", "hello, world! 123", "HELLO, WORLD! 123"},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				result, err := session.CallTool(ctx, &mcp.CallToolParams{
					Name:      "to_upper",
					Arguments: map[string]any{"text": tc.input},
				})
				s.Require().NoError(err)
				s.Require().False(result.IsError)

				// Use StructuredContent which should have the typed result directly
				s.Require().NotNil(result.StructuredContent, "StructuredContent should be populated")

				resultMap, ok := result.StructuredContent.(map[string]any)
				s.Require().True(ok, "StructuredContent should be a map")
				s.Equal(tc.expected, resultMap["result"])
			})
		}
	})
}

func (s *CliSimpleTestSuite) TestToolListing() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("simple-text-processor"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		s.Require().NoError(err)
		s.Require().Len(result.Tools, 1)

		tool := result.Tools[0]
		s.Equal("to_upper", tool.Name)
		s.Equal("Convert text to uppercase", tool.Description)
		s.NotNil(tool.InputSchema)
	})
}

func (s *CliSimpleTestSuite) TestErrorHandling() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("simple-text-processor"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		// Test calling non-existent tool
		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "non_existent",
			Arguments: map[string]any{},
		})
		s.Require().Error(err)
		s.Contains(err.Error(), "unknown tool")
	})
}

func (s *CliSimpleTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()

	// Verify binary was created
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}
