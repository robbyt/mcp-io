package sampling

import "github.com/modelcontextprotocol/go-sdk/mcp"

// messageConfig holds the configuration for a CreateMessage request.
type messageConfig struct {
	maxTokens        int
	systemPrompt     string
	temperature      *float64
	stopSequences    []string
	modelPreferences *mcp.ModelPreferences
	metadata         any
	includeContext   string
}

// ToCreateMessageParams converts the config to MCP CreateMessageParams.
func (c *messageConfig) ToCreateMessageParams(messages []*mcp.SamplingMessage) *mcp.CreateMessageParams {
	params := &mcp.CreateMessageParams{
		Messages:  messages,
		MaxTokens: int64(c.maxTokens),
	}

	if c.systemPrompt != "" {
		params.SystemPrompt = c.systemPrompt
	}

	if c.temperature != nil {
		params.Temperature = *c.temperature
	}

	if len(c.stopSequences) > 0 {
		params.StopSequences = c.stopSequences
	}

	if c.modelPreferences != nil {
		params.ModelPreferences = c.modelPreferences
	}

	if c.metadata != nil {
		params.Metadata = c.metadata
	}

	if c.includeContext != "" {
		params.IncludeContext = c.includeContext
	}

	return params
}
