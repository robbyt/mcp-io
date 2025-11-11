package main

import (
	"context"
	"log"
	"strings"

	mcpio "github.com/robbyt/mcp-io"
)

var data = map[string]string{
	"greeting": "Hello, World!",
	"farewell": "Goodbye, World!",
}

func resourceReader(ctx context.Context, reqCtx mcpio.RequestContext) (*mcpio.ResourceContent, error) {
	uri := reqCtx.GetIdentifier()
	key := strings.TrimPrefix(uri, "res://kv/")
	if value, ok := data[key]; ok {
		return &mcpio.ResourceContent{
			Content:  []byte(value),
			MIMEType: "text/plain",
		}, nil
	}
	return nil, mcpio.ResourceNotFoundError(uri)
}

func main() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("resource-server"),
		mcpio.WithResourceTemplate("res://kv/{key}", "A simple key-value store", resourceReader),
	)
	if err != nil {
		log.Fatal("Failed to create MCP handler:", err)
	}

	if err := handler.Run(context.Background()); err != nil {
		log.Fatal("Failed to serve MCP via stdio:", err)
	}
}
