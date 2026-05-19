package tools

import (
	"context"
	"testing"

	"github.com/jorge/notebooklm-mcp-server/internal/browser"
	"github.com/jorge/notebooklm-mcp-server/internal/config"
	"github.com/jorge/notebooklm-mcp-server/internal/library"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/playwright-community/playwright-go"
)

// mockBrowser implements browser.Manager for testing.
type mockBrowser struct {
	healthy bool
}

func (m *mockBrowser) Launch(ctx context.Context) error              { return nil }
func (m *mockBrowser) NewPage(ctx context.Context) (playwright.Page, error) { return nil, nil }
func (m *mockBrowser) Close() error                                  { return nil }
func (m *mockBrowser) Healthy() bool                                 { return m.healthy }

var _ browser.Manager = (*mockBrowser)(nil)

// Note: We can't easily mock session.Manager, auth.Manager, notebooklm.Controller
// without creating full mock types. For registry tests, we focus on registration,
// profile filtering, and disabled tools — which don't require calling handlers.

func newTestRegistry(t *testing.T) *ToolRegistry {
	t.Helper()
	libPath := t.TempDir() + "/library.json"
	lib, err := library.New(libPath)
	if err != nil {
		t.Fatalf("library.New: %v", err)
	}

	browser := &mockBrowser{healthy: true}

	r := NewRegistry(
		config.Load(),
		lib,
		nil, // sessions — not used in registration tests
		nil, // auth — not used in registration tests
		browser,
		nil, // notebooklm — not used in registration tests
	)
	return r
}

func TestRegistry_RegisterAll_Count(t *testing.T) {
	r := newTestRegistry(t)
	r.RegisterAll()

	// All 20 tools should be registered in the definitions map
	if len(r.definitions) != 20 {
		t.Errorf("RegisterAll: got %d definitions, want 20", len(r.definitions))
	}

	// All 20 handlers should be registered
	if len(r.handlers) != 20 {
		t.Errorf("RegisterAll: got %d handlers, want 20", len(r.handlers))
	}
}

func TestRegistry_ListTools_ProfileFiltering(t *testing.T) {
	r := newTestRegistry(t)
	r.RegisterAll()

	tests := []struct {
		name    string
		profile ToolProfile
		wantLen int
	}{
		{"minimal", ProfileMinimal, 4},
		{"standard", ProfileStandard, 16},
		{"full", ProfileFull, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r.SetProfile(tt.profile)
			tools := r.ListTools()
			if len(tools) != tt.wantLen {
				t.Errorf("ListTools(%s): got %d tools, want %d", tt.profile, len(tools), tt.wantLen)
			}
		})
	}
}

func TestRegistry_ListTools_DisabledExcluded(t *testing.T) {
	r := newTestRegistry(t)
	r.RegisterAll()
	r.SetProfile(ProfileFull)

	// Disable some tools
	r.DisableTools([]string{"setup_auth", "re_auth", "cleanup_data"})

	tools := r.ListTools()

	// Should have 20 - 3 = 17 tools
	if len(tools) != 17 {
		t.Errorf("ListTools with disabled: got %d tools, want 17", len(tools))
	}

	// Verify disabled tools are not in the list
	for _, tool := range tools {
		if tool.Name == "setup_auth" || tool.Name == "re_auth" || tool.Name == "cleanup_data" {
			t.Errorf("disabled tool %q should not be in list", tool.Name)
		}
	}
}

func TestRegistry_IsEnabled(t *testing.T) {
	r := newTestRegistry(t)
	r.RegisterAll()

	r.SetProfile(ProfileMinimal)

	tests := []struct {
		name string
		want bool
	}{
		{"ask_question", true},
		{"get_health", true},
		{"list_notebooks", true},
		{"get_notebook", true},
		{"add_notebook", false}, // not in minimal
		{"cleanup_data", false}, // not in minimal
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.IsEnabled(tt.name); got != tt.want {
				t.Errorf("IsEnabled(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestRegistry_IsEnabled_Disabled(t *testing.T) {
	r := newTestRegistry(t)
	r.RegisterAll()
	r.SetProfile(ProfileFull)

	// Disable a tool that's in the full profile
	r.DisableTools([]string{"ask_question"})

	if r.IsEnabled("ask_question") {
		t.Error("IsEnabled should return false for disabled tool")
	}
}

func TestRegistry_Definition(t *testing.T) {
	r := newTestRegistry(t)
	r.RegisterAll()

	def, ok := r.Definition("ask_question")
	if !ok {
		t.Fatal("Definition(ask_question) not found")
	}
	if def.Name != "ask_question" {
		t.Errorf("Definition name = %q, want %q", def.Name, "ask_question")
	}

	_, ok = r.Definition("nonexistent")
	if ok {
		t.Error("Definition(nonexistent) should not be found")
	}
}

func TestRegistry_HandleTool_Unknown(t *testing.T) {
	r := newTestRegistry(t)
	r.RegisterAll()

	req := mcp.CallToolRequest{}
	req.Params.Name = "nonexistent_tool"

	result, err := r.HandleTool(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleTool returned error: %v", err)
	}
	if !result.IsError {
		t.Error("HandleTool for unknown tool should return error result")
	}
}

func TestRegistry_HandleTool_Dispatch(t *testing.T) {
	r := newTestRegistry(t)
	r.RegisterAll()

	// Verify that known tools have handlers
	knownTools := []string{
		"ask_question", "add_notebook", "list_notebooks", "get_notebook",
		"get_health", "list_sessions", "close_session", "reset_session",
	}

	for _, name := range knownTools {
		if !r.HasHandler(name) {
			t.Errorf("missing handler for %s", name)
		}
	}
}

func TestRegistry_SetProfile(t *testing.T) {
	r := newTestRegistry(t)
	r.RegisterAll()

	r.SetProfile(ProfileMinimal)
	if r.profile != ProfileMinimal {
		t.Errorf("SetProfile: got %q, want %q", r.profile, ProfileMinimal)
	}

	r.SetProfile(ProfileFull)
	if r.profile != ProfileFull {
		t.Errorf("SetProfile: got %q, want %q", r.profile, ProfileFull)
	}
}

func TestRegistry_DisableTools(t *testing.T) {
	r := newTestRegistry(t)
	r.RegisterAll()

	r.DisableTools([]string{"tool_a", "tool_b"})

	if !r.disabled["tool_a"] {
		t.Error("tool_a should be disabled")
	}
	if !r.disabled["tool_b"] {
		t.Error("tool_b should be disabled")
	}
}
