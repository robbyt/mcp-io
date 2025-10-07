# Simple Dungeon Master Example

A stateful MCP server that uses the MCP sampling capability to create an interactive text adventure game.

Demonstrates:
- **MCP sampling capability** to delegate narrative generation to the client's LLM
- **Multi-step LLM orchestration** (generate narrative, then conditionally summarize history)
- **Stateful conversation management** with persistent memory across multiple tool calls
- **Interactive gameplay mechanics** using D20 skill checks with roll validation
- **Context-aware prompting** by including full game history in every LLM request
- **Context window management** via summarization when history grows
- **MCP logging integration** for debugging LLM failures

## Architecture

1. **Turn-based gameplay**: D20 skill checks with roll validation and turn history tracking
2. **LLM narrative generation**: Generates narrative based on player action and full game context
3. **History summarization**: Condenses older events to manage context window growth

## Requirements

This example requires an MCP client with **sampling support**.

See [MCP Clients](https://modelcontextprotocol.io/clients) for clients that support the sampling capability.

## Setup

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

## Tools

The server exposes three tools:

- **`dungeon_master`** - Submit actions, receive narrative (may require skill check)
- **`roll_d20`** - Roll D20 when skill check required
- **`debug_gameState`** - Inspect turn history, summary, and pending checks (for debugging)

## System Prompt for LLM Clients

---

You are playing a text adventure game using two MCP tools:

- **dungeon_master** - Submit player actions, receive narrative
- **roll_d20** - Roll dice when a skill check is required

**Basic workflow:**
1. Call `dungeon_master` with an action
2. If the response has narrative text AND `skillCheckRequired > 0`:
   - Show the narrative to the user
   - For your next turn: call `roll_d20`, then call `dungeon_master` with encryptedData only (no action)
3. If the response has EMPTY narrative AND `skillCheckRequired > 0`:
   - The skill check is for the action you just sent
   - Call `roll_d20`, then call `dungeon_master` with ONLY encryptedData (no action field)
   - Do not resubmit the action

The `encryptedData` completes the pending action - never include a new action when passing it.

---

*(End of system prompt - copy the text above for your LLM client)*
