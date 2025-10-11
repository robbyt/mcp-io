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

// HttpServerTestSuite tests the http_server example
type HttpServerTestSuite struct {
	testutil.ExampleTestSuite
}

func (s *HttpServerTestSuite) SetupSuite() {
	// Get project root - we're in examples/http_server
	_, b, _, _ := runtime.Caller(0)
	exampleDir := filepath.Dir(b)
	s.ProjectRoot = filepath.Join(exampleDir, "..", "..")
	s.ExampleName = "http_server"

	// Call parent SetupSuite
	s.ExampleTestSuite.SetupSuite()
}

func TestHttpServerSuite(t *testing.T) {
	suite.Run(t, new(HttpServerTestSuite))
}

func (s *HttpServerTestSuite) TestTools() {
	ctx := s.T().Context()

	// Create server using the same logic as main()
	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("example-calculator"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("calculate", "Perform arithmetic operations", calculator),
			mcpio.WithTool("echo", "Echo a message", echo),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		// Test calculator tool
		s.Run("Calculator", func() {
			testCases := []struct {
				name      string
				operation string
				a, b      float64
				expected  float64
				hasError  bool
			}{
				{"Add", "add", 5, 3, 8, false},
				{"Subtract", "subtract", 10, 4, 6, false},
				{"Multiply", "multiply", 7, 6, 42, false},
				{"Divide", "divide", 15, 3, 5, false},
				{"DivideByZero", "divide", 10, 0, 0, true},
				{"InvalidOperation", "invalid", 1, 2, 0, true},
			}

			for _, tc := range testCases {
				s.Run(tc.name, func() {
					result, err := session.CallTool(ctx, &mcp.CallToolParams{
						Name: "calculate",
						Arguments: map[string]any{
							"operation": tc.operation,
							"a":         tc.a,
							"b":         tc.b,
						},
					})

					if tc.hasError {
						if err != nil {
							// Protocol-level error
							s.Require().Error(err)
						} else {
							// Tool-level error
							s.True(result.IsError)
						}
					} else {
						s.Require().NoError(err)
						s.Require().False(result.IsError)
						s.Require().NotNil(result.StructuredContent)

						resultMap, ok := result.StructuredContent.(map[string]any)
						s.Require().True(ok)
						s.InEpsilon(tc.expected, resultMap["result"], 0.0001, "Result mismatch for operation %s", tc.operation)
					}
				})
			}
		})

		// Test echo tool
		s.Run("Echo", func() {
			result, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name: "echo",
				Arguments: map[string]any{
					"message": "Hello, World!",
				},
			})
			s.Require().NoError(err)
			s.Require().False(result.IsError)
			s.Require().NotNil(result.StructuredContent)

			resultMap, ok := result.StructuredContent.(map[string]any)
			s.Require().True(ok)
			s.Equal("Hello, World!", resultMap["echo"])
		})
	})
}

func (s *HttpServerTestSuite) TestToolListing() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("example-calculator"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("calculate", "Perform arithmetic operations", calculator),
			mcpio.WithTool("echo", "Echo a message", echo),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		s.Require().NoError(err)
		s.Require().Len(result.Tools, 2)

		// Check tool names and descriptions
		expectedTools := map[string]string{
			"calculate": "Perform arithmetic operations",
			"echo":      "Echo a message",
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

func (s *HttpServerTestSuite) TestErrorHandling() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("example-calculator"),
			mcpio.WithTool("calculate", "Perform arithmetic operations", calculator),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		s.Run("UnknownTool", func() {
			_, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      "unknown_tool",
				Arguments: map[string]any{},
			})
			s.Require().Error(err)
			s.Contains(err.Error(), "unknown tool")
		})

		s.Run("MalformedArguments", func() {
			result, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name: "calculate",
				Arguments: map[string]any{
					"operation": "add",
					// Missing required arguments a and b
				},
			})
			// This should either error or return tool-level error
			if err == nil {
				s.True(result.IsError, "Expected tool error for malformed arguments")
			} else {
				s.Error(err)
			}
		})
	})
}

func (s *HttpServerTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()

	// Verify binary was created
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}
