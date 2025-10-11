package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/suite"
)

// CliResourceTestSuite tests the cli_resource example
type CliResourceTestSuite struct {
	testutil.ExampleTestSuite
}

func (s *CliResourceTestSuite) SetupSuite() {
	// Get project root - we're in examples/cli_resource
	_, b, _, _ := runtime.Caller(0)
	exampleDir := filepath.Dir(b)
	s.ProjectRoot = filepath.Join(exampleDir, "..", "..")
	s.ExampleName = "cli_resource"

	// Call parent SetupSuite
	s.ExampleTestSuite.SetupSuite()
}

func TestCliResourceSuite(t *testing.T) {
	suite.Run(t, new(CliResourceTestSuite))
}

func (s *CliResourceTestSuite) TestFunctionality() {
	ctx := s.T().Context()

	// Create server using the same logic as main()
	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("resource-server"),
			mcpio.WithResourceTemplate("res://kv/{key}", "A simple key-value store", resourceReader),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		testCases := []struct {
			name        string
			uri         string
			expectError bool
			expected    string
		}{
			{
				name:        "ReadGreeting",
				uri:         "res://kv/greeting",
				expectError: false,
				expected:    "Hello, World!",
			},
			{
				name:        "ReadFarewell",
				uri:         "res://kv/farewell",
				expectError: false,
				expected:    "Goodbye, World!",
			},
			{
				name:        "ReadNonExistent",
				uri:         "res://kv/nonexistent",
				expectError: true,
				expected:    "",
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
					URI: tc.uri,
				})

				if tc.expectError {
					s.Require().Error(err)
				} else {
					s.Require().NoError(err)
					s.Require().Len(result.Contents, 1)

					// Resource should return text content
					content := result.Contents[0]
					s.Equal(tc.expected, content.Text)
					s.Equal("text/plain", content.MIMEType)
				}
			})
		}
	})
}

func (s *CliResourceTestSuite) TestListing() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("resource-server"),
			mcpio.WithResourceTemplate("res://kv/{key}", "A simple key-value store", resourceReader),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		// Resource templates may not appear in regular resource listing
		result, err := session.ListResources(ctx, &mcp.ListResourcesParams{})
		s.Require().NoError(err)
		// Templates might not appear in the list, which is expected behavior
		s.T().Logf("Found %d resources", len(result.Resources))
	})
}

func (s *CliResourceTestSuite) TestErrorHandling() {
	ctx := s.T().Context()

	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("resource-server"),
			mcpio.WithResourceTemplate("res://kv/{key}", "A simple key-value store", resourceReader),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	s.WithMCPSession(serverBuilder, func(session *mcp.ClientSession) {
		// Test reading non-existent resource
		_, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
			URI: "res://invalid/uri",
		})
		s.Require().Error(err)
		s.Contains(err.Error(), "not found")
	})
}

func (s *CliResourceTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()

	// Verify binary was created
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}
