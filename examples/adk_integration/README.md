# ADK Weather Comparison Agent

A weather comparison agent built with mcp-io and Google ADK that queries NOAA data for US cities.

## Overview

This example demonstrates integrating mcp-io with the Google Agent Development Kit (ADK). The key pattern is using mcp-io to host tools on an MCP server and exposing them to an ADK agent via `mcptoolset`.

This contrasts with the native ADK `functiontool` pattern, where tools are defined in-process with the agent.

### Benefits of MCP + ADK Integration

- **Decoupling**: Tools can be developed, deployed, and scaled independently of the agent
- **Language Agnostic**: A Go agent can use MCP tool servers written in Python, Node.js, or any language
- **Reusability**: The same MCP tool server can be used by multiple different agents
- **Protocol Standard**: Tools follow the Model Context Protocol specification

## Prerequisites

### API Keys

Depending on which LLM provider you choose, you'll need the corresponding API key:

**Gemini (default):**
```bash
export GOOGLE_API_KEY="..."
```
Get your API key from [Google AI Studio](https://makersuite.google.com/app/apikey).

**Anthropic:**
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```
Get your API key from [Anthropic Console](https://console.anthropic.com/).

**OpenAI:**
```bash
export OPENAI_API_KEY="sk-..."
```
Get your API key from [OpenAI Platform](https://platform.openai.com/api-keys).

## Installation

```bash
cd examples/adk_integration
go mod download
go build .
```

## Usage

### Terminal Chat (Default)

```bash
go run .
```

Starts an interactive terminal chat session. Type your queries and the agent responds using the weather tools.

### Web UI

```bash
go run . web api webui
```

Launches a web interface at `http://localhost:8080` for browser-based interaction.

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--provider` | `gemini` | LLM provider (`gemini`, `google`, `anthropic`, `openai`) |
| `--model` | (provider-specific) | Model name (uses provider defaults if empty) |
| `--temperature` | `0.7` | LLM temperature (0.0-1.0) |
| `--max-tokens` | `2000` | Maximum output tokens |

### Provider-Specific Models

**Gemini (native Google ADK):**
- Default: `gemini-2.0-flash-exp`
- Available: `gemini-2.5-flash`, `gemini-2.5-pro`

**Google (via Fantasy):**
- Default: `gemini-2.0-flash-exp`
- Available: Same as Gemini provider

**Anthropic:**
- Default: `claude-3-5-sonnet-20241022`
- Available: `claude-haiku-4-5-20251001`, other Claude models

**OpenAI:**
- Default: `gpt-4o`
- Available: `gpt-4o-mini`, other GPT models

### Provider Examples

**Native Gemini (default):**
```bash
export GOOGLE_API_KEY="..."
go run .
```

**Anthropic/Claude:**
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
go run . --provider=anthropic
```

**OpenAI:**
```bash
export OPENAI_API_KEY="sk-..."
go run . --provider=openai --model=gpt-4o-mini
```

**Google via Fantasy (testing alternative path):**
```bash
export GOOGLE_API_KEY="..."
go run . --provider=google
```

## Example Queries

Once the agent is running, try asking:

```
What's the coldest major US city right now?

Compare weather in Seattle and Miami.

Which cities are warmer than 70°F?

What's the weather forecast for New York?
```

The agent will use the tools to query weather data and provide comparative analysis.

## Architecture

### MCP Tools

The server provides two tools:

**`geocode_city`**: Converts US city names to geographic coordinates
- Input: `city_name` (string)
- Output: `latitude`, `longitude`, `source`
- Uses hardcoded cache for 25 major cities, falls back to OpenStreetMap geocoding

**`get_weather`**: Fetches NOAA weather forecast for coordinates
- Input: `latitude`, `longitude` (float64)
- Output: `location`, `current_temp`, `unit`, `conditions`, `forecast[]`
- Queries the NOAA API (no authentication required)

### ADK Agent

The agent uses Google ADK's `llmagent` with:
- Configurable LLM provider (Gemini, Anthropic, OpenAI, Google via Fantasy)
- MCP toolset for accessing weather tools
- Instructions for weather comparison workflows

The agent automatically handles parallel tool calls when querying multiple cities.

### LLM Provider Architecture

**Native Gemini:**
- Uses `google.golang.org/adk/model/gemini` directly
- Returns `model.LLM` interface

**Fantasy Providers (Anthropic, OpenAI, Google):**
- Uses `charm.land/fantasy/providers/*` packages
- Wrapped with `github.com/robbyt/fantasy-adapters/adk`
- Adapter pattern converts `fantasy.LanguageModel` to `model.LLM`

## Project Structure

```
examples/adk_integration/
├── main.go           # ADK launcher and agent setup
├── cities.go         # City coordinate cache
├── weather_tools.go  # MCP tool implementations
├── main_test.go      # Integration tests
├── go.mod            # Dependencies (isolated from root)
└── README.md         # This file
```

## Testing

Run all tests:

```bash
go test .
```

Run without NOAA API integration tests:

```bash
go test -short .
```

Tests verify:
- City cache lookups work instantly
- Geocoding fallback functions correctly
- MCP tools are accessible through ADK toolset
- Both tools work together end-to-end
- Error handling for invalid inputs

## How It Works

1. **MCP Server**: `NewInMemoryPair()` creates an mcp-io server with weather tools and returns a client transport
2. **Server Start**: Server runs in a background goroutine using `handler.Run(ctx)`
3. **ADK Toolset**: `mcptoolset.New()` wraps the client transport, making MCP tools available to ADK
4. **LLM Model**: `createModel()` initializes the LLM based on `--provider` flag
   - Native path for Gemini
   - Fantasy adapter for Anthropic, OpenAI, Google
5. **ADK Agent**: `llmagent.New()` creates an agent with the MCP toolset and LLM model
6. **Launcher**: `full.NewLauncher().Execute()` handles the interaction loop (terminal or web)

## Notes

- NOAA API has no authentication requirement but is limited to continental US locations
- The city cache includes 25 major US cities for instant lookups
- Geocoding fallback requires network access to OpenStreetMap
- ADK handles parallel tool execution automatically when the LLM decides to query multiple cities
