package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	mcpio "github.com/robbyt/mcp-io"
)

// Tool input/output types
type TextInput struct {
	Text string `json:"text" jsonschema:"Text to process"`
}

type TextOutput struct {
	Result string `json:"result" jsonschema:"Processed text"`
}

type AnalysisInput struct {
	Text string `json:"text" jsonschema:"Text to analyze"`
}

type AnalysisOutput struct {
	WordCount     int    `json:"wordCount"     jsonschema:"Number of words"`
	CharCount     int    `json:"charCount"     jsonschema:"Number of characters"`
	SentenceCount int    `json:"sentenceCount" jsonschema:"Number of sentences"`
	ReadingTime   string `json:"readingTime"   jsonschema:"Estimated reading time"`
}

// Tool functions
func formatText(ctx context.Context, input TextInput) (TextOutput, error) {
	// Clean up text: trim whitespace, normalize spacing
	cleaned := strings.TrimSpace(input.Text)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return TextOutput{Result: cleaned}, nil
}

func analyzeText(ctx context.Context, input AnalysisInput) (AnalysisOutput, error) {
	text := input.Text

	// Word count
	words := strings.Fields(text)
	wordCount := len(words)

	// Character count
	charCount := len([]rune(text))

	// Sentence count (simple heuristic)
	sentences := strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '!' || r == '?'
	})
	sentenceCount := len(sentences)

	// Reading time (average 200 words per minute)
	readingMinutes := float64(wordCount) / 200.0
	var readingTime string
	if readingMinutes < 1 {
		readingTime = "< 1 minute"
	} else {
		readingTime = strings.TrimSuffix(strings.TrimSuffix(
			time.Duration(readingMinutes*float64(time.Minute)).String(), "0s"), "m0s") + " minutes"
	}

	return AnalysisOutput{
		WordCount:     wordCount,
		CharCount:     charCount,
		SentenceCount: sentenceCount,
		ReadingTime:   readingTime,
	}, nil
}

// Resource handler - provides sample documents and templates
func documentResource(ctx context.Context, uri string) (*mcpio.ResourceContent, error) {
	var content []byte
	var mimeType string

	switch uri {
	case "doc://templates/email":
		content = []byte("Subject: [SUBJECT]\n\nDear [RECIPIENT],\n\n[BODY]\n\nBest regards,\n[SENDER]")
		mimeType = "text/plain"
	case "doc://templates/report":
		content = []byte("# [TITLE]\n\n## Executive Summary\n[SUMMARY]\n\n## Analysis\n[ANALYSIS]\n\n## Recommendations\n[RECOMMENDATIONS]\n\n## Conclusion\n[CONCLUSION]")
		mimeType = "text/plain"
	case "doc://templates/memo":
		content = []byte("MEMORANDUM\n\nTO: [RECIPIENT]\nFROM: [SENDER]\nDATE: [DATE]\nRE: [SUBJECT]\n\n[BODY]")
		mimeType = "text/plain"
	case "doc://samples/analysis":
		content = []byte(`{"sample_analysis": {"word_count": 150, "char_count": 890, "sentence_count": 8, "reading_time": "< 1 minute"}}`)
		mimeType = "text/plain"
	default:
		return nil, mcpio.ResourceNotFoundError(uri)
	}

	return &mcpio.ResourceContent{
		Content:  content,
		MIMEType: mimeType,
	}, nil
}

// Prompt handler - generates document writing prompts
func documentPrompt(ctx context.Context, args map[string]any) (*mcpio.PromptResult, error) {
	documentType, _ := args["document_type"].(string)
	topic, _ := args["topic"].(string)
	tone, _ := args["tone"].(string)

	if documentType == "" {
		return nil, mcpio.ValidationError("document_type is required")
	}
	if topic == "" {
		return nil, mcpio.ValidationError("topic is required")
	}
	if tone == "" {
		tone = "professional"
	}

	var systemPrompt, userPrompt string

	switch strings.ToLower(documentType) {
	case "email":
		systemPrompt = "You are an expert email writer. Create clear, well-structured emails that are appropriate for the specified tone and context."
		userPrompt = "Write an email about: " + topic + ". Use a " + tone + " tone. Include a clear subject line and proper email structure."
	case "report":
		systemPrompt = "You are a professional report writer. Create comprehensive, well-organized reports with clear sections and actionable insights."
		userPrompt = "Write a report about: " + topic + ". Use a " + tone + " tone. Include executive summary, analysis, and recommendations."
	case "memo":
		systemPrompt = "You are an experienced business writer. Create concise, professional memos that clearly communicate key information."
		userPrompt = "Write a memo about: " + topic + ". Use a " + tone + " tone. Keep it concise and action-oriented."
	default:
		return nil, mcpio.ValidationError("unsupported document type: " + documentType + " (supported: email, report, memo)")
	}

	return &mcpio.PromptResult{
		Messages: []mcpio.PromptMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}, nil
}

func main() {
	// Create comprehensive MCP handler with tools, resources, and prompts
	handler, err := mcpio.NewHandler(
		mcpio.WithName("document-processor"),
		mcpio.WithVersion("1.0.0"),

		// Tools for text processing
		mcpio.WithTool("format_text", "Clean and format text by normalizing whitespace", formatText),
		mcpio.WithTool("analyze_text", "Analyze text and provide statistics", analyzeText),

		// Resources for document templates and samples
		mcpio.WithResourceTemplate("doc://templates/email", "Email template", documentResource),
		mcpio.WithResourceTemplate("doc://templates/report", "Report template", documentResource),
		mcpio.WithResourceTemplate("doc://templates/memo", "Memo template", documentResource),
		mcpio.WithResourceTemplate("doc://samples/analysis", "Sample analysis data", documentResource),

		// Prompts for generating documents
		mcpio.WithPrompt("document_writer", "Generate writing prompts for different document types", documentPrompt),
	)
	if err != nil {
		log.Fatalf("Failed to create MCP handler: %v", err)
	}

	// Serve via stdio (standard for CLI MCP servers)
	if err := handler.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
		log.Fatalf("Failed to serve MCP via stdio: %v", err)
	}
}
