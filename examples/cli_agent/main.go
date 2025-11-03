package main

import (
	"context"
	"fmt"
	"log"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/capabilities"
)

// AnalyzeInput represents input for code analysis
type AnalyzeInput struct {
	Code     string `json:"code"               jsonschema:"Code to analyze"`
	Language string `json:"language,omitempty" jsonschema:"Programming language (optional),enum=,enum=go,enum=python,enum=javascript,enum=rust,enum=java"`
}

// AnalyzeOutput represents the analysis results
type AnalyzeOutput struct {
	Analysis     string   `json:"analysis"     jsonschema:"AI-generated analysis"`
	Suggestions  []string `json:"suggestions"  jsonschema:"Improvement suggestions"`
	IssuesFound  int      `json:"issuesFound"  jsonschema:"Number of issues detected"`
	SamplingUsed bool     `json:"samplingUsed" jsonschema:"Whether sampling was available"`
}

// notifyProgress sends a progress notification if supported.
// Treats progress as best-effort - logs failures but never aborts tool execution.
func notifyProgress(ctx context.Context, session *capabilities.Session, reqCtx mcpio.RequestContext,
	progress, total float64, message string,
) {
	opts := []capabilities.ProgressOption{
		capabilities.WithProgressToken(reqCtx.GetProgressToken()),
	}
	if message != "" {
		opts = append(opts, capabilities.WithProgressMessage(message))
	}

	if err := session.NotifyProgress(ctx, progress, total, opts...); err != nil {
		// Log failure but don't abort - progress is best-effort
		_ = session.LogWarning(ctx, "Failed to send progress notification", //nolint:errcheck
			map[string]any{"error": err.Error()})
	}
}

// analyzeTool uses the client's LLM (sampling) to analyze code
func analyzeTool(ctx context.Context, toolCtx mcpio.RequestContext, input AnalyzeInput) (AnalyzeOutput, error) {
	session := toolCtx.GetSession()
	if session == nil {
		return AnalyzeOutput{}, fmt.Errorf("no session available")
	}

	// Check if client supports sampling
	if !session.SupportsSampling() {
		return AnalyzeOutput{
			Analysis:     "Sampling not supported by client",
			Suggestions:  []string{"Use a client that supports sampling to enable AI analysis"},
			IssuesFound:  0,
			SamplingUsed: false,
		}, nil
	}

	// Send progress notification and log analysis start (best-effort)
	notifyProgress(ctx, session, toolCtx, 0.0, 1.0, "Starting code analysis")
	_ = session.LogInfo(ctx, "Starting code analysis", map[string]any{ //nolint:errcheck
		"language":  input.Language,
		"code_size": len(input.Code),
	})

	// Build prompt for the LLM
	prompt := "Analyze this code for bugs, security issues, and potential improvements:\n\n"
	if input.Language != "" {
		prompt += fmt.Sprintf("Language: %s\n\n", input.Language)
	}
	prompt += "```\n" + input.Code + "\n```\n\n"
	prompt += "Provide:\n1. A summary of what the code does\n2. Any issues found (bugs, security, performance)\n3. Specific suggestions for improvement"

	// Send message to client's LLM
	result, err := session.CreateMessage(ctx, []*capabilities.Message{{
		Role:    "user",
		Content: prompt,
	}}, 2000)
	if err != nil {
		// Log error but don't fail if logging fails during error handling
		_ = session.LogError(ctx, "Sampling failed", map[string]any{"error": err.Error()}) //nolint:errcheck
		return AnalyzeOutput{}, fmt.Errorf("sampling failed: %w", err)
	}

	notifyProgress(ctx, session, toolCtx, 0.5, 1.0, "Processing AI response")

	// Parse the response to extract suggestions
	analysis := result.Content.Text
	suggestions := extractSuggestions(analysis)
	issuesFound := countIssues(analysis)

	// Log completion (best-effort)
	_ = session.LogInfo(ctx, "Analysis completed", map[string]any{ //nolint:errcheck
		"issues_found": issuesFound,
		"suggestions":  len(suggestions),
	})

	notifyProgress(ctx, session, toolCtx, 1.0, 1.0, "")

	return AnalyzeOutput{
		Analysis:     analysis,
		Suggestions:  suggestions,
		IssuesFound:  issuesFound,
		SamplingUsed: true,
	}, nil
}

// ImproveInput represents input for code improvement
type ImproveInput struct {
	Code     string `json:"code"               jsonschema:"Code to improve"`
	Focus    string `json:"focus,omitempty"    jsonschema:"What to focus on,enum=,enum=performance,enum=readability,enum=security,enum=testing"`
	Language string `json:"language,omitempty" jsonschema:"Programming language (optional)"`
}

// ImproveOutput represents improved code
type ImproveOutput struct {
	ImprovedCode string `json:"improvedCode" jsonschema:"AI-improved code"`
	Changes      string `json:"changes"      jsonschema:"Description of changes made"`
	SamplingUsed bool   `json:"samplingUsed" jsonschema:"Whether sampling was available"`
}

// improveTool uses sampling to suggest improved code
func improveTool(ctx context.Context, toolCtx mcpio.RequestContext, input ImproveInput) (ImproveOutput, error) {
	session := toolCtx.GetSession()
	if session == nil || !session.SupportsSampling() {
		return ImproveOutput{
			ImprovedCode: input.Code,
			Changes:      "Sampling not available - no improvements made",
			SamplingUsed: false,
		}, nil
	}

	// Build improvement prompt
	prompt := "Improve this code"
	if input.Focus != "" {
		prompt += fmt.Sprintf(" focusing on %s", input.Focus)
	}
	prompt += ":\n\n"
	if input.Language != "" {
		prompt += fmt.Sprintf("Language: %s\n\n", input.Language)
	}
	prompt += "```\n" + input.Code + "\n```\n\n"
	prompt += "Provide the improved code and explain what changes were made."

	result, err := session.CreateMessage(ctx, []*capabilities.Message{{
		Role:    "user",
		Content: prompt,
	}}, 3000)
	if err != nil {
		return ImproveOutput{}, fmt.Errorf("sampling failed: %w", err)
	}

	return ImproveOutput{
		ImprovedCode: result.Content.Text,
		Changes:      "See improved code above",
		SamplingUsed: true,
	}, nil
}

// Helper functions

func extractSuggestions(analysis string) []string {
	// Simple extraction - in a real implementation you'd parse more carefully
	suggestions := []string{}
	// Look for numbered suggestions in the analysis
	// This is a simplified version - real implementation would be more sophisticated
	if len(analysis) > 0 {
		suggestions = append(suggestions, "See full analysis for detailed suggestions")
	}
	return suggestions
}

func countIssues(analysis string) int {
	// Simple count - look for keywords
	// Real implementation would parse the analysis more carefully
	issueKeywords := []string{"bug", "issue", "problem", "error", "vulnerability", "security"}
	count := 0
	for _, keyword := range issueKeywords {
		if contains(analysis, keyword) {
			count++
		}
	}
	return count
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr))
}

func main() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("code-agent"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("analyze_code", "Analyze code using AI to find bugs and suggest improvements", analyzeTool),
		mcpio.WithTool("improve_code", "Use AI to improve code quality", improveTool),
	)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	if err := handler.ServeStdio(context.Background()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
