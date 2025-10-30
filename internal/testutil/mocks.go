package testutil

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/stretchr/testify/mock"
)

// MockServerSession is a test mock for the serverSession interface.
// It mocks the MCP SDK's ServerSession at the lowest level.
type MockServerSession struct {
	mock.Mock
}

func (m *MockServerSession) InitializeParams() *mcp.InitializeParams {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*mcp.InitializeParams)
}

func (m *MockServerSession) Elicit(ctx context.Context, params *mcp.ElicitParams) (*mcp.ElicitResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mcp.ElicitResult), args.Error(1)
}

func (m *MockServerSession) Log(ctx context.Context, params *mcp.LoggingMessageParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockServerSession) NotifyProgress(ctx context.Context, params *mcp.ProgressNotificationParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockServerSession) ListRoots(ctx context.Context, params *mcp.ListRootsParams) (*mcp.ListRootsResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mcp.ListRootsResult), args.Error(1)
}

func (m *MockServerSession) CreateMessage(ctx context.Context, params *mcp.CreateMessageParams) (*mcp.CreateMessageResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mcp.CreateMessageResult), args.Error(1)
}

func (m *MockServerSession) ID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockServerSession) Wait() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockServerSession) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockSession is a backwards-compatible wrapper that provides both the mock
// and the Session for tests. Tests should use the Session field for WithSession.
type MockSession struct {
	*MockServerSession
	Session *capabilities.Session
}

// NewMockSession creates a MockSession with both the low-level mock and the Session wrapper.
func NewMockSession() *MockSession {
	mockServerSession := new(MockServerSession)
	session := capabilities.NewSession(mockServerSession)
	return &MockSession{
		MockServerSession: mockServerSession,
		Session:           session,
	}
}

// SetupElicitation configures the mock to support elicitation.
// Call this before setting up Elicit expectations.
func (m *MockSession) SetupElicitation() {
	m.On("InitializeParams").Return(&mcp.InitializeParams{
		Capabilities: &mcp.ClientCapabilities{
			Elicitation: &mcp.ElicitationCapabilities{},
		},
	})
}

// SetupSampling configures the mock to support sampling.
func (m *MockSession) SetupSampling() {
	m.On("InitializeParams").Return(&mcp.InitializeParams{
		Capabilities: &mcp.ClientCapabilities{
			Sampling: &mcp.SamplingCapabilities{},
		},
	})
}

// SetupNoCapabilities configures the mock with no capabilities.
// Use this when testing capability checks that should return false.
func (m *MockSession) SetupNoCapabilities() {
	m.On("InitializeParams").Return(&mcp.InitializeParams{
		Capabilities: &mcp.ClientCapabilities{},
	})
}

// MockToolContext is a mock implementation of mcpio.ToolContext interface
type MockToolContext struct {
	mock.Mock
	session    *capabilities.Session
	identifier string
	tokenInfo  *auth.TokenInfo
	headers    http.Header
}

// NewMockToolContext creates a new mock tool context with the given session
func NewMockToolContext(session *capabilities.Session) *MockToolContext {
	return &MockToolContext{
		session:    session,
		identifier: "",
		tokenInfo:  nil,
		headers:    http.Header{},
	}
}

// GetSession returns the session capability for advanced features
func (m *MockToolContext) GetSession() *capabilities.Session {
	return m.session
}

// GetIdentifier returns the request identifier
func (m *MockToolContext) GetIdentifier() string {
	return m.identifier
}

// GetTokenInfo returns authentication token information
func (m *MockToolContext) GetTokenInfo() *auth.TokenInfo {
	return m.tokenInfo
}

// GetHeaders returns all request headers
func (m *MockToolContext) GetHeaders() http.Header {
	return m.headers
}

// WithIdentifier sets the identifier for testing
func (m *MockToolContext) WithIdentifier(identifier string) *MockToolContext {
	m.identifier = identifier
	return m
}

// WithTokenInfo sets the token info for testing
func (m *MockToolContext) WithTokenInfo(tokenInfo *auth.TokenInfo) *MockToolContext {
	m.tokenInfo = tokenInfo
	return m
}

// WithHeaders sets the headers for testing
func (m *MockToolContext) WithHeaders(headers http.Header) *MockToolContext {
	m.headers = headers
	return m
}
