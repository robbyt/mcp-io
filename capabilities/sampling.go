package capabilities

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/capabilities/sampling"
)

// SamplingCapabilities indicates the client supports LLM sampling.
type SamplingCapabilities struct{}

// SupportsSampling returns true if the client supports LLM sampling (CreateMessage).
func (s *Session) SupportsSampling() bool {
	return s.session.InitializeParams().Capabilities.Sampling != nil
}

// CreateMessage asks the client's LLM to generate a response to the provided messages.
// This enables servers to use the client's LLM for analysis, suggestions, or processing.
// Options can be provided to configure the sampling behavior (maxTokens, temperature, model preferences, etc).
// If maxTokens is not specified via WithMaxTokens, defaults to 1000.
// Returns nil error and empty result if the client doesn't support sampling.
func (s *Session) CreateMessage(ctx context.Context, messages []*sampling.Message, opts ...sampling.MessageOption) (*sampling.MessageResult, error) {
	// Convert our Message type to mcp.SamplingMessage
	mcpMessages := make([]*mcp.SamplingMessage, len(messages))
	for i, msg := range messages {
		mcpMessages[i] = &mcp.SamplingMessage{
			Role:    mcp.Role(msg.Role),
			Content: &mcp.TextContent{Text: msg.Content},
		}
	}

	// Apply options to build params
	config, err := sampling.ApplyOptions(opts)
	if err != nil {
		return nil, err
	}
	params := config.ToCreateMessageParams(mcpMessages)

	result, err := s.session.CreateMessage(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert result back to our types
	mcpContent, ok := result.Content.(*mcp.TextContent)
	if !ok {
		return &sampling.MessageResult{
			Role:    string(result.Role),
			Content: sampling.TextContent{Text: ""},
		}, nil
	}

	return &sampling.MessageResult{
		Role:    string(result.Role),
		Content: sampling.TextContent{Text: mcpContent.Text},
	}, nil
}

// CreateMessageRaw provides direct access to the MCP CreateMessage API with full control over parameters.
// Use this when you need advanced control beyond what the options provide, or when working with raw MCP types directly.
func (s *Session) CreateMessageRaw(ctx context.Context, params *mcp.CreateMessageParams) (*mcp.CreateMessageResult, error) {
	return s.session.CreateMessage(ctx, params)
}
