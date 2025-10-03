package capabilities

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

// Root represents a client workspace root (directory or file).
type Root struct {
	URI  string
	Name string
}
