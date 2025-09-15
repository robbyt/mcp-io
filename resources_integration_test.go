//go:build integration

package mcpio_test

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type ResourcesIntegrationTestSuite struct {
	testutil.IntegrationSuite
}

func TestResourcesIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ResourcesIntegrationTestSuite))
}

func (s *ResourcesIntegrationTestSuite) TestResourceHandlerIntegration() {
	s.Run("ReadResource", func() {
		ctx := s.Ctx

		// Create in-memory transports
		clientTransport, serverTransport := mcp.NewInMemoryTransports()

		testImpl := &mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}

		// Create test data store
		data := map[string]string{
			"greeting": "Hello, World!",
			"farewell": "Goodbye, World!",
		}

		// Create the server with the same resource as cli-resource
		server, err := mcpio.NewResourceHandler(
			mcpio.WithName("resource-server"),
			mcpio.WithResourceTemplate("res://kv/{key}", "A simple key-value store", func(ctx context.Context, uri string) (*mcpio.ResourceContent, error) {
				key := strings.TrimPrefix(uri, "res://kv/")
				if value, ok := data[key]; ok {
					return &mcpio.ResourceContent{
						Content:  []byte(value),
						MIMEType: "text/plain",
					}, nil
				}
				return nil, mcpio.ResourceNotFoundError(uri)
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

		// Read the greeting resource
		result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
			URI: "res://kv/greeting",
		})
		s.Require().NoError(err)

		s.Assert().Len(result.Contents, 1)

		content := result.Contents[0]
		s.Assert().Equal("res://kv/greeting", content.URI)
		s.Assert().Equal("text/plain", content.MIMEType)
		s.Assert().Equal("Hello, World!", content.Text)
	})
}
