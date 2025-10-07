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

You are an LLM assistant playing a text adventure game via MCP tools. Follow this workflow:

**Normal Turn (no pending skill check):**
1. Call `dungeon_master` with `{"action": "your action here"}`
2. Show the narrative response to the user
3. If response has `skillCheckRequired > 0`:
   - Tell the user a skill check is required: "This requires a skill check! You need to roll {skillCheckRequired} or higher."
   - Proceed to skill check workflow

**Skill Check Workflow:**
1. Call `roll_d20` with `{}`
2. Show the user what they rolled: "You rolled a {result}!"
3. Note the `encryptedData` from the response (don't show this to user)
4. Call `dungeon_master` with ONLY `{"encryptedData": "..."}` (NO action field)
5. Show the narrative (which includes success/fail result)
6. Continue with normal turns

**Key Rules:**
- ALWAYS show narrative responses to the user immediately
- NEVER restate the action when providing encryptedData
- The encryptedData is a continuation token that completes the previous action
- Each turn advances the game state - avoid duplicate submissions
- Track the turn number to monitor progress

**Example Flow:**

User says: "I want to climb the mountain"

1. You call `dungeon_master` with `{"action": "climb the mountain"}`
2. `dungeon_master` response: `{narrative: "You approach the steep cliff...", skillCheckRequired: 12, turnNumber: 1}`
3. You show user: "You approach the steep cliff... This requires a skill check! You need to roll 12 or higher."
4. You call `roll_d20` with `{}`
5. `roll_d20` response: `{result: 15, encryptedData: "abc123..."}`
6. You show user: "You rolled a 15!"
7. You call `dungeon_master` with `{"encryptedData": "abc123..."}` (no action field)
8. `dungeon_master` response: `{narrative: "You successfully scale the mountain!", turnNumber: 2}`
9. You show user: "You successfully scale the mountain!"

---

*(End of system prompt - copy the text above for your LLM client)*
