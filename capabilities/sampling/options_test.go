package sampling

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertFloatEqual handles both exact equality for 0.0 and epsilon comparison for non-zero values
func assertFloatEqual(t *testing.T, expected, actual float64) {
	t.Helper()
	if expected == 0.0 {
		assert.Equal(t, expected, actual, "Exact equality check for 0.0 expected") //nolint:testifylint
	} else {
		assert.InEpsilon(t, expected, actual, 0.001)
	}
}

func TestWithMaxTokens(t *testing.T) {
	t.Run("sets valid maxTokens", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithMaxTokens(500)
		err := opt(config)
		require.NoError(t, err)
		assert.Equal(t, 500, config.maxTokens)
	})

	t.Run("rejects zero maxTokens", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithMaxTokens(0)
		err := opt(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maxTokens must be greater than 0")
	})

	t.Run("rejects negative maxTokens", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithMaxTokens(-100)
		err := opt(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maxTokens must be greater than 0")
	})
}

func TestWithSystemPrompt(t *testing.T) {
	t.Run("sets system prompt", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithSystemPrompt("You are a helpful assistant")
		err := opt(config)
		require.NoError(t, err)
		assert.Equal(t, "You are a helpful assistant", config.systemPrompt)
	})

	t.Run("allows empty prompt", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithSystemPrompt("")
		err := opt(config)
		require.NoError(t, err)
		assert.Empty(t, config.systemPrompt)
	})
}

func TestWithTemperature(t *testing.T) {
	t.Run("sets valid temperature", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithTemperature(0.7)
		err := opt(config)
		require.NoError(t, err)
		require.NotNil(t, config.temperature)
		assertFloatEqual(t, 0.7, *config.temperature)
	})

	t.Run("allows minimum temperature", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithTemperature(0.0)
		err := opt(config)
		require.NoError(t, err)
		require.NotNil(t, config.temperature)
		assertFloatEqual(t, 0.0, *config.temperature)
	})

	t.Run("allows maximum temperature", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithTemperature(2.0)
		err := opt(config)
		require.NoError(t, err)
		require.NotNil(t, config.temperature)
		assertFloatEqual(t, 2.0, *config.temperature)
	})

	t.Run("rejects negative temperature", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithTemperature(-0.1)
		err := opt(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "temperature must be between 0.0 and 2.0")
	})

	t.Run("rejects too high temperature", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithTemperature(2.1)
		err := opt(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "temperature must be between 0.0 and 2.0")
	})
}

func TestWithStopSequences(t *testing.T) {
	t.Run("sets stop sequences", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithStopSequences("END", "STOP", "DONE")
		err := opt(config)
		require.NoError(t, err)
		assert.Equal(t, []string{"END", "STOP", "DONE"}, config.stopSequences)
	})

	t.Run("allows empty sequences", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithStopSequences()
		err := opt(config)
		require.NoError(t, err)
		assert.Empty(t, config.stopSequences)
	})
}

func TestWithMetadata(t *testing.T) {
	t.Run("sets metadata", func(t *testing.T) {
		config := &messageConfig{}
		metadata := map[string]interface{}{
			"key": "value",
			"num": 42,
		}
		opt := WithMetadata(metadata)
		err := opt(config)
		require.NoError(t, err)
		assert.Equal(t, metadata, config.metadata)
	})

	t.Run("allows nil metadata", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithMetadata(nil)
		err := opt(config)
		require.NoError(t, err)
		assert.Nil(t, config.metadata)
	})
}

func TestWithIncludeContext(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"none is valid", "none", false},
		{"thisServer is valid", "thisServer", false},
		{"allServers is valid", "allServers", false},
		{"invalid value", "invalid", true},
		{"empty value", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &messageConfig{}
			opt := WithIncludeContext(tt.value)
			err := opt(config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "includeContext must be")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.value, config.includeContext)
			}
		})
	}
}

func TestWithModelPreferences(t *testing.T) {
	t.Run("sets model preferences", func(t *testing.T) {
		config := &messageConfig{}
		prefs := &mcp.ModelPreferences{
			CostPriority:         0.5,
			IntelligencePriority: 0.8,
			SpeedPriority:        0.2,
		}
		opt := WithModelPreferences(prefs)
		err := opt(config)
		require.NoError(t, err)
		assert.Equal(t, prefs, config.modelPreferences)
	})

	t.Run("rejects nil preferences", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithModelPreferences(nil)
		err := opt(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "model preferences cannot be nil")
	})
}

func TestPriorityFunctions(t *testing.T) {
	tests := []struct {
		name    string
		fn      func(float64) MessageOption
		field   string
		value   float64
		wantErr bool
		errMsg  string
	}{
		{"valid cost priority", WithCostPriority, "cost", 0.5, false, ""},
		{"cost priority at minimum", WithCostPriority, "cost", 0.0, false, ""},
		{"cost priority at maximum", WithCostPriority, "cost", 1.0, false, ""},
		{"cost priority too low", WithCostPriority, "cost", -0.1, true, "cost priority must be between 0.0 and 1.0"},
		{"cost priority too high", WithCostPriority, "cost", 1.1, true, "cost priority must be between 0.0 and 1.0"},

		{"valid intelligence priority", WithIntelligencePriority, "intelligence", 0.7, false, ""},
		{"intelligence priority at minimum", WithIntelligencePriority, "intelligence", 0.0, false, ""},
		{"intelligence priority at maximum", WithIntelligencePriority, "intelligence", 1.0, false, ""},
		{"intelligence priority too low", WithIntelligencePriority, "intelligence", -0.1, true, "intelligence priority must be between 0.0 and 1.0"},
		{"intelligence priority too high", WithIntelligencePriority, "intelligence", 1.1, true, "intelligence priority must be between 0.0 and 1.0"},

		{"valid speed priority", WithSpeedPriority, "speed", 0.3, false, ""},
		{"speed priority at minimum", WithSpeedPriority, "speed", 0.0, false, ""},
		{"speed priority at maximum", WithSpeedPriority, "speed", 1.0, false, ""},
		{"speed priority too low", WithSpeedPriority, "speed", -0.1, true, "speed priority must be between 0.0 and 1.0"},
		{"speed priority too high", WithSpeedPriority, "speed", 1.1, true, "speed priority must be between 0.0 and 1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &messageConfig{}
			opt := tt.fn(tt.value)
			err := opt(config)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config.modelPreferences)

			fieldMap := map[string]float64{
				"cost":         config.modelPreferences.CostPriority,
				"intelligence": config.modelPreferences.IntelligencePriority,
				"speed":        config.modelPreferences.SpeedPriority,
			}

			assertFloatEqual(t, tt.value, fieldMap[tt.field])
		})
	}
}

func TestWithModelHints(t *testing.T) {
	t.Run("sets single hint", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithModelHints("claude-3-5-sonnet")
		err := opt(config)
		require.NoError(t, err)
		require.NotNil(t, config.modelPreferences)
		require.Len(t, config.modelPreferences.Hints, 1)
		assert.Equal(t, "claude-3-5-sonnet", config.modelPreferences.Hints[0].Name)
	})

	t.Run("sets multiple hints", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithModelHints("claude", "sonnet", "gpt-4")
		err := opt(config)
		require.NoError(t, err)
		require.NotNil(t, config.modelPreferences)
		require.Len(t, config.modelPreferences.Hints, 3)
		assert.Equal(t, "claude", config.modelPreferences.Hints[0].Name)
		assert.Equal(t, "sonnet", config.modelPreferences.Hints[1].Name)
		assert.Equal(t, "gpt-4", config.modelPreferences.Hints[2].Name)
	})

	t.Run("rejects empty hints", func(t *testing.T) {
		config := &messageConfig{}
		opt := WithModelHints()
		err := opt(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one model hint must be provided")
	})
}

func TestApplyOptions(t *testing.T) {
	t.Run("applies multiple options", func(t *testing.T) {
		opts := []MessageOption{
			WithSystemPrompt("You are helpful"),
			WithTemperature(0.7),
			WithStopSequences("END"),
			WithCostPriority(0.5),
			WithModelHints("claude"),
		}

		config, err := ApplyOptions(opts)
		require.NoError(t, err)
		assert.Equal(t, "You are helpful", config.systemPrompt)
		assertFloatEqual(t, 0.7, *config.temperature)
		assert.Equal(t, []string{"END"}, config.stopSequences)
		assertFloatEqual(t, 0.5, config.modelPreferences.CostPriority)
		assert.Len(t, config.modelPreferences.Hints, 1)
	})

	t.Run("returns error on invalid option", func(t *testing.T) {
		opts := []MessageOption{
			WithSystemPrompt("You are helpful"),
			WithTemperature(3.0), // Invalid
		}

		config, err := ApplyOptions(opts)
		require.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "temperature must be between 0.0 and 2.0")
	})

	t.Run("applies no options", func(t *testing.T) {
		config, err := ApplyOptions(nil)
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, 1000, config.maxTokens) // Default value
		assert.Empty(t, config.systemPrompt)
		assert.Nil(t, config.temperature)
	})
}

func TestToCreateMessageParams(t *testing.T) {
	messages := []*mcp.SamplingMessage{
		{
			Role:    mcp.Role("user"),
			Content: &mcp.TextContent{Text: "Hello"},
		},
	}

	t.Run("converts full config", func(t *testing.T) {
		temp := 0.7
		config := &messageConfig{
			maxTokens:     100,
			systemPrompt:  "Be helpful",
			temperature:   &temp,
			stopSequences: []string{"END", "STOP"},
			modelPreferences: &mcp.ModelPreferences{
				CostPriority: 0.5,
			},
			metadata:       map[string]string{"key": "value"},
			includeContext: "thisServer",
		}

		params := config.ToCreateMessageParams(messages)

		assert.Equal(t, messages, params.Messages)
		assert.Equal(t, int64(100), params.MaxTokens)
		assert.Equal(t, "Be helpful", params.SystemPrompt)
		assertFloatEqual(t, 0.7, params.Temperature)
		assert.Equal(t, []string{"END", "STOP"}, params.StopSequences)
		assertFloatEqual(t, 0.5, params.ModelPreferences.CostPriority)
		assert.Equal(t, map[string]string{"key": "value"}, params.Metadata)
		assert.Equal(t, "thisServer", params.IncludeContext)
	})

	t.Run("converts minimal config", func(t *testing.T) {
		config := &messageConfig{
			maxTokens: 1000, // Default value
		}
		params := config.ToCreateMessageParams(messages)

		assert.Equal(t, messages, params.Messages)
		assert.Equal(t, int64(1000), params.MaxTokens)
		assert.Empty(t, params.SystemPrompt)
		assertFloatEqual(t, 0.0, params.Temperature)
		assert.Nil(t, params.StopSequences)
		assert.Nil(t, params.ModelPreferences)
		assert.Nil(t, params.Metadata)
		assert.Empty(t, params.IncludeContext)
	})
}
