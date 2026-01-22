// Package main demonstrates mcp-io integration with Google ADK.
//
// The MCP server hosts tools that an ADK agent accesses via mcptoolset.
// This enables language-agnostic, reusable tool servers.
//
// See README.md for usage examples and API key setup.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"

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

func createModel(ctx context.Context, modelName string) (model.LLM, error) {
	if modelName == "" {
		modelName = "gemini-2.5-flash-lite"
	}

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY environment variable not set")
	}
	return gemini.NewModel(ctx, modelName, &genai.ClientConfig{
		APIKey: apiKey,
	})
}

func main() {
	// Parse command-line flags
	modelName := flag.String("model", "", "Gemini model name (defaults to gemini-2.5-flash-lite)")
	temperature := flag.Float64("temperature", 0.7, "LLM temperature (0.0-1.0)")
	maxTokens := flag.Int64("max-tokens", 2000, "Maximum output tokens")
	logFile := flag.String("log", "", "Log file path (optional, logs discarded if not set)")
	flag.Parse()

	// Initialize slog - write to file if specified, otherwise discard
	var logWriter io.Writer = io.Discard
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer f.Close()
		logWriter = f
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(logWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	handler, clientTransport, err := mcpio.NewInMemoryPair(ctx,
		mcpio.WithName("weather-mcp"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("geocode_city", "Convert US city name to geographic coordinates", geocodeTool),
		mcpio.WithTool("get_weather", "Get NOAA weather forecast for coordinates", getWeatherTool),
	)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		if err := handler.Run(ctx); err != nil {
			slog.Error("MCP server error", "error", err)
		}
	})
	defer wg.Wait()

	llm, err := createModel(ctx, *modelName)
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	mcpToolSet, err := mcptoolset.New(mcptoolset.Config{
		Transport: clientTransport,
	})
	if err != nil {
		log.Fatalf("Failed to create MCP toolset: %v", err)
	}

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

	config := &adk.Config{
		AgentLoader: services.NewSingleAgentLoader(agent),
	}
	launcher := full.NewLauncher()

	if err := launcher.Execute(ctx, config, flag.Args()); err != nil {
		log.Fatalf("Launcher failed: %v (syntax: %s)", err, launcher.CommandLineSyntax())
	}
}

const weatherAgentInstruction = `You are a weather comparison assistant for US cities.

You have two tools:
1. geocode_city: Convert city name to coordinates
2. get_weather: Get NOAA forecast for coordinates

To answer queries:
1. Use geocode_city for each city mentioned
2. Use get_weather with the returned coordinates
3. Compare results and cite actual temperature values

Focus on current_temp for "right now" questions and forecast array for future predictions.`
