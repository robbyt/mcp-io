# Basic Elicitation Example

A simple MCP server that introduces elicitation capabilities - the ability for servers to request additional information from users through the MCP client during runtime. This example shows how session-aware tools can gather user input using typed schemas and handle user acceptance/rejection responses. For more complex multi-step workflows, see the http_multistep example.

## MCP Elicitation Flow

```
┌─────────────────┐    1. tools/list           ┌──────────────────┐
│   MCP Client    │ ─────────────────────────> │ cli_elicitation  │
│                 │ ◄───────────────────────── │                  │
│                 │    Available Tools         │                  │
└─────────────────┘    [                       └──────────────────┘
         │               "setup_application"
         │             ]
         │
         │  2. tools/call("setup_application", {})
         v
┌─────────────────┐                            ┌──────────────────┐
│   MCP Client    │ ─────────────────────────> │ cli_elicitation  │
│                 │ ◄───────────────────────── │                  │
│                 │   Elicitation Request      │                  │
└─────────────────┘                            └──────────────────┘
         │            {
         │              message: "Please provide your application configuration:",
         │              requestedSchema: { UserConfig schema }
         │            }
         │
         │  3. Server requests user input
         v
┌─────────────────┐                            ┌──────────────────┐
│     User        │ ◄─ Configuration Form ──── │   MCP Client     │
│   Interface     │ ── Filled Form ──────────> │                  │
└─────────────────┘                            └──────────────────┘
         │                                               │
         │  4. Client sends elicitation response         │
         v                                               v
┌─────────────────┐                            ┌──────────────────┐
│   MCP Client    │ ─────────────────────────> │ cli_elicitation  │
│                 │ ◄───────────────────────── │                  │
│                 │   Final Tool Result        │                  │
└─────────────────┘                            └──────────────────┘
         │            {
         │              status: "configured",
         │              config: { user data }
         │            }
         │
         │  5. Client uses result in LLM context
         v
┌─────────────────┐
│ Application     │ ──> "Successfully configured application with
│ Response        │     name: John, environment: production, port: 8080"
└─────────────────┘
```

**Key Point**: MCP elicitation enables servers to dynamically request information from users during tool execution. Unlike tools (which are called by clients) or resources (which provide static data), elicitation allows servers to initiate interactive dialogs with users. The client maintains full control over user interaction and can decline, cancel, or modify elicitation requests.

**Security Note**: This example includes warnings about sensitive data - elicitation should never be used for passwords, API keys, or other secrets in production environments. It's designed for configuration data, preferences, and other non-sensitive user input.

For more details on MCP elicitation, see the [official MCP specification](https://modelcontextprotocol.io/specification/2025-06-18/client/elicitation).

## Features Demonstrated

**Typed Schemas**: Uses `ElicitTyped[T]()` for automatic JSON schema generation from Go structs, similar to typed tools.

**Response Handling**: Shows proper handling of "accept", "decline", and "cancel" actions from users.

**Interactive Prompts**: The `interactive_document` prompt shows how elicitation can be used within prompt generation to gather requirements dynamically.

**Session-Aware Architecture**: Uses `WithSessionTool` and `WithSessionPrompt` to provide elicitation capabilities to handlers.

## Running

```bash
make build-cli-elicitation
./bin/cli-elicitation
```

## Testing the Elicitation Server

**Important**: The `mcp` CLI tool does not support elicitation features. Elicitation requires a session-aware MCP client that can handle interactive back-and-forth communication. The CLI tool only supports stateless, one-shot operations.

### What you CAN test with mcp CLI:
```bash
# List available tools (works)
mcp tools ./bin/cli-elicitation

# List available prompts (works)
mcp prompts ./bin/cli-elicitation

# Get a prompt template (works)
mcp get-prompt interactive_document --params '{"document_type":"proposal"}' ./bin/cli-elicitation
```

### What you CANNOT test with mcp CLI:
```bash
# This will fail - elicitation requires session support
mcp call setup_application --params '{}' ./bin/cli-elicitation
```

### To test elicitation features:
- Use an MCP client that supports elicitation (like Claude Desktop)
- Write integration tests with mock elicitation handlers
- Use the HTTP version with a custom client that handles elicitation requests

## Example Schema

The example uses a simple schema for basic application configuration:

```go
type UserConfig struct {
    Name        string `json:"name" jsonschema:"description=Your name"`
    Email       string `json:"email" jsonschema:"format=email,description=Your email address"`
    Environment string `json:"environment" jsonschema:"description=Target environment,enum=development,enum=staging,enum=production"`
}
```

This schema is automatically converted to JSON schema that MCP clients use to create appropriate user interfaces with dropdowns for enums, format validation, and helpful field descriptions.