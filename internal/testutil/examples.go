package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/mcpwrapper"
	"github.com/stretchr/testify/suite"
)

// ExampleTestSuite provides utilities for testing MCP examples using testify suites
type ExampleTestSuite struct {
	suite.Suite
	ProjectRoot string
	ExampleName string

	// HTTP server fields (for HTTP-based examples)
	server  *httptest.Server
	handler http.Handler
}

// SetupSuite initializes the test suite
func (s *ExampleTestSuite) SetupSuite() {
	s.Require().NotEmpty(s.ProjectRoot, "ProjectRoot must be set")
	s.Require().NotEmpty(s.ExampleName, "ExampleName must be set")
}

// SetupTest sets up HTTP server if handler is configured
func (s *ExampleTestSuite) SetupTest() {
	if s.handler != nil {
		s.server = httptest.NewServer(s.handler)
	}
}

// TearDownTest stops the HTTP server after each test
func (s *ExampleTestSuite) TearDownTest() {
	if s.server != nil {
		s.server.Close()
		s.server = nil
	}
}

// SetHandler sets the HTTP handler for HTTP-based tests
func (s *ExampleTestSuite) SetHandler(handler http.Handler) {
	s.handler = handler
}

// BinaryPath returns the path to the built example binary
func (s *ExampleTestSuite) BinaryPath() string {
	return filepath.Join(s.ProjectRoot, "bin", s.ExampleName)
}

// WithMCPSession tests an MCP server using in-memory transport
func (s *ExampleTestSuite) WithMCPSession(handler *mcpio.Handler, testFunc func(*mcp.ClientSession)) {
	ctx := s.T().Context()

	// Get SDK server and create in-memory server
	sdkServer, ok := handler.GetServer().Unwrap().(*mcp.Server)
	s.Require().True(ok, "failed to unwrap SDK server")

	wrappedServer, clientTransport := mcpwrapper.NewInMemoryServer(sdkServer)

	// Start server with embedded transport
	go func() {
		err := wrappedServer.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			s.T().Errorf("server run error: %v", err)
		}
	}()

	// Create client
	testImpl := &mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}

	client := mcp.NewClient(testImpl, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	s.Require().NoError(err)

	defer func() {
		if closeErr := session.Close(); closeErr != nil {
			if !errors.Is(closeErr, context.Canceled) {
				s.T().Errorf("session close error: %v", closeErr)
			}
		}
	}()

	testFunc(session)
}

// CallTool calls an MCP tool and returns the result
func (s *ExampleTestSuite) CallTool(ctx context.Context, session *mcp.ClientSession, name string, args map[string]any) ([]mcp.Content, bool, error) {
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return nil, false, err
	}

	return result.Content, result.IsError, nil
}

// GetPrompt gets an MCP prompt and returns the messages
func (s *ExampleTestSuite) GetPrompt(ctx context.Context, session *mcp.ClientSession, name string, args map[string]string) ([]*mcp.PromptMessage, error) {
	result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return nil, err
	}

	return result.Messages, nil
}

// ReadResource reads an MCP resource
func (s *ExampleTestSuite) ReadResource(ctx context.Context, session *mcp.ClientSession, uri string) (*mcp.ReadResourceResult, error) {
	return session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: uri,
	})
}

// ListTools lists available tools from a session
func (s *ExampleTestSuite) ListTools(ctx context.Context, session *mcp.ClientSession) ([]*mcp.Tool, error) {
	result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, err
	}
	return result.Tools, nil
}

// ListPrompts lists available prompts from a session
func (s *ExampleTestSuite) ListPrompts(ctx context.Context, session *mcp.ClientSession) ([]*mcp.Prompt, error) {
	result, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
	if err != nil {
		return nil, err
	}
	return result.Prompts, nil
}

// ListResources lists available resources from a session
func (s *ExampleTestSuite) ListResources(ctx context.Context, session *mcp.ClientSession) ([]*mcp.Resource, error) {
	result, err := session.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		return nil, err
	}
	return result.Resources, nil
}

// URL returns the HTTP server URL (for HTTP-based tests)
func (s *ExampleTestSuite) URL() string {
	if s.server != nil {
		return s.server.URL
	}
	return ""
}

// TestHealth tests the /health endpoint (for HTTP-based tests)
func (s *ExampleTestSuite) TestHealth() {
	if s.server == nil {
		s.T().Skip("No HTTP server configured, skipping health test")
		return
	}

	resp, err := http.Get(s.URL() + "/health")
	s.Require().NoError(err)
	defer func() { s.Require().NoError(resp.Body.Close()) }()

	s.Require().Equal(http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.Require().Contains(string(body), "OK")
}

// MCPRequest represents a JSON-RPC request
type MCPRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
	ID      int    `json:"id"`
}

// MCPResponse represents a JSON-RPC response
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// SendMCPRequest sends a JSON-RPC request to the MCP endpoint (for HTTP-based tests)
func (s *ExampleTestSuite) SendMCPRequest(req MCPRequest) (*MCPResponse, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(s.URL()+"/mcp", "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var mcpResp MCPResponse
	if err := json.Unmarshal(body, &mcpResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &mcpResp, nil
}

// BuildBinary builds the example binary in a temp directory
func (s *ExampleTestSuite) BuildBinary() string {
	s.T().Helper()
	tempDir := s.T().TempDir()
	binaryPath := filepath.Join(tempDir, "test_"+s.ExampleName)
	exampleDir := filepath.Join(s.ProjectRoot, "examples", s.ExampleName)

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = exampleDir
	err := cmd.Run()
	s.Require().NoError(err, "Failed to build %s example", s.ExampleName)

	return binaryPath
}
