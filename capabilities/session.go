package capabilities

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ClientCapabilities represents the capabilities declared by the client.
type ClientCapabilities struct {
	Elicitation *ElicitationCapabilities
	Sampling    *SamplingCapabilities
	Roots       *RootsCapabilities
}

// serverSession defines the interface for MCP server session operations.
// This interface abstracts *mcp.ServerSession to enable testing with mocks.
type serverSession interface {
	InitializeParams() *mcp.InitializeParams
	Elicit(ctx context.Context, params *mcp.ElicitParams) (*mcp.ElicitResult, error)
	Log(ctx context.Context, params *mcp.LoggingMessageParams) error
	NotifyProgress(ctx context.Context, params *mcp.ProgressNotificationParams) error
	ListRoots(ctx context.Context, params *mcp.ListRootsParams) (*mcp.ListRootsResult, error)
	CreateMessage(ctx context.Context, params *mcp.CreateMessageParams) (*mcp.CreateMessageResult, error)
	ID() string
	Wait() error
	Close() error
}

// Session provides access to all MCP session features.
// It is automatically provided to handlers via the RequestContext parameter.
// Access it using reqCtx.GetSession() or toolCtx.GetSession().
//
// Example:
//
//	func myTool(ctx context.Context, toolCtx mcpio.RequestContext, input Input) (Output, error) {
//	    session := toolCtx.GetSession()
//	    if session.SupportsSampling() {
//	        result, _ := session.CreateMessage(ctx, messages, 1000)
//	    }
//	    return output, nil
//	}
//
// Session wraps mcp.ServerSession and provides ergonomic access to session capabilities.
type Session struct {
	session serverSession
	logger  *slog.Logger
}

// NewSession creates a Session from a serverSession interface.
// In production, pass *mcp.ServerSession. In tests, pass a mock implementing serverSession.
func NewSession(session serverSession) *Session {
	var logger *slog.Logger
	// Try to create MCP logging handler if we have a concrete ServerSession
	if ss, ok := session.(*mcp.ServerSession); ok {
		logger = slog.New(mcp.NewLoggingHandler(ss, nil))
	} else {
		// For tests/mocks, use default logger
		logger = slog.Default()
	}
	return &Session{
		session: session,
		logger:  logger,
	}
}

// Logger returns a slog.Logger that sends logs to the client.
// This provides standard Go logging integration with MCP client logging.
func (s *Session) Logger() *slog.Logger {
	return s.logger
}

// SessionID returns the unique identifier for this session.
func (s *Session) SessionID() string {
	return s.session.ID()
}

// Wait blocks until the client disconnects or the session is closed.
func (s *Session) Wait() error {
	return s.session.Wait()
}

// Close closes this session and the underlying connection.
func (s *Session) Close() error {
	return s.session.Close()
}

// ClientCapabilities returns the capabilities the client declared during initialization.
func (s *Session) ClientCapabilities() *ClientCapabilities {
	initParams := s.session.InitializeParams()
	caps := &ClientCapabilities{}

	if initParams.Capabilities.Elicitation != nil {
		caps.Elicitation = &ElicitationCapabilities{}
	}
	if initParams.Capabilities.Sampling != nil {
		caps.Sampling = &SamplingCapabilities{}
	}
	if initParams.Capabilities.RootsV2 != nil {
		caps.Roots = &RootsCapabilities{
			ListChanged: initParams.Capabilities.RootsV2.ListChanged,
		}
	}

	return caps
}
