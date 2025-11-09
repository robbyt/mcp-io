// Package sampling provides types and options for LLM sampling via the MCP protocol.
package sampling

// Message represents a message for LLM sampling.
type Message struct {
	Role    string
	Content string
}

// MessageResult represents the result of an LLM sampling request.
type MessageResult struct {
	Role    string
	Content TextContent
}

// TextContent represents text content in a message.
type TextContent struct {
	Text string
}
