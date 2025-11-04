# SDK Examples

Examples that translate the [upstream MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
examples to mcp-io's API. Each example demonstrates the differences between the SDK's direct API
and mcp-io's functional options pattern. All of these examples use in-memory transport for
demonstration, but production code should use `handler.ServeHTTP()` or `handler.ServeStdio()`.

## Examples

Each example includes [testable example functions](https://go.dev/blog/examples), that can be run
with `go test`.

### [basic](./basic/)
Tool registration and invocation using functional options.

Original: https://github.com/modelcontextprotocol/go-sdk/tree/main/examples/basic

### [elicitation](./elicitation/)
Server-side elicitation with automatic schema generation from Go structs.

Original: https://github.com/modelcontextprotocol/go-sdk/tree/main/examples/elicitation

### [resources](./resources/)
Static resources and dynamic resource templates with functional options.

Original: https://github.com/modelcontextprotocol/go-sdk/blob/main/mcp/server_example_test.go (Example_resources)
