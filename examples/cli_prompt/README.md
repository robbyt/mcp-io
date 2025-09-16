# CLI Prompt Example

An MCP prompt server that generates conversation templates for language model interactions. This example shows how to create dynamic prompts that accept arguments and construct multi-message conversations with system and user roles. The prompt handler takes a `name` parameter and builds a structured conversation template that instructs the language model to generate greetings. This pattern is useful for building libraries of reusable prompt templates, conversation starters, or structured interaction patterns that can be parameterized and invoked by MCP clients.

## MCP Prompt Flow

```
┌─────────────────┐   1. prompts/list      ┌──────────────────┐
│   MCP Client    │ ─────────────────────> │   cli_prompt     │
│                 │ ◄───────────────────── │                  │
│                 │      ["greeter"]       │                  │
└─────────────────┘                        └──────────────────┘
         │
         │  2. prompts/get("greeter", {name: "World"})
         v
┌─────────────────┐                        ┌──────────────────┐
│   MCP Client    │ ─────────────────────> │   cli_prompt     │
│                 │ ◄───────────────────── │                  │
│                 │   Message Template     │                  │
└─────────────────┘                        └──────────────────┘
         │            [
         │              {role: "system", content: "You are..."},
         │              {role: "user", content: "Create greeting for World"}
         │            ]
         │
         │  3. Client uses template with LLM API
         v
┌─────────────────┐
│ Application     │ ───> "Hello World! How are you today?"
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

# Get prompt template (Note: May fail with "unsupported role: system"
# due to mcp CLI limitations with system role messages)
mcp get-prompt greeter --params '{"name":"Alice"}' ./bin/cli-prompt
```

**Note**: The mcp CLI tool may not fully support prompts that use "system" role messages. For full testing, use an MCP client that supports all message roles.