package mcpio

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewInMemoryPair creates a Handler and client Transport for in-memory communication.
// This is useful for testing and integrating mcp-io servers with MCP clients like Google ADK.
//
// The returned handler is configured with an in-memory server transport, and the returned
// client transport can be used to connect to the server. The handler must be started with
// Run() before the client transport is used.
//
// Example with Google ADK:
//
//	handler, clientTransport, err := mcpio.NewInMemoryPair(ctx,
//	    mcpio.WithTool("weather", "Get weather", getWeather),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Start server in background
//	var wg sync.WaitGroup
//	wg.Go(func() {
//	    if err := handler.Run(ctx); err != nil {
//	        log.Printf("Server error: %v", err)
//	    }
//	})
//	defer wg.Wait()
//
//	// Use with ADK (requires google.golang.org/adk)
//	mcpToolSet, err := mcptoolset.New(mcptoolset.Config{
//	    Transport: clientTransport,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	agent, err := llmagent.New(llmagent.Config{
//	    Toolsets: []tool.Toolset{mcpToolSet},
//	})
func NewInMemoryPair(ctx context.Context, opts ...Option) (*Handler, mcp.Transport, error) {
	// Create paired in-memory transports
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	// Build handler using NewHandler with the server transport
	handler, err := NewHandler(append(opts, WithTransport(serverTransport))...)
	if err != nil {
		return nil, nil, err
	}

	return handler, clientTransport, nil
}
