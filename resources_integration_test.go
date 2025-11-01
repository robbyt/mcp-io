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
		// Create test data store
		data := map[string]string{
			"greeting": "Hello, World!",
			"farewell": "Goodbye, World!",
		}

		// Create the server with the same resource as cli-resource
		handler, err := mcpio.NewHandler(
			mcpio.WithName("resource-server"),
			mcpio.WithResourceTemplate("res://kv/{key}", "A simple key-value store", func(ctx context.Context, reqCtx mcpio.RequestContext) (*mcpio.ResourceContent, error) {
				uri := reqCtx.GetIdentifier()
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

		// Connect client using testutil helper
		session := testutil.ConnectInMemory(s.T(), handler)

		// Read the greeting resource
		result, err := session.ReadResource(s.Ctx, &mcp.ReadResourceParams{
			URI: "res://kv/greeting",
		})
		s.Require().NoError(err)

		s.Len(result.Contents, 1)

		content := result.Contents[0]
		s.Equal("res://kv/greeting", content.URI)
		s.Equal("text/plain", content.MIMEType)
		s.Equal("Hello, World!", content.Text)
	})
}
