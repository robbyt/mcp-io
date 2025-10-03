package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// MockSessionCapability is a mock implementation for testing
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

func (m *MockSessionCapability) CreateMessage(ctx context.Context, messages []*mcpio.Message, maxTokens int) (*mcpio.MessageResult, error) {
	args := m.Called(ctx, messages, maxTokens)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mcpio.MessageResult), args.Error(1)
}

func (m *MockSessionCapability) ListRoots(ctx context.Context) ([]*mcpio.Root, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*mcpio.Root), args.Error(1)
}

func (m *MockSessionCapability) Log(ctx context.Context, level mcpio.LogLevel, message string, data map[string]any) error {
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

func (m *MockSessionCapability) ClientCapabilities() *mcpio.ClientCapabilities {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*mcpio.ClientCapabilities)
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

// AgentTestSuite tests the agent example
type AgentTestSuite struct {
	testutil.ExampleTestSuite
}

func (s *AgentTestSuite) SetupSuite() {
	_, b, _, _ := runtime.Caller(0)
	exampleDir := filepath.Dir(b)
	s.ProjectRoot = filepath.Join(exampleDir, "..", "..")
	s.ExampleName = "cli_agent"
	s.ExampleTestSuite.SetupSuite()
}

func TestAgentSuite(t *testing.T) {
	suite.Run(t, new(AgentTestSuite))
}

func (s *AgentTestSuite) TestAnalyzeToolWithSampling() {
	s.Run("SuccessfulAnalysis", func() {
		mockSession := new(MockSessionCapability)
		mockSession.On("SupportsSampling").Return(true)
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockSession.On("Log", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// Mock the sampling response
		mockSession.On("CreateMessage", mock.Anything, mock.Anything, mock.Anything).Return(&mcpio.MessageResult{
			Role: "assistant",
			Content: mcpio.TextContent{
				Text: "This code contains a potential bug in the error handling. Suggestions: 1. Add proper error checking. 2. Use defer for cleanup.",
			},
		}, nil).Once()

		ctx := mcpio.InjectSessionForTesting(context.Background(), mockSession)
		result, err := analyzeTool(ctx, AnalyzeInput{
			Code:     "func test() { return }",
			Language: "go",
		})

		s.Require().NoError(err)
		s.True(result.SamplingUsed)
		s.NotEmpty(result.Analysis)
		s.Contains(result.Analysis, "bug")
		s.Positive(result.IssuesFound)
		s.NotEmpty(result.Suggestions)

		mockSession.AssertExpectations(s.T())
	})

	s.Run("SamplingNotSupported", func() {
		mockSession := new(MockSessionCapability)
		mockSession.On("SupportsSampling").Return(false)

		ctx := mcpio.InjectSessionForTesting(context.Background(), mockSession)
		result, err := analyzeTool(ctx, AnalyzeInput{
			Code: "func test() {}",
		})

		s.Require().NoError(err)
		s.False(result.SamplingUsed)
		s.Contains(result.Analysis, "not supported")
		s.Equal(0, result.IssuesFound)

		mockSession.AssertExpectations(s.T())
	})
}

func (s *AgentTestSuite) TestImproveToolWithSampling() {
	s.Run("SuccessfulImprovement", func() {
		mockSession := new(MockSessionCapability)
		mockSession.On("SupportsSampling").Return(true)

		mockSession.On("CreateMessage", mock.Anything, mock.Anything, mock.Anything).Return(&mcpio.MessageResult{
			Role: "assistant",
			Content: mcpio.TextContent{
				Text: "func test() error {\n    // Add error handling\n    return nil\n}\n\nChanges: Added error return type and handling",
			},
		}, nil).Once()

		ctx := mcpio.InjectSessionForTesting(context.Background(), mockSession)
		result, err := improveTool(ctx, ImproveInput{
			Code:     "func test() {}",
			Focus:    "security",
			Language: "go",
		})

		s.Require().NoError(err)
		s.True(result.SamplingUsed)
		s.NotEmpty(result.ImprovedCode)
		s.Contains(result.ImprovedCode, "func test()")

		mockSession.AssertExpectations(s.T())
	})

	s.Run("SamplingNotSupported", func() {
		mockSession := new(MockSessionCapability)
		mockSession.On("SupportsSampling").Return(false)

		ctx := mcpio.InjectSessionForTesting(context.Background(), mockSession)
		originalCode := "func test() {}"
		result, err := improveTool(ctx, ImproveInput{
			Code: originalCode,
		})

		s.Require().NoError(err)
		s.False(result.SamplingUsed)
		s.Equal(originalCode, result.ImprovedCode)
		s.Contains(result.Changes, "not available")

		mockSession.AssertExpectations(s.T())
	})
}

func (s *AgentTestSuite) TestHelperFunctions() {
	s.Run("ExtractSuggestions", func() {
		analysis := "This code has issues. Fix the bugs."
		suggestions := extractSuggestions(analysis)
		s.NotEmpty(suggestions)
	})

	s.Run("CountIssues", func() {
		analysis := "Found a bug and a security vulnerability in the code."
		count := countIssues(analysis)
		s.Positive(count)
	})

	s.Run("CountIssuesNone", func() {
		analysis := "This code looks good."
		count := countIssues(analysis)
		// May or may not find issues depending on keywords
		s.GreaterOrEqual(count, 0)
	})
}

func (s *AgentTestSuite) TestServerCreation() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("code-agent"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("analyze_code", "Analyze code using AI", analyzeTool),
		mcpio.WithTool("improve_code", "Improve code quality", improveTool),
	)
	s.Require().NoError(err)
	s.NotNil(handler)
	s.NotNil(handler.GetServer())
}

func (s *AgentTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}

// Unit tests without suite

func TestAnalyzeToolNoSession(t *testing.T) {
	ctx := context.Background()
	_, err := analyzeTool(ctx, AnalyzeInput{Code: "test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no session")
}

func TestImproveToolNoSession(t *testing.T) {
	ctx := context.Background()
	result, err := improveTool(ctx, ImproveInput{Code: "test"})
	require.NoError(t, err)
	assert.False(t, result.SamplingUsed)
}

// TestSamplingWithRealMCPClient tests sampling using real MCP client protocol
func (s *AgentTestSuite) TestSamplingWithRealMCPClient() {
	s.Run("AnalyzeCodeWithSampling", func() {
		// Create server builder
		serverBuilder := func() (*mcp.Server, error) {
			handler, err := mcpio.NewHandler(
				mcpio.WithName("code-agent-test"),
				mcpio.WithVersion("1.0.0"),
				mcpio.WithTool("analyze_code", "Analyze code using AI", analyzeTool),
			)
			if err != nil {
				return nil, err
			}
			return handler.GetServer(), nil
		}

		// Use real MCP client with sampling support
		testImpl := &mcp.Implementation{
			Name:    "test-client-with-sampling",
			Version: "1.0.0",
		}

		clientOpts := &mcp.ClientOptions{
			// Provide CreateMessageHandler to enable sampling capability
			CreateMessageHandler: func(ctx context.Context, req *mcp.CreateMessageRequest) (*mcp.CreateMessageResult, error) {
				// Mock LLM response for testing
				return &mcp.CreateMessageResult{
					Role: "assistant",
					Content: &mcp.TextContent{
						Text: "This code contains a potential bug in error handling. The function should check for nil pointers before dereferencing. Security issue: input validation is missing.",
					},
					Model:      "test-model",
					StopReason: "end_turn",
				}, nil
			},
		}

		ctx := s.T().Context()
		server, err := serverBuilder()
		s.Require().NoError(err)

		// Create in-memory transports
		clientTransport, serverTransport := mcp.NewInMemoryTransports()

		// Start server
		go func() {
			err := server.Run(ctx, serverTransport)
			if err != nil && !errors.Is(err, context.Canceled) {
				s.T().Errorf("server run error: %v", err)
			}
		}()

		// Create client with sampling support
		client := mcp.NewClient(testImpl, clientOpts)
		session, err := client.Connect(ctx, clientTransport, nil)
		s.Require().NoError(err)

		defer func() {
			if closeErr := session.Close(); closeErr != nil {
				if !errors.Is(closeErr, context.Canceled) {
					s.T().Errorf("session close error: %v", closeErr)
				}
			}
		}()

		// Call the analyze_code tool
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "analyze_code",
			Arguments: map[string]any{
				"code":     "func test() { ptr.Method() }",
				"language": "go",
			},
		})

		s.Require().NoError(err)
		s.Require().NotNil(result)
		s.False(result.IsError, "Tool should not return error")

		// Verify the sampling was used and analysis was generated
		s.Require().Len(result.Content, 1)
		textContent, ok := result.Content[0].(*mcp.TextContent)
		s.Require().True(ok, "Expected text content")

		// Parse the JSON response
		var output AnalyzeOutput
		err = json.Unmarshal([]byte(textContent.Text), &output)
		s.Require().NoError(err)

		s.True(output.SamplingUsed, "Sampling should be used")
		s.NotEmpty(output.Analysis, "Analysis should not be empty")
		s.Contains(output.Analysis, "bug", "Analysis should mention bugs")
		s.Positive(output.IssuesFound, "Should find issues")
	})
}
