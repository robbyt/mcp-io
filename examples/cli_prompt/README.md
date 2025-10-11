# CLI Prompt Example

An MCP prompt server that demonstrates both map-based and typed prompts for generating conversation templates. This example showcases two different prompt patterns:

1. **Map-based prompts** (`greeter`) - Traditional prompts that accept string arguments and handle type casting manually
2. **Typed prompts** (`document_writer`) - Type-safe prompts with automatic schema generation from Go structs

## Handler Setup

This example uses `NewHandler()` to create an MCP server with two prompts:
```go
handler, err := mcpio.NewHandler(
    mcpio.WithName("prompt-server"),
    mcpio.WithPrompt("greeter", "Generates a friendly greeting", greeterPrompt),
    mcpio.WithTypedPrompt("document_writer", "Generates writing prompts with typed arguments", documentPrompt),
)
```

The `NewHandler()` constructor supports any combination of MCP resources (tools, prompts, resources, resource templates). This example demonstrates a prompt-only server, but you can add tools or resources using additional `With*` options

## Map-based vs Typed Prompts

### Map-based Prompts (`WithPrompt`)
- **Pros**: Simple, flexible, works with any argument structure
- **Cons**: Requires manual type casting, no compile-time validation, no automatic schema generation
- **Best for**: Simple prompts with basic string arguments

### Typed Prompts (`WithTypedPrompt`)
- **Pros**: Compile-time type safety, automatic schema generation, no type casting needed
- **Cons**: Must use string fields due to MCP protocol limitation (arguments are `map[string]string`)
- **Best for**: Complex prompts with multiple parameters and validation requirements

**Note**: Due to MCP protocol limitations, all arguments are passed as strings, so even typed prompts should use `string` fields and handle conversion internally if needed.

## Available Prompts

This example provides two prompts:

### 1. `greeter` (Map-based)
- **Description**: Generates a friendly greeting
- **Parameters**: `name` (string, optional) - Name to greet
- **Example**: `{name: "Alice"}`

### 2. `document_writer` (Typed)
- **Description**: Generates writing prompts with typed arguments and schema validation
- **Parameters**:
  - `documentType` (string, required) - Type of document to generate
  - `topic` (string, required) - Main topic or subject
  - `tone` (string, required) - Writing tone (formal, casual, etc)
  - `length` (string, optional) - Target length in words (e.g. "500", "1000")
- **Example**: `{documentType: "blog post", topic: "AI", tone: "conversational", length: "500"}`

## MCP Prompt Flow

```
┌─────────────────┐   1. prompts/list      ┌──────────────────┐
│   MCP Client    │ ─────────────────────> │   cli_prompt     │
│                 │ ◄───────────────────── │                  │
│                 │  ["greeter",           │                  │
│                 │   "document_writer"]   │                  │
└─────────────────┘                        └──────────────────┘
         │
         │  2. prompts/get("document_writer", {...})
         v
┌─────────────────┐                        ┌──────────────────┐
│   MCP Client    │ ─────────────────────> │   cli_prompt     │
│                 │ ◄───────────────────── │                  │
│                 │   Message Template     │                  │
└─────────────────┘                        └──────────────────┘
         │            [
         │              {role: "system", content: "You are an expert writer..."},
         │              {role: "user", content: "Write a blog post about AI..."}
         │            ]
         │
         │  3. Client uses template with LLM API
         v
┌─────────────────┐
│ Application     │ ───> Generated content based on prompt
│ Response        │
└─────────────────┘
```

**Key Point**: This MCP server provides prompt templates, not LLM responses. The client receives the structured conversation template and then makes its own call to an LLM API (OpenAI, Anthropic, local model) using the template as input.

For more details on MCP prompts, see the [official MCP documentation](https://modelcontextprotocol.io/docs/concepts/prompts).

## Running

```bash
make build-cli-prompt
./bin/cli-prompt
```

## Testing with MCP CLI

```bash
# List available prompts
mcp prompts ./bin/cli-prompt

# Test map-based prompt
mcp get-prompt greeter --params '{"name":"Alice"}' ./bin/cli-prompt

# Test typed prompt
mcp get-prompt document_writer --params '{"documentType":"blog post","topic":"AI","tone":"conversational","length":"500"}' ./bin/cli-prompt
```

**Note**: The mcp CLI tool may not fully support prompts that use "system" role messages. For full testing, use an MCP client that supports all message roles.