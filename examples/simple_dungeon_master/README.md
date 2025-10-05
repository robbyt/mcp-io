# Simple Dungeon Master Example

A stateful MCP server that uses the MCP sampling capability to create an interactive text adventure game.

Demonstrates:
- **MCP sampling capability** to delegate narrative generation to the client's LLM
- **Multi-step LLM orchestration** (generate narrative, then conditionally summarize history)
- **Stateful conversation management** with persistent memory across multiple tool calls
- **Interactive gameplay mechanics** using D20 skill checks with roll validation
- **Context-aware prompting** by including full game history in every LLM request
- **Automatic context window management** via intelligent summarization when history grows
- **MCP logging integration** for debugging LLM failures

## Requirements

This example requires an MCP client with **sampling support**.

See [MCP Clients](https://modelcontextprotocol.io/clients) for clients that support the sampling capability.

## Using with oterm

[oterm](https://github.com/ggozad/oterm) is a terminal-based MCP client that supports sampling and can use local LLMs.

First, build the example binary for this MCP server:

```bash
make build-simple-dungeon-master
```

The oterm configuration file is located at:
- **Linux**: `~/.local/share/oterm/config.json`
- **macOS**: `~/Library/Application Support/oterm/config.json`
- **Windows**: `C:/Users/<USER>/AppData/Roaming/oterm/config.json`

Add the MCP server configuration:

```json
{
  "mcpServers": {
    "dungeon-master": {
      "command": "/full/path/to/mcp-io/bin/simple-dungeon-master"
    }
  }
}
```

Replace `/full/path/to/mcp-io` with your actual project path.

## Playing the Game

The server exposes three tools:

### Available Tools

**`dungeon_master`** - Submit player actions and receive generated narrative
- All player actions must be sent to this tool (e.g., "look around", "open door", "attack goblin")
- Returns narrative describing what happens
- May indicate a skill check is required for the next action

**`roll_d20`** - Roll a 20-sided die for skill checks
- Called when the narrative indicates a skill check is required
- Roll result is validated against the difficulty threshold
- Must be called before submitting your next action if a check is pending

**`debug_gameState`** - Inspect internal game state
- Returns current turn history, summary, pending skill checks, and last roll
- Useful for debugging or understanding game state

### How to Play

All player actions go to the `dungeon_master` tool:
```
Use the dungeon_master MCP tool with action: "I look around"
```

Or with a simple prompt:
```
action: look around
```

**Skill Checks**: When the narrative requires a skill check, you must call `roll_d20` before your next action. The game will validate your roll and incorporate the success/failure into the narrative.

The server maintains a persistent adventure with intelligent memory management:
- Remembers all previous actions and their consequences
- Automatically condenses long histories while preserving key events
- Enforces narrative consistency - the LLM validates actions against established context
- Prevents contradictions (e.g., can't open a door if you don't have the key)

## How It Works

1. **Skill Check Validation**: If a previous turn required a skill check, validates the D20 roll before proceeding
2. **State Management**: Persistent game history with automatic memory management
3. **LLM Call 1**: Generate narrative based on player action + full historical context
4. **Skill Check Assignment**: Randomly determines if the next action requires a D20 roll (configurable probability)
5. **LLM Call 2** (conditional): Intelligently summarize older events when history grows long
6. **Context Preservation**: Every prompt includes both condensed summary and recent events for continuity
