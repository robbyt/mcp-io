package testutil

import (
	"context"

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
	session := capabilities.NewSessionCapability(mockServerSession)
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
