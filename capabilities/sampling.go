package capabilities

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Message represents a message for LLM sampling.
type Message struct {
	Role    string
	Content string
}

// MessageResult represents the result of an LLM sampling request.
type MessageResult struct {
	Role    string
	Content TextContent
}

// TextContent represents text content in a message.
type TextContent struct {
	Text string
}

// SupportsSampling returns true if the client supports LLM sampling (CreateMessage).
func (s *sessionCapability) SupportsSampling() bool {
	return s.session.InitializeParams().Capabilities.Sampling != nil
}

// CreateMessage asks the client's LLM to generate a response to the provided messages.
// This enables servers to use the client's LLM for analysis, suggestions, or processing.
// Returns nil error and empty result if the client doesn't support sampling.
func (s *sessionCapability) CreateMessage(ctx context.Context, messages []*Message, maxTokens int) (*MessageResult, error) {
	// Convert our Message type to mcp.SamplingMessage
	mcpMessages := make([]*mcp.SamplingMessage, len(messages))
	for i, msg := range messages {
		mcpMessages[i] = &mcp.SamplingMessage{
			Role:    mcp.Role(msg.Role),
			Content: &mcp.TextContent{Text: msg.Content},
		}
	}

	params := &mcp.CreateMessageParams{
		Messages:  mcpMessages,
		MaxTokens: int64(maxTokens),
	}

	result, err := s.session.CreateMessage(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert result back to our types
	mcpContent, ok := result.Content.(*mcp.TextContent)
	if !ok {
		return &MessageResult{
			Role:    string(result.Role),
			Content: TextContent{Text: ""},
		}, nil
	}

	return &MessageResult{
		Role:    string(result.Role),
		Content: TextContent{Text: mcpContent.Text},
	}, nil
}

// CreateMessageRaw provides direct access to the MCP CreateMessage API with full control over parameters.
// Use this when you need to specify model preferences, temperature, system prompts, or other advanced options.
func (s *sessionCapability) CreateMessageRaw(ctx context.Context, params *mcp.CreateMessageParams) (*mcp.CreateMessageResult, error) {
	return s.session.CreateMessage(ctx, params)
}
