package capabilities

import (
	"context"
	"maps"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// LogLevel represents logging severity levels.
type LogLevel string

const (
	LogLevelDebug     LogLevel = "debug"
	LogLevelInfo      LogLevel = "info"
	LogLevelNotice    LogLevel = "notice"
	LogLevelWarning   LogLevel = "warning"
	LogLevelError     LogLevel = "error"
	LogLevelCritical  LogLevel = "critical"
	LogLevelAlert     LogLevel = "alert"
	LogLevelEmergency LogLevel = "emergency"
)

// Log sends a structured log message to the client.
// The message will only be sent if it meets the client's minimum log level.
func (s *Session) Log(ctx context.Context, level LogLevel, message string, data map[string]any) error {
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
