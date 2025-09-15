package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"

	mcpio "github.com/robbyt/mcp-io"
)

var data = map[string]string{
	"greeting": "Hello, World!",
	"farewell": "Goodbye, World!",
}

func resourceReader(ctx context.Context, uri string) (*mcpio.ResourceContent, error) {
	key := strings.TrimPrefix(uri, "res://kv/")
	if value, ok := data[key]; ok {
		return &mcpio.ResourceContent{
			Content:  []byte(value),
			MIMEType: "text/plain",
		}, nil
	}
	return nil, errors.New("resource not found") // a real implementation would have a typed error
}

func main() {
	handler, err := mcpio.NewResourceHandler(
		mcpio.WithName("resource-server"),
		mcpio.WithResource("res://kv/", "A simple key-value store", resourceReader),
	)
	if err != nil {
		log.Fatal("Failed to create MCP handler:", err)
	}

	if err := handler.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
		log.Fatal("Failed to serve MCP via stdio:", err)
	}
}
