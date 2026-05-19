package transport

import (
	"context"
	"fmt"
	"net/http"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

// ServeStreamableHTTP starts the MCP server using the Streamable HTTP transport.
// It listens on the given address and handles JSON-RPC over HTTP at /mcp.
// The server supports session management via the Mcp-Session-Id header.
func ServeStreamableHTTP(ctx context.Context, server *mcpserver.MCPServer, addr string) error {
	httpSrv := mcpserver.NewStreamableHTTPServer(server,
		mcpserver.WithEndpointPath("/mcp"),
		mcpserver.WithStateful(true),
	)

	mux := http.NewServeMux()
	mux.Handle("/mcp", httpSrv)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http server: %w", err)
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
