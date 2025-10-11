# Simple Text Processor

A minimal MCP server demonstrating the basic setup with a single tool. This example showcases both stdio and HTTP transports using the same handler.

## Features

- Single tool: `to_upper` - converts text to uppercase
- Dual transport support: stdio (default) or HTTP (with `-listen` flag)
- Minimal code demonstrating the core mcp-io API

## Running

### Stdio Transport (default)

```bash
make build-cli-simple
./bin/cli-simple
```

### HTTP Transport

```bash
./bin/cli-simple -listen :8080
```

The server will start on `http://localhost:8080/mcp`

## Testing

### Stdio Mode

```bash
# List available tools
mcp tools ./bin/cli-simple

# Call the to_upper tool
mcp call to_upper --params '{"text":"hello world"}' ./bin/cli-simple
```

### HTTP Mode

```bash
# Start the server in another terminal
./bin/cli-simple -listen :8080

# List available tools
mcp tools http://localhost:8080/mcp

# Call the to_upper tool
mcp call to_upper --params '{"text":"hello world"}' http://localhost:8080/mcp
```

## Transport Selection

This example demonstrates how a single handler can serve multiple transports:

```go
handler, err := mcpio.NewHandler(
    mcpio.WithName("simple-text-processor"),
    mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),
)

// Choose transport based on CLI flag
if *listen != "" {
    http.Handle("/mcp", handler)
    log.Fatal(http.ListenAndServe(*listen, nil))
} else {
    handler.ServeStdio(context.Background(), os.Stdin, os.Stdout)
}
```
