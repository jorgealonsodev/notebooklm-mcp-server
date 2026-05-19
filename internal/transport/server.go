// Package transport provides MCP server transport implementations (stdio and
// Streamable HTTP) with graceful shutdown support.
package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jorge/notebooklm-mcp-server/internal/auth"
	"github.com/jorge/notebooklm-mcp-server/internal/browser"
	"github.com/jorge/notebooklm-mcp-server/internal/config"
	"github.com/jorge/notebooklm-mcp-server/internal/library"
	"github.com/jorge/notebooklm-mcp-server/internal/notebooklm"
	"github.com/jorge/notebooklm-mcp-server/internal/resources"
	"github.com/jorge/notebooklm-mcp-server/internal/session"
	"github.com/jorge/notebooklm-mcp-server/internal/tools"
	"github.com/jorge/notebooklm-mcp-server/internal/utils"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// TransportType represents the transport mode.
type TransportType string

const (
	// TransportStdio uses stdin/stdout for JSON-RPC.
	TransportStdio TransportType = "stdio"
	// TransportHTTP uses Streamable HTTP for JSON-RPC.
	TransportHTTP TransportType = "http"
)

// ServerConfig holds transport configuration.
type ServerConfig struct {
	Transport   TransportType
	Host        string
	Port        int
	Profile     tools.ToolProfile
	Disabled    []string
}

// MCPServer wraps the mcp-go server with our domain components and transport.
type MCPServer struct {
	cfg        config.Config
	server     *mcpserver.MCPServer
	transport  TransportType
	host       string
	port       int
	lib        *library.NotebookLibrary
	sessions   *session.Manager
	authMgr    *auth.Manager
	browser    browser.Manager
	notebooklm *notebooklm.Controller
	toolReg    *tools.ToolRegistry
	resReg     *resources.Registry
	logger     *utils.Logger
	httpServer *mcpserver.StreamableHTTPServer
	mu         sync.Mutex
}

// New creates a new MCPServer with all components wired together.
func New(
	cfg config.Config,
	lib *library.NotebookLibrary,
	sessions *session.Manager,
	authMgr *auth.Manager,
	browser browser.Manager,
	notebooklm *notebooklm.Controller,
	toolReg *tools.ToolRegistry,
	resReg *resources.Registry,
	logger *utils.Logger,
	transport TransportType,
	host string,
	port int,
) *MCPServer {
	// Create the mcp-go server
	srv := mcpserver.NewMCPServer(
		"notebooklm-mcp-server",
		"1.0.0",
		mcpserver.WithInstructions("NotebookLM MCP Server — AI-powered notebook assistant"),
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithResourceCapabilities(true, true),
		mcpserver.WithLogging(),
	)

	return &MCPServer{
		cfg:        cfg,
		server:     srv,
		transport:  transport,
		host:       host,
		port:       port,
		lib:        lib,
		sessions:   sessions,
		authMgr:    authMgr,
		browser:    browser,
		notebooklm: notebooklm,
		toolReg:    toolReg,
		resReg:     resReg,
		logger:     logger,
	}
}

// RegisterTools registers all tools with the MCP server based on profile.
func (s *MCPServer) RegisterTools() {
	s.toolReg.RegisterAll()
	s.toolReg.RegisterWithServer(s.server)
}

// RegisterResources registers all resource handlers with the MCP server.
func (s *MCPServer) RegisterResources() {
	s.resReg.RegisterWithServer(s.server)
}

// Start starts the transport. It blocks until the transport stops.
func (s *MCPServer) Start(ctx context.Context) error {
	switch s.transport {
	case TransportStdio:
		return s.startStdio(ctx)
	case TransportHTTP:
		return s.startHTTP(ctx)
	default:
		return fmt.Errorf("unknown transport: %s", s.transport)
	}
}

// startStdio starts the stdio transport.
func (s *MCPServer) startStdio(ctx context.Context) error {
	s.logger.Info("Starting stdio transport")

	err := mcpserver.ServeStdio(s.server, mcpserver.WithErrorLogger(
		s.logger.StdLogger(utils.LevelError),
	))
	if err != nil {
		return fmt.Errorf("stdio server: %w", err)
	}
	return nil
}

// startHTTP starts the Streamable HTTP transport.
func (s *MCPServer) startHTTP(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	s.logger.Info("Starting HTTP transport", "addr", addr)

	// Create the streamable HTTP server
	httpSrv := mcpserver.NewStreamableHTTPServer(s.server,
		mcpserver.WithEndpointPath("/mcp"),
		mcpserver.WithStateful(true),
	)
	s.mu.Lock()
	s.httpServer = httpSrv
	s.mu.Unlock()

	// Create a custom mux for additional routes
	mux := http.NewServeMux()

	// POST /mcp, GET /mcp, DELETE /mcp — handled by StreamableHTTPServer
	mux.Handle("/mcp", httpSrv)

	// GET /healthz — liveness check
	mux.HandleFunc("/healthz", s.handleHealthz)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("HTTP server listening", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http server: %w", err)
		}
		close(errCh)
	}()

	// Wait for context cancellation
	select {
	case <-ctx.Done():
		s.logger.Info("Shutting down HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("HTTP shutdown error", "error", err)
		}
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// handleHealthz serves the liveness check endpoint.
func (s *MCPServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	authValid := false
	if err := s.authMgr.Validate(time.Now()); err == nil {
		authValid = true
	}

	stats := s.sessions.Stats()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"status":         "ok",
		"authenticated":  authValid,
		"active_sessions": stats.Active,
	})
}

// Stop gracefully shuts down the server.
func (s *MCPServer) Stop() error {
	s.logger.Info("Stopping server")

	s.mu.Lock()
	httpSrv := s.httpServer
	s.mu.Unlock()

	if httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpSrv.Shutdown(ctx)
	}
	return nil
}

// Run starts the server and waits for shutdown signals.
func (s *MCPServer) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	// Wait for signal or error
	select {
	case <-ctx.Done():
		s.logger.Info("Received shutdown signal")
		// Graceful shutdown: close sessions → close browser → stop server
		s.gracefulShutdown()
		// Wait for server to stop
		<-errCh
		return nil
	case err := <-errCh:
		return err
	}
}

// gracefulShutdown performs ordered shutdown: sessions → browser → server.
func (s *MCPServer) gracefulShutdown() {
	s.logger.Info("Graceful shutdown: closing sessions")

	// Phase 1: Close all sessions
	if err := s.sessions.CloseAll(); err != nil {
		s.logger.Error("Error closing sessions", "error", err)
	}

	// Phase 2: Close browser
	if err := s.browser.Close(); err != nil {
		s.logger.Error("Error closing browser", "error", err)
	}

	// Phase 3: Stop HTTP server if running
	s.mu.Lock()
	httpSrv := s.httpServer
	s.mu.Unlock()

	if httpSrv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Error shutting down HTTP server", "error", err)
		}
	}

	s.logger.Info("Shutdown complete")
}

// Server returns the underlying mcp-go server for advanced use.
func (s *MCPServer) Server() *mcpserver.MCPServer {
	return s.server
}

// PrintBanner prints the startup banner to stderr.
func (s *MCPServer) PrintBanner() {
	transportStr := string(s.transport)
	if s.transport == TransportHTTP {
		transportStr = fmt.Sprintf("http://%s:%d", s.host, s.port)
	}

	headlessStr := "true"
	if !s.cfg.Headless {
		headlessStr = "false"
	}

	stealthStr := "disabled"
	if s.cfg.StealthEnabled {
		stealthStr = "enabled"
	}

	authStr := "not configured"
	if err := s.authMgr.Validate(time.Now()); err == nil {
		authStr = "valid"
	} else {
		// Check if it's expired vs not configured
		authStr = "expired"
	}

	libStats := s.lib.Stats()
	notebookCount := libStats.TotalNotebooks
	activeNotebook := "none"
	if libStats.ActiveNotebook != nil {
		activeNotebook = *libStats.ActiveNotebook
	}

	profileStr := string(s.toolReg.Profile())
	if profileStr == "" {
		profileStr = "standard"
	}

	banner := fmt.Sprintf(`
NotebookLM MCP Server
  Transport:  %s
  Headless:   %s
  Stealth:    %s
  Auth:       %s
  Library:    %d notebooks (active: %s)
  Profile:    %s
`,
		transportStr, headlessStr, stealthStr, authStr,
		notebookCount, activeNotebook, profileStr)

	s.logger.Info(banner)
}
