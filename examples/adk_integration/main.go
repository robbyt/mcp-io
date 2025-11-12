// Package main demonstrates integrating mcp-io with the Google Agent Development Kit (ADK).
//
// # Architecture: MCP Server + ADK Agent
//
// This example uses mcp-io to host tools on an MCP server and exposes them to an ADK
// agent via mcptoolset. This contrasts with the native ADK functiontool pattern,
// where tools are defined in-process with the agent.
//
// Benefits of the MCP + ADK approach:
//   - Decoupling: Tools can be developed, deployed, and scaled independently of the agent
//   - Language Agnostic: A Go agent can use MCP tool servers written in any language
//   - Reusability: The same MCP tool server can be used by multiple different agents
//   - Protocol Standard: Tools follow the Model Context Protocol specification
//
// When to use each approach:
//   - MCP tools: Cross-language integration, shared tooling, production deployments
//   - Native ADK functiontool: Simple in-process tools, prototyping, single-language systems
//
// # Usage
//
// Terminal chat (default):
//
//	export GOOGLE_API_KEY="..."
//	go run .
//
// Web UI at http://localhost:8080:
//
//	go run . web api webui
//
// Requires GOOGLE_API_KEY environment variable for Gemini API access.
//
// Example queries:
//   - "What's the coldest major US city right now?"
//   - "Compare weather in Seattle and Miami"
//   - "Which cities are warmer than 70°F?"
//
// See README.md in this directory for complete documentation.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"charm.land/fantasy/providers/anthropic"
	"charm.land/fantasy/providers/google"
	"charm.land/fantasy/providers/openai"
	adapter "github.com/robbyt/fantasy-adapters/adk"
	mcpio "github.com/robbyt/mcp-io"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher/adk"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/server/restapi/services"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/mcptoolset"
	"google.golang.org/genai"
)

func createModel(ctx context.Context, providerName, modelName string) (model.LLM, error) {
	// Default model names per provider
	defaultModels := map[string]string{
		"gemini":    "gemini-2.0-flash-exp",
		"google":    "gemini-2.0-flash-exp",
		"anthropic": "claude-3-5-sonnet-20241022",
		"openai":    "gpt-4o",
	}

	// Use default if model not specified
	if modelName == "" {
		var ok bool
		modelName, ok = defaultModels[providerName]
		if !ok {
			return nil, fmt.Errorf("unknown provider: %s (supported: gemini, google, anthropic, openai)", providerName)
		}
	}

	switch providerName {
	case "gemini":
		// Native Gemini implementation
		apiKey := os.Getenv("GOOGLE_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("GOOGLE_API_KEY environment variable not set")
		}
		return gemini.NewModel(ctx, modelName, &genai.ClientConfig{
			APIKey: apiKey,
		})

	case "google":
		// Fantasy Google provider
		apiKey := os.Getenv("GOOGLE_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("GOOGLE_API_KEY environment variable not set")
		}
		provider, err := google.New(google.WithGeminiAPIKey(apiKey))
		if err != nil {
			return nil, err
		}
		fantasyModel, err := provider.LanguageModel(ctx, modelName)
		if err != nil {
			return nil, err
		}
		return adapter.NewAdapter(fantasyModel), nil

	case "anthropic":
		// Fantasy Anthropic provider
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
		}
		provider, err := anthropic.New(anthropic.WithAPIKey(apiKey))
		if err != nil {
			return nil, err
		}
		fantasyModel, err := provider.LanguageModel(ctx, modelName)
		if err != nil {
			return nil, err
		}
		return adapter.NewAdapter(fantasyModel), nil

	case "openai":
		// Fantasy OpenAI provider
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
		}
		provider, err := openai.New(openai.WithAPIKey(apiKey))
		if err != nil {
			return nil, err
		}
		fantasyModel, err := provider.LanguageModel(ctx, modelName)
		if err != nil {
			return nil, err
		}
		return adapter.NewAdapter(fantasyModel), nil

	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: gemini, google, anthropic, openai)", providerName)
	}
}

func main() {
	// Parse command-line flags
	provider := flag.String("provider", "gemini", "LLM provider (gemini, google, anthropic, openai)")
	modelName := flag.String("model", "", "Model name (provider-specific, uses defaults if empty)")
	temperature := flag.Float64("temperature", 0.7, "LLM temperature (0.0-1.0)")
	maxTokens := flag.Int64("max-tokens", 2000, "Maximum output tokens")
	flag.Parse()

	ctx := context.Background()

	// Create MCP server with weather tools
	handler, clientTransport, err := mcpio.NewInMemoryPair(ctx,
		mcpio.WithName("weather-mcp"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("geocode_city", "Convert US city name to geographic coordinates", geocodeTool),
		mcpio.WithTool("get_weather", "Get NOAA weather forecast for coordinates", getWeatherTool),
	)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Start MCP server in background
	var wg sync.WaitGroup
	wg.Go(func() {
		if err := handler.Run(ctx); err != nil {
			log.Printf("MCP server error: %v", err)
		}
	})
	defer wg.Wait()

	// Create LLM model based on provider
	llm, err := createModel(ctx, *provider, *modelName)
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create MCP toolset for ADK
	mcpToolSet, err := mcptoolset.New(mcptoolset.Config{
		Transport: clientTransport,
	})
	if err != nil {
		log.Fatalf("Failed to create MCP toolset: %v", err)
	}

	// Create ADK LLM agent
	temp := float32(*temperature)
	agent, err := llmagent.New(llmagent.Config{
		Name:        "weather_agent",
		Model:       llm,
		Description: "Weather comparison agent for US cities",
		Instruction: weatherAgentInstruction,
		Toolsets:    []tool.Toolset{mcpToolSet},
		GenerateContentConfig: &genai.GenerateContentConfig{
			Temperature:     &temp,
			MaxOutputTokens: int32(*maxTokens),
		},
	})
	if err != nil {
		log.Fatalf("Failed to create ADK agent: %v", err)
	}

	// Launch agent via ADK launcher
	config := &adk.Config{
		AgentLoader: services.NewSingleAgentLoader(agent),
	}
	launcher := full.NewLauncher()

	// Execute launcher with remaining args (e.g., "web api webui")
	if err := launcher.Execute(ctx, config, flag.Args()); err != nil {
		log.Fatalf("Launcher failed: %v\n\n%s", err, launcher.CommandLineSyntax())
	}
}

// weatherAgentInstruction defines the agent's behavior and tool usage guidance.
const weatherAgentInstruction = `You are a weather comparison assistant for US cities.

You have access to two tools:
1. **geocode_city**: Converts a US city name to geographic coordinates (latitude/longitude)
2. **get_weather**: Fetches NOAA weather forecast data for specific coordinates

When answering questions about weather:
1. Identify which cities need to be queried
2. For each city:
   - First use geocode_city to get coordinates
   - Then use get_weather with those coordinates to get forecast data
3. Compare the results and provide clear, concise answers
4. Always cite the actual temperature values and conditions from the data

Example workflow for "What's the coldest city right now?":
1. Call geocode_city for multiple major cities (New York, Los Angeles, Chicago, etc.)
2. Call get_weather for each set of coordinates (these can be done in parallel)
3. Compare the current_temp values
4. Return the coldest city with its temperature and conditions

When comparing cities:
- Focus on current_temp for "right now" questions
- Use the forecast array for future predictions
- Include relevant details like conditions and windspeed when helpful
- Be concise but informative

Always show your data sources and reasoning.`
