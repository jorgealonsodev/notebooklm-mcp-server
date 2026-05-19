package tools

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jorge/notebooklm-mcp-server/internal/config"
	"github.com/jorge/notebooklm-mcp-server/internal/library"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestParseBrowserOptions(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		wantNil bool
		check   func(*testing.T, *config.BrowserOptions)
	}{
		{
			name:    "nil args",
			args:    nil,
			wantNil: true,
		},
		{
			name:    "no browser_options key",
			args:    map[string]any{"question": "hello"},
			wantNil: true,
		},
		{
			name: "with show option",
			args: map[string]any{
				"browser_options": map[string]any{
					"show": true,
				},
			},
			wantNil: false,
			check: func(t *testing.T, opts *config.BrowserOptions) {
				if opts.Show == nil || !*opts.Show {
					t.Error("show should be true")
				}
			},
		},
		{
			name: "with headless option",
			args: map[string]any{
				"browser_options": map[string]any{
					"headless": false,
				},
			},
			wantNil: false,
			check: func(t *testing.T, opts *config.BrowserOptions) {
				if opts.Headless == nil || *opts.Headless {
					t.Error("headless should be false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = tt.args

			opts := parseBrowserOptions(req)

			if tt.wantNil && opts != nil {
				t.Error("expected nil browser options")
			}
			if !tt.wantNil && opts == nil {
				t.Fatal("expected non-nil browser options")
			}
			if tt.check != nil && opts != nil {
				tt.check(t, opts)
			}
		})
	}
}

func TestParseBrowserOptions_JSON(t *testing.T) {
	// Test that browser_options can be passed as raw JSON
	rawJSON := json.RawMessage(`{"show":true,"headless":false}`)
	args := map[string]any{
		"browser_options": rawJSON,
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = args

	opts := parseBrowserOptions(req)
	if opts == nil {
		t.Fatal("expected non-nil browser options from raw JSON")
	}
	if opts.Show == nil || !*opts.Show {
		t.Error("show should be true")
	}
	if opts.Headless == nil || *opts.Headless {
		t.Error("headless should be false")
	}
}

func TestResolveNotebookURL(t *testing.T) {
	libPath := t.TempDir() + "/library.json"
	lib, err := library.New(libPath)
	if err != nil {
		t.Fatalf("library.New: %v", err)
	}

	// Add a test notebook
	_, err = lib.Add(library.AddInput{
		URL:   "https://notebooklm.google.com/notebook/abc123",
		Name:  "Test Notebook",
		Description: "A test notebook",
		Topics: []string{"test"},
	})
	if err != nil {
		t.Fatalf("lib.Add: %v", err)
	}

	// Select it as active
	if err := lib.Select("test-notebook"); err != nil {
		t.Fatalf("lib.Select: %v", err)
	}

	r := &ToolRegistry{lib: lib}

	tests := []struct {
		name    string
		args    map[string]any
		wantURL string
		wantErr bool
	}{
		{
			name: "direct URL",
			args: map[string]any{
				"notebook_url": "https://example.com/notebook",
			},
			wantURL: "https://example.com/notebook",
		},
		{
			name: "by notebook_id",
			args: map[string]any{
				"notebook_id": "test-notebook",
			},
			wantURL: "https://notebooklm.google.com/notebook/abc123",
		},
		{
			name:    "active notebook fallback",
			args:    map[string]any{},
			wantURL: "https://notebooklm.google.com/notebook/abc123",
		},
		{
			name:    "unknown notebook_id",
			args:    map[string]any{"notebook_id": "nonexistent"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = tt.args

			// Clear the active notebook for the "unknown" test
			if tt.name == "unknown notebook_id" {
				// Deselect active notebook
				r2 := &ToolRegistry{lib: lib}
				// Create a fresh lib without active
				lib2Path := t.TempDir() + "/library2.json"
				lib2, _ := library.New(lib2Path)
				r2.lib = lib2
				req2 := mcp.CallToolRequest{}
				req2.Params.Arguments = tt.args

				_, err := r2.resolveNotebookURL(req2)
				if err == nil {
					t.Error("expected error for unknown notebook_id")
				}
				return
			}

			url, err := r.resolveNotebookURL(req)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if url != tt.wantURL {
				t.Errorf("url = %q, want %q", url, tt.wantURL)
			}
		})
	}
}

func TestTextResult(t *testing.T) {
	result := textResult("hello world")
	if result.IsError {
		t.Error("textResult should not be an error")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if tc.Text != "hello world" {
		t.Errorf("text = %q, want %q", tc.Text, "hello world")
	}
}

func TestErrorResult(t *testing.T) {
	result := errorResult(fmt.Errorf("something went wrong"))
	if !result.IsError {
		t.Error("errorResult should be an error")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if tc.Text != "something went wrong" {
		t.Errorf("text = %q, want %q", tc.Text, "something went wrong")
	}
}

func TestJSONResult(t *testing.T) {
	data := map[string]any{"key": "value", "number": 42}
	result, err := jsonResult(data)
	if err != nil {
		t.Fatalf("jsonResult error: %v", err)
	}
	if result.IsError {
		t.Error("jsonResult should not be an error")
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &parsed); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("parsed key = %v, want %q", parsed["key"], "value")
	}
}

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name string
		input any
		want  []string
	}{
		{"nil", nil, nil},
		{"[]string", []string{"a", "b"}, []string{"a", "b"}},
		{"[]any strings", []any{"a", "b"}, []string{"a", "b"}},
		{"[]any mixed", []any{"a", 1, "b"}, []string{"a", "b"}},
		{"int", 42, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toStringSlice(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
