//go:build integration

package mcpio_test

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type PromptsIntegrationTestSuite struct {
	testutil.IntegrationSuite
}

func TestPromptsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PromptsIntegrationTestSuite))
}

func (s *PromptsIntegrationTestSuite) TestPromptHandlerIntegration() {
	s.Run("Greeter", func() {
		ctx := s.Ctx

		// Create in-memory transports
		clientTransport, serverTransport := mcp.NewInMemoryTransports()

		testImpl := &mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}

		// Create the server with the same prompt as cli-prompt
		server, err := mcpio.NewHandler(
			mcpio.WithName("prompt-server"),
			mcpio.WithPrompt("greeter", "Generates a friendly greeting", func(ctx context.Context, args map[string]any) (*mcpio.PromptResult, error) {
				name, _ := args["name"].(string)
				if name == "" {
					name = "World"
				}
				return &mcpio.PromptResult{
					Messages: []mcpio.PromptMessage{
						{
							Role:    "system",
							Content: "You are a helpful assistant.",
						},
						{
							Role:    "user",
							Content: "Create a friendly greeting for " + name,
						},
					},
				}, nil
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

		// Get the greeter prompt
		result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
			Name:      "greeter",
			Arguments: map[string]string{"name": "World"},
		})
		s.Require().NoError(err)

		s.Len(result.Messages, 2)

		// Check system message
		s.Equal(mcp.Role("system"), result.Messages[0].Role)
		if systemContent, ok := result.Messages[0].Content.(*mcp.TextContent); s.True(ok) {
			s.Equal("You are a helpful assistant.", systemContent.Text)
		}

		// Check user message
		s.Equal(mcp.Role("user"), result.Messages[1].Role)
		if userContent, ok := result.Messages[1].Content.(*mcp.TextContent); s.True(ok) {
			s.Equal("Create a friendly greeting for World", userContent.Text)
		}
	})
}
