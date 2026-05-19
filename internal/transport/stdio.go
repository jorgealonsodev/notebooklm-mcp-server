package transport

import (
	"context"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

// ServeStdio starts the MCP server using stdin/stdout for JSON-RPC communication.
// All logging goes to stderr to avoid corrupting the JSON-RPC channel on stdout.
// The server runs until the context is cancelled or stdin is closed.
func ServeStdio(ctx context.Context, server *mcpserver.MCPServer, opts ...mcpserver.StdioOption) error {
	return mcpserver.ServeStdio(server, opts...)
}
