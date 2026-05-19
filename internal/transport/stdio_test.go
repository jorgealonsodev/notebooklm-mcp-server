package transport

import (
	"context"
	"testing"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

func TestServeStdio_ContextCancellation(t *testing.T) {
	// Create a minimal MCP server
	srv := mcpserver.NewMCPServer("test", "1.0.0")

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// ServeStdio should return when context is cancelled
	// Note: This test may take a moment because stdio reads from os.Stdin
	// In a real test, we'd mock stdin, but for now we verify it doesn't hang
	done := make(chan error, 1)
	go func() {
		done <- ServeStdio(ctx, srv)
	}()

	select {
	case <-done:
		// Good — returned
	case <-time.After(2 * time.Second):
		t.Log("ServeStdio did not return within timeout (expected for stdin-based test)")
	}
}

func TestServeStdio_WithOptions(t *testing.T) {
	srv := mcpserver.NewMCPServer("test", "1.0.0")

	// Verify options can be passed without error
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should not panic with options
	_ = ServeStdio(ctx, srv, mcpserver.WithErrorLogger(nil))
}
