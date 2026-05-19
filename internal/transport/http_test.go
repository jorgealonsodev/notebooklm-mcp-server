package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

func TestServeStreamableHTTP_Initialization(t *testing.T) {
	srv := mcpserver.NewMCPServer("test", "1.0.0")

	// Use httptest to test the HTTP handler directly
	handler := mcpserver.NewStreamableHTTPServer(srv,
		mcpserver.WithEndpointPath("/mcp"),
		mcpserver.WithStateful(true),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Send an initialize request
	initReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	body, _ := json.Marshal(initReq)
	resp, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("POST /mcp: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Verify session ID header
	sessionID := resp.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Error("expected Mcp-Session-Id header")
	}

	// Verify response body
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if result["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", result["jsonrpc"])
	}
}

func TestServeStreamableHTTP_Healthz(t *testing.T) {
	// Test the healthz endpoint through our MCPServer
	srv, _ := testComponents(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	srv.handleHealthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
}

func TestServeStreamableHTTP_BadMethod(t *testing.T) {
	srv := mcpserver.NewMCPServer("test", "1.0.0")
	handler := mcpserver.NewStreamableHTTPServer(srv,
		mcpserver.WithEndpointPath("/mcp"),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	// PUT is not supported
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/mcp", strings.NewReader("{}"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT /mcp: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("PUT status = %d, want 404", resp.StatusCode)
	}
}

func TestServeStreamableHTTP_WrongContentType(t *testing.T) {
	srv := mcpserver.NewMCPServer("test", "1.0.0")
	handler := mcpserver.NewStreamableHTTPServer(srv,
		mcpserver.WithEndpointPath("/mcp"),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Wrong content type
	resp, err := http.Post(ts.URL+"/mcp", "text/plain", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("POST /mcp: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestServeStreamableHTTP_ContextTimeout(t *testing.T) {
	srv := mcpserver.NewMCPServer("test", "1.0.0")
	handler := mcpserver.NewStreamableHTTPServer(srv,
		mcpserver.WithEndpointPath("/mcp"),
		mcpserver.WithStateful(true),
	)

	// Test that Shutdown works
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// The handler itself doesn't manage the HTTP server lifecycle,
	// so we test the shutdown path
	err := handler.Shutdown(ctx)
	if err != nil {
		t.Logf("Shutdown: %v (expected for no-op)", err)
	}
}
