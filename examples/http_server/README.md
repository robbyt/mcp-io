# HTTP Server Example

An HTTP-based MCP server that exposes calculator and echo tools through web transport instead of stdio. This example demonstrates how to integrate MCP functionality into web applications by mounting the handler at `/mcp` and adding additional endpoints like `/health` for monitoring. The server listens on port 8080 and provides arithmetic operations (add, subtract, multiply, divide) with division-by-zero error handling, alongside a simple message echo service. This pattern enables MCP servers to be deployed as web services, accessed through HTTP by MCP clients that support web transport, and integrated into larger web application architectures.

## Running

```bash
make build-http-server
./bin/http-server
```

The server will start on `http://localhost:8080` with MCP endpoint at `/mcp` and health check at `/health`.