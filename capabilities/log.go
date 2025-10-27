package capabilities

import (
	"context"
	"maps"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Log sends a structured log message to the client.
// The message will only be sent if it meets the client's minimum log level.
func (s *sessionCapability) Log(ctx context.Context, level LogLevel, message string, data map[string]any) error {
	logData := make(map[string]any)
	if data != nil {
		maps.Copy(logData, data)
	}
	logData["message"] = message

	params := &mcp.LoggingMessageParams{
		Level: mcp.LoggingLevel(level),
		Data:  logData,
	}
	return s.session.Log(ctx, params)
}
