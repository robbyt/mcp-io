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

// LogDebug sends a debug-level log message to the client.
func (s *Session) LogDebug(ctx context.Context, message string, data map[string]any) error {
	return s.Log(ctx, LogLevelDebug, message, data)
}

// LogInfo sends an info-level log message to the client.
func (s *Session) LogInfo(ctx context.Context, message string, data map[string]any) error {
	return s.Log(ctx, LogLevelInfo, message, data)
}

// LogNotice sends a notice-level log message to the client.
func (s *Session) LogNotice(ctx context.Context, message string, data map[string]any) error {
	return s.Log(ctx, LogLevelNotice, message, data)
}

// LogWarning sends a warning-level log message to the client.
func (s *Session) LogWarning(ctx context.Context, message string, data map[string]any) error {
	return s.Log(ctx, LogLevelWarning, message, data)
}

// LogError sends an error-level log message to the client.
func (s *Session) LogError(ctx context.Context, message string, data map[string]any) error {
	return s.Log(ctx, LogLevelError, message, data)
}

// LogCritical sends a critical-level log message to the client.
func (s *Session) LogCritical(ctx context.Context, message string, data map[string]any) error {
	return s.Log(ctx, LogLevelCritical, message, data)
}

// LogAlert sends an alert-level log message to the client.
func (s *Session) LogAlert(ctx context.Context, message string, data map[string]any) error {
	return s.Log(ctx, LogLevelAlert, message, data)
}

// LogEmergency sends an emergency-level log message to the client.
func (s *Session) LogEmergency(ctx context.Context, message string, data map[string]any) error {
	return s.Log(ctx, LogLevelEmergency, message, data)
}
