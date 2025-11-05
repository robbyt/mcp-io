# SDK Examples

[Testable example functions](https://go.dev/blog/examples) that translate examples from the [upstream MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) to mcp-io's API. Each example is self-contained with all types, handlers, and setup code inline, following the pattern in the SDK's `mcp/server_example_test.go`.

## Examples

All examples use in-memory transport for demonstration. In production, use `handler.ServeHTTP()` or `handler.ServeStdio()`.

### [Example_tools](./tools_test.go)

Demonstrates basic tool registration and invocation using mcp-io's functional options API.

Key differences from SDK:
- Type-safe input/output with automatic schema generation
- Functional options instead of direct server initialization
- Simplified function signatures

Compare with SDK: https://github.com/modelcontextprotocol/go-sdk/blob/main/mcp/server_example_test.go (Example_tools pattern)

### [Example_elicitation](./elicitation_test.go)

Demonstrates server-side elicitation for gathering structured data from clients.

Key differences from SDK:
- `ElicitTyped[T]` with automatic schema generation from Go structs
- Type-safe `DecodeContent()` instead of untyped map access
- Helper methods: `IsAccepted()`, `IsDeclined()`, `IsCancelled()`

Compare with SDK: https://github.com/modelcontextprotocol/go-sdk/blob/main/mcp/server_example_test.go (Example_prompts for elicitation pattern)

### [Example_resources](./resources_test.go)

Demonstrates resource registration with both static resources and dynamic templates.

Key differences from SDK:
- Functional options for resource configuration
- Simplified `ResourceFunc` signature
- `RequestContext.GetIdentifier()` for URI access

Compare with SDK: https://github.com/modelcontextprotocol/go-sdk/blob/main/mcp/server_example_test.go (Example_resources)

## Running the Examples

These examples appear in `go doc` output and can be tested:

```bash
# View example documentation
go doc -all github.com/robbyt/mcp-io/examples/sdk

# Run all examples as tests (verifies output)
go test -v ./examples/sdk

# Run a specific example
go test -v ./examples/sdk -run Example_tools
```

## Comparison with CLI Examples

The `examples/cli_*` directories contain runnable programs with HTTP/stdio transports and command-line flags. These SDK examples focus on API comparison and documentation.
