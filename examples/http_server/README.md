# HTTP Server Example

An HTTP-based MCP server that exposes calculator and echo tools through web transport instead of stdio. This example demonstrates how to integrate MCP functionality into web applications by mounting the handler at `/mcp` and adding additional endpoints like `/health` for monitoring. The server listens on port 8080 and provides arithmetic operations (add, subtract, multiply, divide) with division-by-zero error handling, alongside a simple message echo service. This pattern enables MCP servers to be deployed as web services, accessed through HTTP by MCP clients that support web transport, and integrated into larger web application architectures.

## MCP HTTP Transport Flow

```
┌─────────────────┐    1. tools/list           ┌──────────────────┐
│   MCP Client    │ ─────────────────────────> │ server:8080/mcp  │
│                 │ ◄───────────────────────── │                  │
│                 │    Available Tools         │                  │
└─────────────────┘    [                       └──────────────────┘
         │                "add",
         │                "subtract",
         │                "multiply",
         │                "divide",
         │                "echo"
         │              ]
         │
         │  2. tools/call("add", {a: 5, b: 3})
         v
┌─────────────────┐                            ┌──────────────────┐
│   MCP Client    │ ─────────────────────────> │ server:8080/mcp  │
│                 │ ◄───────────────────────── │                  │
│                 │   Tool Result              │                  │
└─────────────────┘                            └──────────────────┘
         │            { "result": 8 }
         │
         │  3. Client uses result in LLM context.
         v
┌─────────────────┐
│ Application     │ ──> Calculator: 5 + 3 = 8
│ Response        │
└─────────────────┘
```

**Transport Details**: This server uses MCP's Streamable HTTP transport, which supports JSON-RPC communication over HTTP POST requests. The transport handles both request-response patterns and supports Server-Sent Events (SSE) for streaming responses. The server binds to localhost for security and validates requests according to MCP transport specifications.

**Key Point**: MCP HTTP transport enables web-based integration where MCP servers can be deployed as web services, accessed by web clients, and integrated into larger application architectures. This contrasts with stdio transport which is designed for command-line integration.

For more details on MCP HTTP transport, see the [official MCP documentation](https://modelcontextprotocol.io/docs/concepts/transports).

## Running

```bash
make build-http-server
./bin/http-server
```

The server will start on `http://localhost:8080` with MCP endpoint at `/mcp` and health check at `/health`.

## Testing the HTTP Server

```bash
# Check server health (works)
curl http://localhost:8080/health

# MCP CLI does not work with HTTP servers
# This will fail with 405 Method Not Allowed:
# mcp tools http://localhost:8080/mcp

# Direct JSON-RPC calls require session initialization
# These are examples but may not work without proper MCP session setup:

# List available tools (requires session)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "method": "tools/list", "id": 1}'

# Call a tool (requires session)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "add", "arguments": {"a": 5, "b": 3}}, "id": 2}'
```

**Note**: HTTP MCP servers require proper session initialization and are designed for MCP clients, not direct curl testing. Use the health endpoint to verify the server is running.