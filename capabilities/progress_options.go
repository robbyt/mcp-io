package capabilities

// ProgressOption configures optional fields for progress notifications.
type ProgressOption func(*progressConfig)

// progressConfig holds optional progress notification parameters.
type progressConfig struct {
	token   any
	message string
	meta    map[string]any
}

// WithProgressToken associates the progress notification with a specific request
// for concurrent tracking. The token is an opaque identifier (int or string per
// MCP specification) that must be echoed back exactly as received from the client.
// Extract the token from RequestContext via GetProgressToken().
func WithProgressToken(token any) ProgressOption {
	return func(c *progressConfig) {
		c.token = token
	}
}

// WithProgressMessage adds a descriptive message to the progress notification.
// The message provides user-facing context about what's being processed.
func WithProgressMessage(message string) ProgressOption {
	return func(c *progressConfig) {
		c.message = message
	}
}

// WithProgressMeta adds custom metadata to the progress notification.
// Use this for application-specific information that extends the MCP protocol.
func WithProgressMeta(meta map[string]any) ProgressOption {
	return func(c *progressConfig) {
		c.meta = meta
	}
}
