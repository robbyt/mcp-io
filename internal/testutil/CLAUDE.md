# Testing Utilities (internal/testutil)

This package provides testing utilities for MCP integration tests.

## ExampleTestSuite

Base struct for testing MCP examples with both in-memory and HTTP transports:

```go
type ExampleTestSuite struct {
    suite.Suite
    ProjectRoot string
    ExampleName string
}
```

### Key Methods

**`WithMCPSession(handler, testFunc)`**: Tests MCP servers using in-memory transport
- Creates client/server in-memory connections
- Automatically handles cleanup
- Use for fast integration tests

**`ConnectInMemory(t, handler)`**: Helper that returns a connected MCP session
- Simpler than WithMCPSession for direct session access
- Returns `*mcp.ClientSession`

### Usage Pattern

```go
type MyTestSuite struct {
    testutil.ExampleTestSuite
}

func (s *MyTestSuite) SetupSuite() {
    s.ProjectRoot = filepath.Join(...)
    s.ExampleName = "my_example"
    s.ExampleTestSuite.SetupSuite()
}

func (s *MyTestSuite) TestFeature() {
    handler, err := mcpio.NewHandler(...)
    s.Require().NoError(err)

    session := testutil.ConnectInMemory(s.T(), handler)
    result, err := session.CallTool(...)
}
```

## Mock Utilities

**`MockSession`**: Mock MCP session for unit tests
- Use when you don't need real protocol interactions
- Lightweight, fast for unit tests
- See `mocks.go` for available methods

## Testing Best Practices

### Use t.TempDir() for File Operations
**CRITICAL**: NEVER use shell commands like `rm -rf` in tests. Always use `t.TempDir()`:

```go
func TestSomething(t *testing.T) {
    tmpDir := t.TempDir() // Automatically cleaned up
    // Use tmpDir...
}
```

### Avoid Manual JSON Handling
Use MCP SDK's `StructuredContent` instead of manual marshaling:

```go
// Good
result := &mcp.TextContent{Text: "response"}

// Avoid
jsonBytes, _ := json.Marshal(map[string]any{"text": "response"})
```

### JSON Number Gotcha
When unmarshaling JSON to `map[string]any`, numbers become `float64`:

```go
data := map[string]any{}
json.Unmarshal([]byte(`{"count": 5}`), &data)
// data["count"] is float64(5), not int(5)
```

## HTTP Transport Testing

For testing HTTP-specific features (headers, auth), use `httptest`:

```go
func TestHTTPFeature(t *testing.T) {
    handler, _ := mcpio.NewHandler(
        mcpio.WithHTTPTransport(
            mcpwrapper.WithStateless(), // Simpler for tests
        ),
        mcpio.WithTool(...),
    )

    server := httptest.NewServer(handler)
    defer server.Close()

    // Use MCP client with StreamableClientTransport
    transport := &mcp.StreamableClientTransport{
        Endpoint: server.URL,
    }
    // ...
}
```
