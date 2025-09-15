# CLI Tool Example

An MCP tool server that provides multiple text processing tools for string manipulation including uppercase conversion, lowercase conversion, text reversal, and word/character/line counting. This example demonstrates how to organize multiple related tools within a single server, showing different input/output type patterns and error handling with `ValidationError` for user-facing error messages. The server uses stdio transport for command-line integration, making it useful for building text processing utilities that can be accessed by language models. This pattern can be extended to create specialized toolkits for document processing, data transformation, or content analysis tasks.

## MCP Tool Flow

```
┌─────────────────┐    1. tools/list       ┌──────────────────┐
│   LLM Client    │ ─────────────────────> │  cli_tool        │
│   Application   │                        │     Server       │
│                 │ <───────────────────── │                  │
│                 │    Available Tools     │                  │
└─────────────────┘    [                   └──────────────────┘
         │                "to_upper",
         │                "to_lower",
         │                "reverse",
         │                "count"
         │              ]
         │
         │ 2. tools/call("to_upper", {text: "hello"})
         │
         v
┌─────────────────┐                        ┌──────────────────┐
│   LLM Client    │ ─────────────────────> │  cli_tool        │
│   Application   │                        │     Server       │
│                 │ <───────────────────── │                  │
│                 │   Tool Result          │                  │
└─────────────────┘                        └──────────────────┘
         │            {
         │              result: "HELLO"
         │            }
         │
         │ 3. LLM incorporates result into response
         v
┌─────────────────┐
│ LLM Response    │ ──> "I've converted 'hello' to uppercase: HELLO"
│ with Tool Data  │
└─────────────────┘
```

**Key Point**: MCP tools perform actions on behalf of language models (unlike resources that provide data or prompts that provide templates). Tools are model-controlled, meaning the AI can discover and invoke them automatically, but there should always be human oversight. This example shows both typed tools with automatic schema generation (`WithTool`) and demonstrates error handling patterns for robust tool execution.

**Tool Types**: This example uses `WithTool` for type-safe tools where Go structs automatically generate JSON schemas. For more complex scenarios, `WithRawTool` allows manual schema definition and direct JSON handling.

For more details on MCP tools, see the [official MCP documentation](https://modelcontextprotocol.io/docs/concepts/tools).

## Running

```bash
make build-cli-tool
./bin/cli-tool
```