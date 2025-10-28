package main

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
)

// createLLMMessage sends a prompt to the client's LLM and returns the text response
func createLLMMessage(ctx context.Context, prompt string, maxTokens int, preferredModel string) (string, error) {
	session := mcpio.GetSession(ctx)
	if session == nil {
		return "", fmt.Errorf("no session available")
	}

	params := &mcp.CreateMessageParams{
		Messages: []*mcp.SamplingMessage{{
			Role:    "user",
			Content: &mcp.TextContent{Text: prompt},
		}},
		MaxTokens: int64(maxTokens),
	}
	if preferredModel != "" {
		params.ModelPreferences = &mcp.ModelPreferences{
			Hints: []*mcp.ModelHint{{Name: preferredModel}},
		}
	}
	result, err := session.CreateMessageRaw(ctx, params)
	if err != nil {
		return "", err
	}

	textContent, ok := result.Content.(*mcp.TextContent)
	if !ok {
		return "", fmt.Errorf("unexpected content type from LLM")
	}

	return textContent.Text, nil
}
