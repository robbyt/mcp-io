// Package sampling provides options for configuring LLM sampling requests.
package sampling

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MessageOption is a functional option for configuring CreateMessage requests.
type MessageOption func(*messageConfig) error

// WithMaxTokens sets the maximum number of tokens to generate.
// If not specified, defaults to 1000.
func WithMaxTokens(maxTokens int) MessageOption {
	return func(c *messageConfig) error {
		if maxTokens <= 0 {
			return fmt.Errorf("maxTokens must be greater than 0, got %d", maxTokens)
		}
		c.maxTokens = maxTokens
		return nil
	}
}

// WithSystemPrompt sets an optional system prompt for the LLM.
// The client may modify or omit this prompt.
func WithSystemPrompt(prompt string) MessageOption {
	return func(c *messageConfig) error {
		c.systemPrompt = prompt
		return nil
	}
}

// WithTemperature sets the sampling temperature.
// Valid range is 0.0 to 2.0. Higher values make output more random.
func WithTemperature(temp float64) MessageOption {
	return func(c *messageConfig) error {
		if temp < 0.0 || temp > 2.0 {
			return fmt.Errorf("temperature must be between 0.0 and 2.0, got %f", temp)
		}
		c.temperature = &temp
		return nil
	}
}

// WithStopSequences sets sequences where the LLM should stop generating.
func WithStopSequences(sequences ...string) MessageOption {
	return func(c *messageConfig) error {
		c.stopSequences = sequences
		return nil
	}
}

// WithMetadata sets provider-specific metadata to pass through to the LLM.
// The format of this metadata is provider-specific.
func WithMetadata(metadata any) MessageOption {
	return func(c *messageConfig) error {
		c.metadata = metadata
		return nil
	}
}

// WithIncludeContext requests to include context from one or more MCP servers.
// Valid values are "none", "thisServer", or "allServers".
// The client may ignore this request.
func WithIncludeContext(context string) MessageOption {
	return func(c *messageConfig) error {
		validValues := map[string]bool{
			"none":       true,
			"thisServer": true,
			"allServers": true,
		}
		if !validValues[context] {
			return fmt.Errorf("includeContext must be 'none', 'thisServer', or 'allServers', got %q", context)
		}
		c.includeContext = context
		return nil
	}
}

// WithModelPreferences sets the complete model preferences for model selection.
// Use this for full control, or use the individual priority functions below.
func WithModelPreferences(prefs *mcp.ModelPreferences) MessageOption {
	return func(c *messageConfig) error {
		if prefs == nil {
			return fmt.Errorf("model preferences cannot be nil")
		}
		c.modelPreferences = prefs
		return nil
	}
}

// WithCostPriority sets how much to prioritize cost when selecting a model.
// Value must be between 0.0 (cost not important) and 1.0 (cost most important).
func WithCostPriority(priority float64) MessageOption {
	return func(c *messageConfig) error {
		if priority < 0.0 || priority > 1.0 {
			return fmt.Errorf("cost priority must be between 0.0 and 1.0, got %f", priority)
		}
		if c.modelPreferences == nil {
			c.modelPreferences = &mcp.ModelPreferences{}
		}
		c.modelPreferences.CostPriority = priority
		return nil
	}
}

// WithIntelligencePriority sets how much to prioritize intelligence when selecting a model.
// Value must be between 0.0 (intelligence not important) and 1.0 (intelligence most important).
func WithIntelligencePriority(priority float64) MessageOption {
	return func(c *messageConfig) error {
		if priority < 0.0 || priority > 1.0 {
			return fmt.Errorf("intelligence priority must be between 0.0 and 1.0, got %f", priority)
		}
		if c.modelPreferences == nil {
			c.modelPreferences = &mcp.ModelPreferences{}
		}
		c.modelPreferences.IntelligencePriority = priority
		return nil
	}
}

// WithSpeedPriority sets how much to prioritize speed (latency) when selecting a model.
// Value must be between 0.0 (speed not important) and 1.0 (speed most important).
func WithSpeedPriority(priority float64) MessageOption {
	return func(c *messageConfig) error {
		if priority < 0.0 || priority > 1.0 {
			return fmt.Errorf("speed priority must be between 0.0 and 1.0, got %f", priority)
		}
		if c.modelPreferences == nil {
			c.modelPreferences = &mcp.ModelPreferences{}
		}
		c.modelPreferences.SpeedPriority = priority
		return nil
	}
}

// WithModelHints provides hints for model selection.
// Hints are treated as substrings of model names.
// For example, "claude-3-5-sonnet" matches "claude-3-5-sonnet-20241022".
func WithModelHints(hints ...string) MessageOption {
	return func(c *messageConfig) error {
		if len(hints) == 0 {
			return fmt.Errorf("at least one model hint must be provided")
		}

		if c.modelPreferences == nil {
			c.modelPreferences = &mcp.ModelPreferences{}
		}

		c.modelPreferences.Hints = make([]*mcp.ModelHint, len(hints))
		for i, hint := range hints {
			c.modelPreferences.Hints[i] = &mcp.ModelHint{
				Name: hint,
			}
		}
		return nil
	}
}

// ApplyOptions applies all the given options to create a messageConfig.
func ApplyOptions(opts []MessageOption) (*messageConfig, error) {
	config := &messageConfig{
		maxTokens: 1000, // Default value
	}
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, err
		}
	}
	return config, nil
}
