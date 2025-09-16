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

// CliCombinedTestSuite tests the cli_combined example
type CliCombinedTestSuite struct {
	testutil.ExampleTestSuite
}

func (s *CliCombinedTestSuite) SetupSuite() {
	// Get project root - we're in examples/cli_combined
	_, b, _, _ := runtime.Caller(0)
	exampleDir := filepath.Dir(b)
	s.ProjectRoot = filepath.Join(exampleDir, "..", "..")
	s.ExampleName = "cli_combined"

	// Call parent SetupSuite
	s.ExampleTestSuite.SetupSuite()
}

func TestCliCombinedSuite(t *testing.T) {
	suite.Run(t, new(CliCombinedTestSuite))
}

func (s *CliCombinedTestSuite) TestTools() {
	ctx := s.T().Context()

	// Create server using the same logic as main()
	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("document-processor"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("format_text", "Clean and format text by normalizing whitespace", formatText),
			mcpio.WithTool("analyze_text", "Analyze text and provide statistics", analyzeText),
			mcpio.WithResourceTemplate("doc://templates/email", "Email template", documentResource),
			mcpio.WithResourceTemplate("doc://templates/report", "Report template", documentResource),
			mcpio.WithResourceTemplate("doc://templates/memo", "Memo template", documentResource),
			mcpio.WithResourceTemplate("doc://samples/analysis", "Sample analysis data", documentResource),
			mcpio.WithPrompt("document_writer", "Generate writing prompts for different document types", documentPrompt),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		s.Run("FormatText", func() {
			result, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      "format_text",
				Arguments: map[string]any{"text": "  hello    world  \n  extra   spaces  "},
			})
			s.Require().NoError(err)
			s.Require().False(result.IsError)
			s.Require().NotNil(result.StructuredContent)

			resultMap, ok := result.StructuredContent.(map[string]any)
			s.Require().True(ok)
			s.Equal("hello world extra spaces", resultMap["result"])
		})

		s.Run("AnalyzeText", func() {
			result, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      "analyze_text",
				Arguments: map[string]any{"text": "Hello world! This is a test. How are you?"},
			})
			s.Require().NoError(err)
			s.Require().False(result.IsError)
			s.Require().NotNil(result.StructuredContent)

			resultMap, ok := result.StructuredContent.(map[string]any)
			s.Require().True(ok)
			s.InDelta(9, resultMap["wordCount"], 0)
			s.InDelta(41, resultMap["charCount"], 0)
			s.InDelta(3, resultMap["sentenceCount"], 0)
			s.Equal("< 1 minute", resultMap["readingTime"])
		})
	})
}

func (s *CliCombinedTestSuite) TestResources() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("document-processor"),
			mcpio.WithResourceTemplate("doc://templates/email", "Email template", documentResource),
			mcpio.WithResourceTemplate("doc://templates/report", "Report template", documentResource),
			mcpio.WithResourceTemplate("doc://templates/memo", "Memo template", documentResource),
			mcpio.WithResourceTemplate("doc://samples/analysis", "Sample analysis data", documentResource),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		testCases := []struct {
			name        string
			uri         string
			expectError bool
			mimeType    string
			contains    string
		}{
			{
				name:        "EmailTemplate",
				uri:         "doc://templates/email",
				expectError: false,
				mimeType:    "text/plain",
				contains:    "Subject: [SUBJECT]",
			},
			{
				name:        "ReportTemplate",
				uri:         "doc://templates/report",
				expectError: false,
				mimeType:    "text/plain",
				contains:    "# [TITLE]",
			},
			{
				name:        "MemoTemplate",
				uri:         "doc://templates/memo",
				expectError: false,
				mimeType:    "text/plain",
				contains:    "MEMORANDUM",
			},
			{
				name:        "SampleAnalysis",
				uri:         "doc://samples/analysis",
				expectError: false,
				mimeType:    "text/plain",
				contains:    "sample_analysis",
			},
			{
				name:        "NonExistentResource",
				uri:         "doc://templates/nonexistent",
				expectError: true,
				mimeType:    "",
				contains:    "",
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
					URI: tc.uri,
				})

				if tc.expectError {
					s.Require().Error(err)
				} else {
					s.Require().NoError(err)
					s.Require().Len(result.Contents, 1)

					content := result.Contents[0]
					s.Equal(tc.mimeType, content.MIMEType)
					s.Contains(content.Text, tc.contains)
				}
			})
		}
	})
}

func (s *CliCombinedTestSuite) TestPrompts() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("document-processor"),
			mcpio.WithPrompt("document_writer", "Generate writing prompts for different document types", documentPrompt),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		testCases := []struct {
			name           string
			args           map[string]string
			expectError    bool
			systemContains string
			userContains   string
		}{
			{
				name: "EmailPrompt",
				args: map[string]string{
					"document_type": "email",
					"topic":         "project update",
					"tone":          "professional",
				},
				expectError:    false,
				systemContains: "email writer",
				userContains:   "project update",
			},
			{
				name: "ReportPrompt",
				args: map[string]string{
					"document_type": "report",
					"topic":         "quarterly results",
					"tone":          "formal",
				},
				expectError:    false,
				systemContains: "report writer",
				userContains:   "quarterly results",
			},
			{
				name: "MemoPrompt",
				args: map[string]string{
					"document_type": "memo",
					"topic":         "policy change",
				},
				expectError:    false,
				systemContains: "business writer",
				userContains:   "policy change",
			},
			{
				name: "MissingDocumentType",
				args: map[string]string{
					"topic": "test topic",
				},
				expectError: true,
			},
			{
				name: "MissingTopic",
				args: map[string]string{
					"document_type": "email",
				},
				expectError: true,
			},
			{
				name: "UnsupportedDocumentType",
				args: map[string]string{
					"document_type": "novel",
					"topic":         "test topic",
				},
				expectError: true,
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
					Name:      "document_writer",
					Arguments: tc.args,
				})

				if tc.expectError {
					s.Require().Error(err)
				} else {
					s.Require().NoError(err)
					s.Require().Len(result.Messages, 2)

					// Check system message
					systemMsg := result.Messages[0]
					s.Equal("system", string(systemMsg.Role))
					if textContent, ok := systemMsg.Content.(*mcp.TextContent); ok {
						s.Contains(textContent.Text, tc.systemContains)
					}

					// Check user message
					userMsg := result.Messages[1]
					s.Equal("user", string(userMsg.Role))
					if textContent, ok := userMsg.Content.(*mcp.TextContent); ok {
						s.Contains(textContent.Text, tc.userContains)
					}
				}
			})
		}
	})
}

func (s *CliCombinedTestSuite) TestListing() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("document-processor"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("format_text", "Clean and format text by normalizing whitespace", formatText),
			mcpio.WithTool("analyze_text", "Analyze text and provide statistics", analyzeText),
			mcpio.WithResourceTemplate("doc://templates/email", "Email template", documentResource),
			mcpio.WithResourceTemplate("doc://templates/report", "Report template", documentResource),
			mcpio.WithResourceTemplate("doc://templates/memo", "Memo template", documentResource),
			mcpio.WithResourceTemplate("doc://samples/analysis", "Sample analysis data", documentResource),
			mcpio.WithPrompt("document_writer", "Generate writing prompts for different document types", documentPrompt),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		// Test tools listing
		s.Run("ListTools", func() {
			result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
			s.Require().NoError(err)
			s.Require().Len(result.Tools, 2)

			expectedTools := map[string]string{
				"format_text":  "Clean and format text by normalizing whitespace",
				"analyze_text": "Analyze text and provide statistics",
			}

			actualTools := make(map[string]string)
			for _, tool := range result.Tools {
				actualTools[tool.Name] = tool.Description
			}

			s.Equal(expectedTools, actualTools)
		})

		// Test resources listing
		s.Run("ListResources", func() {
			result, err := session.ListResources(ctx, &mcp.ListResourcesParams{})
			s.Require().NoError(err)
			// Resources may not appear in the list if they're templates
			s.T().Logf("Found %d resources", len(result.Resources))
		})

		// Test prompts listing
		s.Run("ListPrompts", func() {
			result, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
			s.Require().NoError(err)
			s.Require().Len(result.Prompts, 1)

			prompt := result.Prompts[0]
			s.Equal("document_writer", prompt.Name)
			s.Equal("Generate writing prompts for different document types", prompt.Description)
		})
	})
}

func (s *CliCombinedTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()

	// Verify binary was created
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}
