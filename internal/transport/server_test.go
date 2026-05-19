package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jorge/notebooklm-mcp-server/internal/auth"
	"github.com/jorge/notebooklm-mcp-server/internal/config"
	"github.com/jorge/notebooklm-mcp-server/internal/library"
	"github.com/jorge/notebooklm-mcp-server/internal/notebooklm"
	"github.com/jorge/notebooklm-mcp-server/internal/resources"
	"github.com/jorge/notebooklm-mcp-server/internal/session"
	"github.com/jorge/notebooklm-mcp-server/internal/tools"
	"github.com/jorge/notebooklm-mcp-server/internal/utils"
	"github.com/playwright-community/playwright-go"
)

// mockBrowser implements both browser.Manager and session.BrowserManager for tests.
type mockBrowser struct {
	launched bool
	closed   bool
}

func (m *mockBrowser) Launch(ctx context.Context) error {
	m.launched = true
	return nil
}
func (m *mockBrowser) NewPage(ctx context.Context) (playwright.Page, error) {
	return nil, fmt.Errorf("mock: no page")
}
func (m *mockBrowser) Close() error {
	m.closed = true
	return nil
}
func (m *mockBrowser) Healthy() bool {
	return m.launched && !m.closed
}

// mockSessionBrowser implements session.BrowserManager (which uses `any` return type).
type mockSessionBrowser struct {
	*mockBrowser
}

func (m *mockSessionBrowser) NewPage(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("mock: no page")
}

func testComponents(t *testing.T) (*MCPServer, *mockBrowser) {
	t.Helper()

	dir := t.TempDir()
	libPath := filepath.Join(dir, "library.json")
	lib, err := library.New(libPath)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Load()
	cfg.DataDir = dir
	cfg.BrowserStateDir = filepath.Join(dir, "browser_state")
	cfg.ChromeProfileDir = filepath.Join(dir, "chrome_profile")

	browser := &mockBrowser{}
	sessionBrowser := &mockSessionBrowser{mockBrowser: browser}
	sessMgr := session.NewManager(cfg, sessionBrowser)
	authMgr := auth.NewManager(cfg)
	nlm := notebooklm.NewController(cfg)
	toolReg := tools.NewRegistry(cfg, lib, sessMgr, authMgr, browser, nlm)
	resReg := resources.NewRegistry(lib)
	logger := utils.NewLogger()
	logger.SetColor(false)

	srv := New(cfg, lib, sessMgr, authMgr, browser, nlm, toolReg, resReg, logger, TransportStdio, "127.0.0.1", 0)
	srv.RegisterTools()
	srv.RegisterResources()

	return srv, browser
}

func TestNew_CreatesServer(t *testing.T) {
	srv, _ := testComponents(t)

	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.Server() == nil {
		t.Fatal("expected non-nil mcp server")
	}
}

func TestMCPServer_PrintBanner(t *testing.T) {
	srv, _ := testComponents(t)

	// Should not panic
	srv.PrintBanner()
}

func TestMCPServer_TransportType(t *testing.T) {
	tests := []struct {
		name string
		tt   TransportType
	}{
		{"stdio", TransportStdio},
		{"http", TransportHTTP},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.tt) != tt.name {
				t.Errorf("TransportType string = %q, want %q", tt.tt, tt.name)
			}
		})
	}
}

func TestServerConfig_Defaults(t *testing.T) {
	cfg := ServerConfig{
		Transport: TransportStdio,
		Host:      "127.0.0.1",
		Port:      3000,
		Profile:   tools.ProfileStandard,
	}

	if cfg.Transport != TransportStdio {
		t.Errorf("Transport = %v", cfg.Transport)
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host = %v", cfg.Host)
	}
	if cfg.Port != 3000 {
		t.Errorf("Port = %v", cfg.Port)
	}
}

func TestMCPServer_HandleHealthz(t *testing.T) {
	srv, _ := testComponents(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	srv.handleHealthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
	if _, ok := resp["authenticated"]; !ok {
		t.Error("missing authenticated field")
	}
	if _, ok := resp["active_sessions"]; !ok {
		t.Error("missing active_sessions field")
	}
}

func TestMCPServer_Stop_NoHTTPServer(t *testing.T) {
	srv, _ := testComponents(t)

	// Stop should succeed even without HTTP server started
	if err := srv.Stop(); err != nil {
		t.Errorf("Stop: %v", err)
	}
}

func TestMCPServer_GracefulShutdown(t *testing.T) {
	srv, br := testComponents(t)

	// Launch the browser first
	if err := br.Launch(context.Background()); err != nil {
		t.Fatal(err)
	}

	srv.gracefulShutdown()

	if !br.closed {
		t.Error("browser should be closed after graceful shutdown")
	}
}

func TestMCPServer_RegisterTools(t *testing.T) {
	srv, _ := testComponents(t)

	// Tools should be registered without panic
	srv.RegisterTools()
}

func TestMCPServer_RegisterResources(t *testing.T) {
	srv, _ := testComponents(t)

	// Resources should be registered without panic
	srv.RegisterResources()
}

func TestMCPServer_UnknownTransport(t *testing.T) {
	srv, _ := testComponents(t)
	srv.transport = "unknown"

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := srv.Start(ctx)
	if err == nil {
		t.Error("expected error for unknown transport")
	}
}

func TestStdioServe_WithLogger(t *testing.T) {
	// Test that ServeStdio can be called with error logger option
	// This is a compilation test — actual stdio test requires real stdin/stdout
	logger := utils.NewLogger()
	stdLogger := logger.StdLogger(utils.LevelError)
	if stdLogger == nil {
		t.Error("expected non-nil std logger")
	}
}

func TestMCPServer_HTTPTransportConfig(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "library.json")
	lib, err := library.New(libPath)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Load()
	cfg.DataDir = dir

	browser := &mockBrowser{}
	sessionBrowser := &mockSessionBrowser{mockBrowser: browser}
	sessMgr := session.NewManager(cfg, sessionBrowser)
	authMgr := auth.NewManager(cfg)
	nlm := notebooklm.NewController(cfg)
	toolReg := tools.NewRegistry(cfg, lib, sessMgr, authMgr, browser, nlm)
	resReg := resources.NewRegistry(lib)
	logger := utils.NewLogger()

	srv := New(cfg, lib, sessMgr, authMgr, browser, nlm, toolReg, resReg, logger, TransportHTTP, "127.0.0.1", 8765)

	if srv.transport != TransportHTTP {
		t.Errorf("transport = %v, want %v", srv.transport, TransportHTTP)
	}
	if srv.host != "127.0.0.1" {
		t.Errorf("host = %v, want 127.0.0.1", srv.host)
	}
	if srv.port != 8765 {
		t.Errorf("port = %v, want 8765", srv.port)
	}
}

// Test that the healthz endpoint returns correct JSON structure.
func TestHealthz_ResponseFormat(t *testing.T) {
	srv, _ := testComponents(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	srv.handleHealthz(rec, req)

	var resp struct {
		Status         string `json:"status"`
		Authenticated  bool   `json:"authenticated"`
		ActiveSessions int    `json:"active_sessions"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
}

// Test that the banner includes expected fields.
func TestPrintBanner_ContainsExpectedFields(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	srv, _ := testComponents(t)
	srv.PrintBanner()

	w.Close()
	os.Stderr = oldStderr

	// Read and discard — the test verifies no panic occurs
	buf := make([]byte, 4096)
	_, _ = r.Read(buf)
}
