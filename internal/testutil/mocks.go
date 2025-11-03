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

// MockRequestContext is a mock implementation of mcpio.RequestContext interface
type MockRequestContext struct {
	mock.Mock
	session       *capabilities.Session
	identifier    string
	tokenInfo     *auth.TokenInfo
	headers       http.Header
	progressToken any
}

// NewMockRequestContext creates a new mock request context with auto-configured mock expectations.
// Tests can override the default return values by calling .On() methods.
func NewMockRequestContext(session *capabilities.Session) *MockRequestContext {
	m := &MockRequestContext{
		session:       session,
		identifier:    "",
		tokenInfo:     nil,
		headers:       http.Header{},
		progressToken: nil,
	}

	// Auto-setup default mock expectations
	m.On("GetSession").Return(session)
	m.On("GetIdentifier").Return("")
	m.On("GetTokenInfo").Return((*auth.TokenInfo)(nil))
	m.On("GetHeaders").Return(http.Header{})
	m.On("GetMeta").Return(map[string]any{})
	m.On("GetProgressToken").Return(nil)

	return m
}

// GetSession returns the session capability for advanced features
func (m *MockRequestContext) GetSession() *capabilities.Session {
	args := m.Called()
	return args.Get(0).(*capabilities.Session)
}

// GetIdentifier returns the request identifier
func (m *MockRequestContext) GetIdentifier() string {
	args := m.Called()
	return args.String(0)
}

// GetTokenInfo returns authentication token information
func (m *MockRequestContext) GetTokenInfo() *auth.TokenInfo {
	args := m.Called()
	return args.Get(0).(*auth.TokenInfo)
}

// GetHeaders returns all request headers
func (m *MockRequestContext) GetHeaders() http.Header {
	args := m.Called()
	return args.Get(0).(http.Header)
}

// GetMeta returns metadata from the request parameters
func (m *MockRequestContext) GetMeta() map[string]any {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[string]any)
}

// GetProgressToken returns the progress token from the request
func (m *MockRequestContext) GetProgressToken() any {
	args := m.Called()
	return args.Get(0)
}
