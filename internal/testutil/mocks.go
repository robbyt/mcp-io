package testutil

import (
	"context"
	"errors"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/stretchr/testify/mock"
)

// MockSessionCapability is a test mock for capabilities.SessionCapability
// that can be used across example tests
type MockSessionCapability struct {
	mock.Mock
}

func (m *MockSessionCapability) Elicit(ctx context.Context, message string, requestedSchema any) (*mcp.ElicitResult, error) {
	args := m.Called(ctx, message, requestedSchema)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mcp.ElicitResult), args.Error(1)
}

func (m *MockSessionCapability) CreateMessage(ctx context.Context, messages []*capabilities.Message, maxTokens int) (*capabilities.MessageResult, error) {
	args := m.Called(ctx, messages, maxTokens)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*capabilities.MessageResult), args.Error(1)
}

func (m *MockSessionCapability) CreateMessageRaw(ctx context.Context, params *mcp.CreateMessageParams) (*mcp.CreateMessageResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mcp.CreateMessageResult), args.Error(1)
}

func (m *MockSessionCapability) ListRoots(ctx context.Context) ([]*capabilities.Root, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*capabilities.Root), args.Error(1)
}

func (m *MockSessionCapability) Log(ctx context.Context, level capabilities.LogLevel, message string, data map[string]any) error {
	args := m.Called(ctx, level, message, data)
	return args.Error(0)
}

func (m *MockSessionCapability) Logger() *slog.Logger {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*slog.Logger)
}

func (m *MockSessionCapability) NotifyProgress(ctx context.Context, progress, total float64) error {
	args := m.Called(ctx, progress, total)
	return args.Error(0)
}

func (m *MockSessionCapability) SessionID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSessionCapability) ClientCapabilities() *capabilities.ClientCapabilities {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*capabilities.ClientCapabilities)
}

func (m *MockSessionCapability) SupportsElicitation() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSessionCapability) SupportsSampling() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSessionCapability) Wait() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSessionCapability) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockSamplingSession is a minimal mock for testing sampling
type MockSamplingSession struct {
	samplingSupported bool
	response          *capabilities.MessageResult
}

func (m *MockSamplingSession) Elicit(ctx context.Context, message string, requestedSchema any) (*mcp.ElicitResult, error) {
	return nil, errors.New("elicitation not supported")
}

func (m *MockSamplingSession) CreateMessage(ctx context.Context, messages []*capabilities.Message, maxTokens int) (*capabilities.MessageResult, error) {
	return m.response, nil
}

func (m *MockSamplingSession) CreateMessageRaw(ctx context.Context, params *mcp.CreateMessageParams) (*mcp.CreateMessageResult, error) {
	return nil, nil
}

func (m *MockSamplingSession) ListRoots(ctx context.Context) ([]*capabilities.Root, error) {
	return nil, nil
}

func (m *MockSamplingSession) Log(ctx context.Context, level capabilities.LogLevel, message string, data map[string]any) error {
	return nil
}

func (m *MockSamplingSession) Logger() *slog.Logger {
	return slog.Default()
}

func (m *MockSamplingSession) NotifyProgress(ctx context.Context, progress, total float64) error {
	return nil
}

func (m *MockSamplingSession) SessionID() string {
	return "test-sampling-session"
}

func (m *MockSamplingSession) ClientCapabilities() *capabilities.ClientCapabilities {
	return &capabilities.ClientCapabilities{}
}

func (m *MockSamplingSession) SupportsElicitation() bool {
	return false
}

func (m *MockSamplingSession) SupportsSampling() bool {
	return m.samplingSupported
}

func (m *MockSamplingSession) Wait() error {
	return nil
}

func (m *MockSamplingSession) Close() error {
	return nil
}

// NewMockSamplingSession creates a new MockSamplingSession with the given sampling support and response
func NewMockSamplingSession(samplingSupported bool, response *capabilities.MessageResult) *MockSamplingSession {
	return &MockSamplingSession{
		samplingSupported: samplingSupported,
		response:          response,
	}
}
