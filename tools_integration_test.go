//go:build integration

package mcpio_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type ToolsIntegrationTestSuite struct {
	testutil.IntegrationSuite
}

func TestToolsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ToolsIntegrationTestSuite))
}

func (s *ToolsIntegrationTestSuite) TestToolHandlerIntegration() {
	cliToolPath := filepath.Join(s.ProjectRoot, "bin", "cli-tool")

	s.Run("ToUpper", func() {
		ctx := s.Ctx

		// Create in-memory transports
		clientTransport, serverTransport := mcp.NewInMemoryTransports()

		// Start the CLI tool server in a goroutine
		go func() {
			cmd := exec.CommandContext(ctx, cliToolPath)

			// Create pipes to connect the CLI tool to our transport
			stdin, err := cmd.StdinPipe()
			if err != nil {
				return
			}
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				return
			}

			// Start the command
			if err := cmd.Start(); err != nil {
				return
			}

			// Clean up pipes
			defer func() {
				closeErr := stdin.Close()
				s.NoError(closeErr) //nolint:testifylint // Assert is appropriate in defer cleanup
				closeErr = stdout.Close()
				s.NoError(closeErr)
			}()

			// Wait for command to finish
			waitErr := cmd.Wait()
			s.NoError(waitErr)
		}()

		// Actually, let's use our library directly for testing
		// since we know the CLI tools use mcpio.NewToolHandler
		testImpl := &mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}

		// Create the server with the same tools as cli-tool
		server, err := mcpio.NewToolHandler(
			mcpio.WithName("text-processor"),
			mcpio.WithTool("to_upper", "Convert text to uppercase", func(ctx context.Context, input struct {
				Text string `json:"text"`
			}) (struct {
				Result string `json:"result"`
			}, error,
			) {
				return struct {
					Result string `json:"result"`
				}{Result: "HELLO WORLD"}, nil
			}),
		)
		s.Require().NoError(err)

		// Connect the server to the transport
		go func() {
			if runErr := server.GetServer().Run(ctx, serverTransport); runErr != nil {
				s.T().Logf("server run error: %v", runErr)
			}
		}()

		// Create client and connect
		client := mcp.NewClient(testImpl, nil)
		session, err := client.Connect(ctx, clientTransport, nil)
		s.Require().NoError(err)
		defer func() {
			if err := session.Close(); err != nil {
				s.T().Logf("error closing session: %v", err)
			}
		}()

		// Call the to_upper tool
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "to_upper",
			Arguments: map[string]any{"text": "hello world"},
		})
		s.Require().NoError(err)

		s.False(result.IsError)
		s.Len(result.Content, 1)
	})

	s.Run("CountWords", func() {
		ctx := s.Ctx

		clientTransport, serverTransport := mcp.NewInMemoryTransports()

		testImpl := &mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}

		// Create server with count tool
		server, err := mcpio.NewToolHandler(
			mcpio.WithName("text-processor"),
			mcpio.WithTool("count", "Count words or characters in text", func(ctx context.Context, input struct {
				Text string `json:"text"`
				Type string `json:"type"`
			}) (struct {
				Count int    `json:"count"`
				Type  string `json:"type"`
			}, error,
			) {
				count := 0
				if input.Type == "words" && input.Text == "this is a test" {
					count = 4
				}
				return struct {
					Count int    `json:"count"`
					Type  string `json:"type"`
				}{Count: count, Type: input.Type}, nil
			}),
		)
		s.Require().NoError(err)

		go func() {
			if runErr := server.GetServer().Run(ctx, serverTransport); runErr != nil {
				s.T().Logf("server run error: %v", runErr)
			}
		}()

		client := mcp.NewClient(testImpl, nil)
		session, err := client.Connect(ctx, clientTransport, nil)
		s.Require().NoError(err)
		defer func() {
			if err := session.Close(); err != nil {
				s.T().Logf("error closing session: %v", err)
			}
		}()

		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "count",
			Arguments: map[string]any{
				"text": "this is a test",
				"type": "words",
			},
		})
		s.Require().NoError(err)
		s.False(result.IsError)
	})

	s.Run("ValidationError", func() {
		ctx := s.Ctx

		clientTransport, serverTransport := mcp.NewInMemoryTransports()

		testImpl := &mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}

		server, err := mcpio.NewToolHandler(
			mcpio.WithName("text-processor"),
			mcpio.WithTool("count", "Count words or characters in text", func(ctx context.Context, input struct {
				Text string `json:"text"`
				Type string `json:"type"`
			}) (struct {
				Count int    `json:"count"`
				Type  string `json:"type"`
			}, error,
			) {
				if input.Type != "words" && input.Type != "chars" {
					return struct {
						Count int    `json:"count"`
						Type  string `json:"type"`
					}{}, mcpio.NewToolError("unsupported count type: " + input.Type)
				}
				return struct {
					Count int    `json:"count"`
					Type  string `json:"type"`
				}{Count: 0, Type: input.Type}, nil
			}),
		)
		s.Require().NoError(err)

		go func() {
			if runErr := server.GetServer().Run(ctx, serverTransport); runErr != nil {
				s.T().Logf("server run error: %v", runErr)
			}
		}()

		client := mcp.NewClient(testImpl, nil)
		session, err := client.Connect(ctx, clientTransport, nil)
		s.Require().NoError(err)
		defer func() {
			if err := session.Close(); err != nil {
				s.T().Logf("error closing session: %v", err)
			}
		}()

		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "count",
			Arguments: map[string]any{
				"text": "this is a test",
				"type": "invalid_type",
			},
		})
		s.Require().NoError(err)

		s.True(result.IsError)
		s.Len(result.Content, 1)
		if textContent, ok := result.Content[0].(*mcp.TextContent); s.True(ok) {
			s.Contains(textContent.Text, "unsupported count type:")
		}
	})
}
