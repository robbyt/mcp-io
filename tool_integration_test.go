//go:build integration

package mcpio_test

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type ToolIntegrationTestSuite struct {
	testutil.IntegrationSuite
}

func TestToolIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ToolIntegrationTestSuite))
}

func (s *ToolIntegrationTestSuite) TestToolHandlerIntegration() {
	s.Run("ToUpper", func() {
		// Create the server with the same tools as cli-tool
		handler, err := mcpio.NewHandler(
			mcpio.WithName("text-processor"),
			mcpio.WithTool("to_upper", "Convert text to uppercase", func(ctx context.Context, toolCtx mcpio.RequestContext, input struct {
				Text string `json:"text"`
			}) (struct {
				Result string `json:"result"`
			}, error,
			) {
				return struct {
					Result string `json:"result"`
				}{Result: "HELLO WORLD"}, nil
			}),
		)
		s.Require().NoError(err)

		// Connect client using testutil helper
		session := testutil.ConnectInMemory(s.T(), handler)

		// Call the to_upper tool
		result, err := session.CallTool(s.Ctx, &mcp.CallToolParams{
			Name:      "to_upper",
			Arguments: map[string]any{"text": "hello world"},
		})
		s.Require().NoError(err)

		s.False(result.IsError)
		s.Len(result.Content, 1)
	})

	s.Run("CountWords", func() {
		// Create server with count tool
		handler, err := mcpio.NewHandler(
			mcpio.WithName("text-processor"),
			mcpio.WithTool("count", "Count words or characters in text", func(ctx context.Context, toolCtx mcpio.RequestContext, input struct {
				Text string `json:"text"`
				Type string `json:"type"`
			}) (struct {
				Count int    `json:"count"`
				Type  string `json:"type"`
			}, error,
			) {
				count := 0
				if input.Type == "words" && input.Text == "this is a test" {
					count = 4
				}
				return struct {
					Count int    `json:"count"`
					Type  string `json:"type"`
				}{Count: count, Type: input.Type}, nil
			}),
		)
		s.Require().NoError(err)

		// Connect client using testutil helper
		session := testutil.ConnectInMemory(s.T(), handler)

		result, err := session.CallTool(s.Ctx, &mcp.CallToolParams{
			Name: "count",
			Arguments: map[string]any{
				"text": "this is a test",
				"type": "words",
			},
		})
		s.Require().NoError(err)
		s.False(result.IsError)
	})

	s.Run("ValidationError", func() {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("text-processor"),
			mcpio.WithTool("count", "Count words or characters in text", func(ctx context.Context, toolCtx mcpio.RequestContext, input struct {
				Text string `json:"text"`
				Type string `json:"type"`
			}) (struct {
				Count int    `json:"count"`
				Type  string `json:"type"`
			}, error,
			) {
				if input.Type != "words" && input.Type != "chars" {
					return struct {
						Count int    `json:"count"`
						Type  string `json:"type"`
					}{}, mcpio.NewToolError("unsupported count type: " + input.Type)
				}
				return struct {
					Count int    `json:"count"`
					Type  string `json:"type"`
				}{Count: 0, Type: input.Type}, nil
			}),
		)
		s.Require().NoError(err)

		// Connect client using testutil helper
		session := testutil.ConnectInMemory(s.T(), handler)

		result, err := session.CallTool(s.Ctx, &mcp.CallToolParams{
			Name: "count",
			Arguments: map[string]any{
				"text": "this is a test",
				"type": "invalid_type",
			},
		})
		s.Require().NoError(err)

		s.True(result.IsError)
		s.Len(result.Content, 1)
		if textContent, ok := result.Content[0].(*mcp.TextContent); s.True(ok) {
			s.Contains(textContent.Text, "unsupported count type:")
		}
	})

	s.Run("RawToolErrorCodePreservation", func() {
		// Create handler with raw tool that returns errors with codes
		handler, err := mcpio.NewHandler(
			mcpio.WithName("error-test"),
			mcpio.WithRawTool("validate", "Test error code preservation",
				`{"type":"object","properties":{"input":{"type":"string"}}}`,
				func(ctx context.Context, toolCtx mcpio.RequestContext, input []byte) ([]byte, error) {
					// Return an error with code
					return nil, mcpio.NewToolErrorWithCode("validation failed", mcpio.ErrorCodeValidation)
				},
			),
		)
		s.Require().NoError(err)

		// Connect client using testutil helper
		session := testutil.ConnectInMemory(s.T(), handler)

		result, err := session.CallTool(s.Ctx, &mcp.CallToolParams{
			Name:      "validate",
			Arguments: map[string]any{"input": "test"},
		})
		s.Require().NoError(err)

		// Verify error code is preserved in the error message
		s.True(result.IsError, "result should be marked as error")
		s.Require().Len(result.Content, 1, "result should have exactly one content item")
		s.Require().IsType(&mcp.TextContent{}, result.Content[0], "content should be TextContent")
		textContent := result.Content[0].(*mcp.TextContent)
		s.Contains(textContent.Text, "[VALIDATION_ERROR]", "error message should include error code")
		s.Contains(textContent.Text, "validation failed", "error message should include original message")
	})

	s.Run("RawToolErrorWithoutCode", func() {
		// Create handler with raw tool that returns errors without codes
		handler, err := mcpio.NewHandler(
			mcpio.WithName("error-test"),
			mcpio.WithRawTool("process", "Test error without code",
				`{"type":"object","properties":{"input":{"type":"string"}}}`,
				func(ctx context.Context, toolCtx mcpio.RequestContext, input []byte) ([]byte, error) {
					// Return an error without code
					return nil, mcpio.NewToolError("processing failed")
				},
			),
		)
		s.Require().NoError(err)

		// Connect client using testutil helper
		session := testutil.ConnectInMemory(s.T(), handler)

		result, err := session.CallTool(s.Ctx, &mcp.CallToolParams{
			Name:      "process",
			Arguments: map[string]any{"input": "test"},
		})
		s.Require().NoError(err)

		// Verify error message appears without code prefix
		s.True(result.IsError, "result should be marked as error")
		s.Require().Len(result.Content, 1, "result should have exactly one content item")
		s.Require().IsType(&mcp.TextContent{}, result.Content[0], "content should be TextContent")
		textContent := result.Content[0].(*mcp.TextContent)
		s.Equal("processing failed", textContent.Text, "error message should match exactly")
		s.NotContains(textContent.Text, "[", "error message should not have code prefix")
	})
}
