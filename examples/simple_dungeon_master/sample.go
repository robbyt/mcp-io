package main

import (
	"context"
	"fmt"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/robbyt/mcp-io/capabilities/sampling"
)

// createLLMMessage sends a prompt to the client's LLM and returns the text response
func createLLMMessage(ctx context.Context, toolCtx mcpio.RequestContext, prompt string, maxTokens int, preferredModel string) (string, error) {
	session := toolCtx.GetSession()
	if session == nil {
		return "", fmt.Errorf("no session available")
	}

	opts := []sampling.MessageOption{
		sampling.WithMaxTokens(maxTokens),
	}
	if preferredModel != "" {
		opts = append(opts, sampling.WithModelHints(preferredModel))
	}

	result, err := session.CreateMessage(ctx, []*capabilities.Message{{
		Role:    "user",
		Content: prompt,
	}}, opts...)
	if err != nil {
		return "", err
	}

	return result.Content.Text, nil
}
