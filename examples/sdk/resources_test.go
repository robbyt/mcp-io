// This example demonstrates mcp-io's resource registration using functional options.
// Compare with the SDK's AddResource/AddResourceTemplate methods.
//
// SDK pattern:
//   server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
//   server.AddResource(&mcp.Resource{URI: "file:///a"}, handler)
//   server.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: "file:///dir/{f}"}, handler)
//
// mcp-io pattern:
//   handler, err := mcpio.NewHandler(
//     mcpio.WithName("resource-server"),
//     mcpio.WithResource("file:///a", "Static resource", handler),
//     mcpio.WithResourceTemplate("file:///dir/{f}", "Dynamic template", handler),
//   )

package sdk_test

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/mcpwrapper"
)

// Example_resources demonstrates mcp-io's functional options API for resource registration.
//
// Key differences from the SDK:
//   - Functional options pattern for resource configuration
//   - Simplified ResourceFunc signature
//   - RequestContext provides access to URI via GetIdentifier()
//   - Cleaner content handling with ResourceContent wrapper
func Example_resources() {
	ctx := context.Background()

	// Simulate a simple file system with resource URIs and content
	resourceData := map[string]string{
		"file:///a":     "a",
		"file:///dir/x": "x",
		"file:///dir/y": "y",
	}

	// Resource handler using mcp-io's ResourceFunc signature
	// SDK version:    func(ctx, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error)
	// mcp-io version: func(ctx, mcpio.RequestContext) (*mcpio.ResourceContent, error)
	resourceHandler := func(ctx context.Context, reqCtx mcpio.RequestContext) (*mcpio.ResourceContent, error) {
		// Get the URI from the request context
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

	// Create handler using functional options API
	handler, err := mcpio.NewHandler(
		mcpio.WithName("resource-server"),
		mcpio.WithVersion("v1.0.0"),
		// Static resource - always available at file:///a
		mcpio.WithResource("file:///a", "Static resource", resourceHandler),
		// Dynamic resource template - matches file:///dir/{f} where {f} is any filename
		mcpio.WithResourceTemplate("file:///dir/{f}", "Dynamic directory template", resourceHandler),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Set up in-memory transport
	sdkServer, ok := handler.GetServer().Unwrap().(*mcp.Server)
	if !ok {
		log.Fatal("failed to unwrap SDK server")
	}
	wrappedServer, clientTransport := mcpwrapper.NewInMemoryServer(sdkServer)

	// Run the server
	go func() {
		if runErr := wrappedServer.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			log.Printf("server run error: %v", runErr)
		}
	}()

	// Create client and connect
	client := mcp.NewClient(&mcp.Implementation{Name: "client"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		log.Fatal(err)
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
			log.Fatal(err)
		}
		resources = append(resources, r.URI)
	}

	// List resource templates
	var templates []string
	for t, err := range clientSession.ResourceTemplates(ctx, nil) {
		if err != nil {
			log.Fatal(err)
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

	fmt.Printf("Resources: %v\n", resources)
	fmt.Printf("Templates: %v\n", templates)
	fmt.Printf("Contents: %v\n", contents)

	// Output:
	// Resources: [file:///a]
	// Templates: [file:///dir/{f}]
	// Contents: [a x error: calling "resources/read": Resource not found]
}
