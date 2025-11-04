// This example demonstrates the mcp-io equivalent of the SDK resources example.
// It shows how mcp-io's functional options API simplifies resource registration
// compared to the SDK's AddResource/AddResourceTemplate methods.
//
// Original SDK example:
// https://github.com/modelcontextprotocol/go-sdk/blob/main/mcp/server_example_test.go
// See Example_resources() function (lines 150-218)

package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/mcpwrapper"
)

// resourceData simulates a simple file system with resource URIs and their content.
var resourceData = map[string]string{
	"file:///a":     "a",
	"file:///dir/x": "x",
	"file:///dir/y": "y",
}

// resourceHandler implements mcp-io's ResourceFunc signature.
// Compare with SDK version:
//
//	SDK:    func(ctx, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error)
//	mcp-io: func(ctx, mcpio.RequestContext) (*mcpio.ResourceContent, error)
//
// Benefits:
//   - No manual ReadResourceResult construction
//   - Simpler content handling with ResourceContent wrapper
//   - RequestContext provides access to URI via GetIdentifier()
func resourceHandler(ctx context.Context, reqCtx mcpio.RequestContext) (*mcpio.ResourceContent, error) {
	// Get the URI from the request context (GetIdentifier returns URI for resources)
	uri := reqCtx.GetIdentifier()

	// Look up the resource content
	content, ok := resourceData[uri]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", uri)
	}

	// Return text content
	return &mcpio.ResourceContent{
		Content:  []byte(content),
		MIMEType: "text/plain",
	}, nil
}

// runResources creates an MCP server with both a static resource and a dynamic
// resource template, calls them via in-memory transport, and returns the formatted
// result. This function encapsulates the full round-trip for testing.
func runResources(ctx context.Context) (string, error) {
	// Create handler using mcp-io's functional options API
	// Compare with SDK:
	//   s := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	//   s.AddResource(&mcp.Resource{URI: "file:///a"}, handler)
	//   s.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: "file:///dir/{f}"}, handler)
	//
	//   mcp-io: Single NewHandler call with functional options
	handler, err := mcpio.NewHandler(
		mcpio.WithName("resource-server"),
		mcpio.WithVersion("v1.0.0"),
		// Static resource - always available at file:///a
		mcpio.WithResource("file:///a", "Static resource", resourceHandler),
		// Dynamic resource template - matches file:///dir/{f} where {f} is any filename
		mcpio.WithResourceTemplate("file:///dir/{f}", "Dynamic directory template", resourceHandler),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create handler: %w", err)
	}

	// Set up in-memory transport using mcp-io's wrapper pattern
	sdkServer := handler.GetServer().Unwrap().(*mcp.Server)
	wrappedServer, clientTransport := mcpwrapper.NewInMemoryServer(sdkServer)

	// Run the server in a goroutine
	go func() {
		if runErr := wrappedServer.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			log.Printf("server run error: %v", runErr)
		}
	}()

	// Create client and connect
	client := mcp.NewClient(&mcp.Implementation{Name: "client"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return "", fmt.Errorf("failed to connect client: %w", err)
	}
	defer func() {
		if closeErr := clientSession.Close(); closeErr != nil {
			log.Printf("error closing client session: %v", closeErr)
		}
	}()

	// List resources
	var resources []string
	for r, err := range clientSession.Resources(ctx, nil) {
		if err != nil {
			return "", fmt.Errorf("failed to list resources: %w", err)
		}
		resources = append(resources, r.URI)
	}

	// List resource templates
	var templates []string
	for t, err := range clientSession.ResourceTemplates(ctx, nil) {
		if err != nil {
			return "", fmt.Errorf("failed to list resource templates: %w", err)
		}
		templates = append(templates, t.URITemplate)
	}

	// Read resources - including one that doesn't exist
	var contents []string
	for _, path := range []string{"a", "dir/x", "b"} {
		res, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "file:///" + path})
		if err != nil {
			contents = append(contents, fmt.Sprintf("error: %v", err))
		} else {
			contents = append(contents, res.Contents[0].Text)
		}
	}

	return fmt.Sprintf("Resources: %v\nTemplates: %v\nContents: %v",
		resources, templates, contents), nil
}

func main() {
	ctx := context.Background()

	output, err := runResources(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(output)

	// Output:
	// Resources: [file:///a]
	// Templates: [file:///dir/{f}]
	// Contents: [a x error: calling "resources/read": Resource not found]
}
