# CLI Combined Example

A comprehensive MCP server that demonstrates all three core MCP primitives working together: tools for text processing, resources for document templates, and prompts for content generation. This example showcases how to build a complete document processing system that can format and analyze text, provide template resources, and generate writing prompts for different document types. This pattern demonstrates the full power of MCP's architecture and how mcp-io's functional options make it simple to combine multiple capabilities in a single server.

## MCP Combined Flow

```
┌─────────────────┐    1. Capabilities discovery     ┌──────────────────┐
│   MCP Client    │ ───────────────────────────────> │  cli_combined    │
│                 │ ◄─────────────────────────────── │                  │
│                 │   Tools, Resources, Prompts      │                  │
└─────────────────┘                                  └──────────────────┘
         │ 2a. tools/list -> ["format_text", "analyze_text"]
         │ 2b. resources/list -> ["doc://templates/", "doc://samples/"]
         │ 2c. prompts/list -> ["document_writer"]
         │
         │  3. Workflow example:
         ▼
┌─────────────────┐                                   ┌──────────────────┐
│   MCP Client    │ ── resources/read(doc://email) ─> │  cli_combined    │
│                 │ ◄─ email template ─────────────── │                  │
│                 │                                   │                  │
│                 │ ── tools/call(analyze_text) ────> │                  │
│                 │ ◄─ {wordCount: 150, ...} ──────── │                  │
│                 │                                   │                  │
│                 │ ── prompts/get(doc_writer) ─────> │                  │
│                 │ ◄─ writing prompt ─────────────── │                  │
└─────────────────┘                                   └──────────────────┘
         │
         │  4. Client combines all data in LLM context
         v
┌─────────────────┐
│ Application     │ ──> "Using email template, analyzed 150 words,
│ Response        │     generated professional email about quarterly results"
└─────────────────┘
```

## Real-World Usage Pattern

This example demonstrates a common pattern where MCP primitives work together:

1. **Resources** provide structured data (templates, samples, configurations)
2. **Tools** perform computations and transformations (analysis, formatting, validation)
3. **Prompts** generate contextual AI interactions (content generation, guided workflows)

A typical workflow might be:
- LLM reads document template from resources
- User provides content, LLM uses tools to analyze and format it
- LLM uses prompts to generate additional content based on the analysis

## Key Features Demonstrated

**Multiple Tool Types**: Shows both simple transformations (`format_text`) and complex analysis (`analyze_text`) with structured output schemas.

**Dynamic Resources**: URI-based resource serving with different content types (templates as text/plain, samples as application/json).

**Parameterized Prompts**: Context-aware prompt generation that adapts system and user messages based on document type, topic, and tone.

**Unified Handler**: Uses `mcpio.NewHandler()` to combine all three primitives with mcp-io's clean functional options API, contrasting with the verbose setup required by the raw MCP SDK.

**Key Point**: This demonstrates MCP's full ecosystem where tools perform actions, resources provide data, and prompts enable AI generation - all working together seamlessly. Unlike the raw MCP SDK which requires complex boilerplate for each primitive, mcp-io's functional options make it simple to build comprehensive servers that leverage all of MCP's capabilities.

For more details on MCP architecture, see the [official MCP documentation](https://modelcontextprotocol.io/docs/concepts).

## Running

```bash
make build-cli-combined
./bin/cli-combined
```

## Testing the Combined Server

```bash
# List all capabilities
mcp tools ./bin/cli-combined
mcp resources ./bin/cli-combined
mcp prompts ./bin/cli-combined

# Use tools
mcp call analyze_text --params '{"text":"The quick brown fox jumps over the lazy dog."}' ./bin/cli-combined

# Access resources (may have limitations with current mcp CLI)
mcp read-resource "doc://templates/email" ./bin/cli-combined

# Generate prompts (may fail due to "system" role limitations)
mcp get-prompt document_writer --params '{"document_type":"email","topic":"project update","tone":"professional"}' ./bin/cli-combined
```

**Note**: Some commands may have limitations with the current mcp CLI tool version, particularly resource reading and prompts using "system" role messages.