# CLI Resource Example

An MCP resource server that exposes data through URI-based access patterns. This example implements a key-value store accessible via `res://kv/{key}` URIs, demonstrating how to handle resource requests and return content with appropriate MIME types. The resource handler extracts the key from the URI path and returns stored data as text/plain content, showing the basic pattern for building data access servers. This approach can be extended to expose files, database records, API responses, or any other data sources that need to be accessible to language models through the MCP resource protocol.

## MCP Resource Flow

```
┌─────────────────┐    1. resources/list       ┌──────────────────┐
│   MCP Client    │ ─────────────────────────> │  cli_resource    │
│                 │ ◄───────────────────────── │                  │
│                 │    Available Resources     │                  │
└─────────────────┘    [                       └──────────────────┘
         │                {
         │                  uri: "res://kv/",
         │                  name: "Key-Value Store"
         │                }
         │              ]
         │
         │  2. resources/read("res://kv/greeting")
         v
┌─────────────────┐                            ┌──────────────────┐
│   MCP Client    │ ─────────────────────────> │  cli_resource    │
│                 │ ◄───────────────────────── │                  │
│                 │   Resource Content         │                  │
└─────────────────┘                            └──────────────────┘
         │            {
         │              uri: "res://kv/greeting",
         │              mimeType: "text/plain",
         │              text: "Hello, World!"
         │            }
         │
         │  3. Client uses resource in LLM context
         v
┌─────────────────┐
│ Application     │ ──> "Based on greeting resource: Hello, World!"
│ Response        │
└─────────────────┘
```

**Key Point**: MCP resources provide contextual data that clients can incorporate into language model conversations. Unlike tools (which perform actions) or prompts (which provide templates), resources expose existing data for the LLM to reference and use in its responses.

**URI Pattern**: This example uses `WithResource` to register a static URI prefix (`res://kv/`), but real applications often use `WithResourceTemplate` for dynamic URI patterns like `file:///{path}` or `db://table/{id}`.

For more details on MCP resources, see the [official MCP documentation](https://modelcontextprotocol.io/docs/concepts/resources).

## Running

```bash
make build-cli-resource
./bin/cli-resource
```

## Testing with MCP CLI

```bash
# List available resources (resource templates may not show in list)
mcp resources ./bin/cli_resource

# Read specific resources
mcp read-resource "res://kv/greeting" ./bin/cli_resource
mcp read-resource "res://kv/farewell" ./bin/cli_resource

# Test non-existent key (returns error)
mcp read-resource "res://kv/nonexistent" ./bin/cli_resource
```

**Note**: The example uses `WithResourceTemplate` to support dynamic resource access via `res://kv/{key}` URIs. Available keys are "greeting" and "farewell".