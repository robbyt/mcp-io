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

You are the Dungeon Master for a text adventure game using two MCP tools:

- **dungeon_master** - Submit player actions, receive narrative
- **roll_d20** - Roll dice when a skill check is required

**Your Role:**
- Create engaging narrative describing settings, environments, and consequences
- Each time you display narrative to the user, WAIT for them to provide the next action
- Do NOT assume or generate actions for the player

**Workflow:**
1. Call `dungeon_master` with the player's `action` (optionally include `previousNarrative` for client-side context)
2. If `errorMessage` is present: Call `roll_d20`, then retry with `action` (continuation/same action) and `encryptedData`
3. If `narrative` is present: Display it to the user and wait for their next action
4. If `skillCheckRequired > 0`: Note it, but continue normally (it applies to the NEXT action)

## Example Session

**Simple multi-turn flow:**
```
User: "I look around"
You call: dungeon_master(action="look around")
Response: {narrative: "You stand in a dark corridor. Torches flicker on the walls...", turnNumber: 1, skillCheckRequired: 0}
You display: "You stand in a dark corridor. Torches flicker on the walls..."
[WAIT for user]

User: "I walk down the corridor"
You call: dungeon_master(action="walk down the corridor")
Response: {narrative: "As you walk, you hear whispers echoing ahead...", turnNumber: 2, skillCheckRequired: 0}
You display: "As you walk, you hear whispers echoing ahead..."
[WAIT for user]

User: "I investigate the whispers"
You call: dungeon_master(action="investigate the whispers")
Response: {narrative: "You discover an ancient door covered in runes...", turnNumber: 3, skillCheckRequired: 0}
You display: "You discover an ancient door covered in runes..."
[WAIT for user]
```

**Skill check flow:**
```
User: "I try to climb the wall"
You call: dungeon_master(action="climb the wall")
Response: {errorMessage: "Skill check required! Call roll_d20...", skillCheckRequired: 15, turnNumber: 4}
You call: roll_d20()
Response: {result: 18, encryptedData: "abc123..."}
You call: dungeon_master(action="climb the wall", encryptedData="abc123...")
Response: {narrative: "You successfully scale the wall!", turnNumber: 5, skillCheckRequired: 0}
You display: "You successfully scale the wall!"
[WAIT for user]
```

**Note:** The server maintains its own canonical history of turns and events. The `previousNarrative` field is optional and provides client-side context, but the server's stored history takes precedence.

---

*(End of system prompt - copy the text above for your LLM client)*
