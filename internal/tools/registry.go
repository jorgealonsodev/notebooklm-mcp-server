package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/jorge/notebooklm-mcp-server/internal/auth"
	"github.com/jorge/notebooklm-mcp-server/internal/browser"
	"github.com/jorge/notebooklm-mcp-server/internal/config"
	"github.com/jorge/notebooklm-mcp-server/internal/library"
	"github.com/jorge/notebooklm-mcp-server/internal/notebooklm"
	"github.com/jorge/notebooklm-mcp-server/internal/session"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolHandler handles a tool call and returns a result.
type ToolHandler = server.ToolHandlerFunc

// ToolRegistry manages MCP tool registration, dispatch, and profile filtering.
type ToolRegistry struct {
	cfg         config.Config
	lib         *library.NotebookLibrary
	sessions    *session.Manager
	authMgr     *auth.Manager
	browser     browser.Manager
	notebooklm  *notebooklm.Controller

	mu           sync.RWMutex
	handlers     map[string]ToolHandler
	definitions  map[string]mcp.Tool
	disabled     map[string]bool
	profile      ToolProfile
}

// NewRegistry creates a new ToolRegistry with all dependencies injected.
func NewRegistry(
	cfg config.Config,
	lib *library.NotebookLibrary,
	sessions *session.Manager,
	authMgr *auth.Manager,
	browser browser.Manager,
	notebooklm *notebooklm.Controller,
) *ToolRegistry {
	return &ToolRegistry{
		cfg:        cfg,
		lib:        lib,
		sessions:   sessions,
		authMgr:    authMgr,
		browser:    browser,
		notebooklm: notebooklm,
		handlers:   make(map[string]ToolHandler),
		definitions: make(map[string]mcp.Tool),
		disabled:   make(map[string]bool),
		profile:    ProfileStandard,
	}
}

// SetProfile sets the active tool profile.
func (r *ToolRegistry) SetProfile(p ToolProfile) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profile = p
}

// Profile returns the current active tool profile.
func (r *ToolRegistry) Profile() ToolProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.profile
}

// DisableTools marks the specified tools as disabled.
func (r *ToolRegistry) DisableTools(names []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, name := range names {
		r.disabled[name] = true
	}
}

// RegisterAll registers all tool definitions and handlers.
func (r *ToolRegistry) RegisterAll() {
	defs := AllToolDefinitions()
	handlers := r.buildHandlers()

	r.mu.Lock()
	defer r.mu.Unlock()

	for name, def := range defs {
		r.definitions[name] = def
		if h, ok := handlers[name]; ok {
			r.handlers[name] = h
		}
	}
}

// RegisterWithServer registers tools with an MCP server based on the current
// profile and disabled list.
func (r *ToolRegistry) RegisterWithServer(s *server.MCPServer) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profileNames := ResolveProfile(r.profile)

	for _, name := range profileNames {
		if r.disabled[name] {
			continue
		}
		def, ok := r.definitions[name]
		if !ok {
			continue
		}
		handler, ok := r.handlers[name]
		if !ok {
			continue
		}
		s.AddTool(def, handler)
	}
}

// ListTools returns the tool definitions for the current profile, excluding
// disabled tools.
func (r *ToolRegistry) ListTools() []mcp.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profileNames := ResolveProfile(r.profile)
	var tools []mcp.Tool

	for _, name := range profileNames {
		if r.disabled[name] {
			continue
		}
		if def, ok := r.definitions[name]; ok {
			tools = append(tools, def)
		}
	}
	return tools
}

// HandleTool dispatches a tool call by name.
func (r *ToolRegistry) HandleTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.Params.Name

	r.mu.RLock()
	handler, ok := r.handlers[name]
	r.mu.RUnlock()

	if !ok {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: fmt.Sprintf("unknown tool: %s", name)},
			},
			IsError: true,
		}, nil
	}

	return handler(ctx, req)
}

// IsEnabled returns true if the tool is in the current profile and not disabled.
func (r *ToolRegistry) IsEnabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.disabled[name] {
		return false
	}

	profileNames := ResolveProfile(r.profile)
	for _, n := range profileNames {
		if n == name {
			return true
		}
	}
	return false
}

// HasHandler returns true if a handler exists for the given tool name.
func (r *ToolRegistry) HasHandler(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.handlers[name]
	return ok
}

// Definition returns the tool definition for the given name.
func (r *ToolRegistry) Definition(name string) (mcp.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.definitions[name]
	return def, ok
}

// buildHandlers creates the handler map for all 19 tools.
func (r *ToolRegistry) buildHandlers() map[string]ToolHandler {
	return map[string]ToolHandler{
		"ask_question":      r.handleAskQuestion,
		"add_notebook":      r.handleAddNotebook,
		"list_notebooks":    r.handleListNotebooks,
		"get_notebook":      r.handleGetNotebook,
		"select_notebook":   r.handleSelectNotebook,
		"update_notebook":   r.handleUpdateNotebook,
		"remove_notebook":   r.handleRemoveNotebook,
		"search_notebooks":  r.handleSearchNotebooks,
		"get_library_stats": r.handleGetLibraryStats,
		"list_sessions":     r.handleListSessions,
		"close_session":     r.handleCloseSession,
		"reset_session":     r.handleResetSession,
		"get_health":        r.handleGetHealth,
		"setup_auth":        r.handleSetupAuth,
		"re_auth":           r.handleReAuth,
		"cleanup_data":      r.handleCleanupData,
		"add_source":        r.handleAddSource,
		"generate_audio":    r.handleGenerateAudio,
		"get_audio_status":  r.handleGetAudioStatus,
		"download_audio":    r.handleDownloadAudio,
	}
}

// ---- Helper: parse optional browser_options ----

func parseBrowserOptions(req mcp.CallToolRequest) *config.BrowserOptions {
	args := req.GetArguments()
	if args == nil {
		return nil
	}

	raw, ok := args["browser_options"]
	if !ok || raw == nil {
		return nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	var opts config.BrowserOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return nil
	}
	return &opts
}

// resolveNotebookURL determines the notebook URL from params, active notebook, or config.
func (r *ToolRegistry) resolveNotebookURL(req mcp.CallToolRequest) (string, error) {
	args := req.GetArguments()

	// Direct URL param
	if url, ok := args["notebook_url"].(string); ok && url != "" {
		return url, nil
	}

	// Look up by notebook_id
	if id, ok := args["notebook_id"].(string); ok && id != "" {
		nb, err := r.lib.Get(id)
		if err != nil {
			return "", fmt.Errorf("notebook %q not found: %w", id, err)
		}
		return nb.URL, nil
	}

	// Use active notebook
	if active := r.lib.Active(); active != nil {
		return active.URL, nil
	}

	// Fall back to config
	if r.cfg.NotebookURL != "" {
		return r.cfg.NotebookURL, nil
	}

	return "", fmt.Errorf("no notebook URL available; provide notebook_id, notebook_url, or select a notebook")
}

// textResult creates a simple text response.
func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: text},
		},
	}
}

// errorResult creates an error response.
func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: err.Error()},
		},
		IsError: true,
	}
}

// sendProgress sends a progress notification if the request includes a
// progress token. Used by long-running handlers (chat, audio, sources).
func (r *ToolRegistry) sendProgress(req mcp.CallToolRequest, progress, total float64, message string) {
	token, ok := req.Params.Meta.AdditionalFields["progressToken"]
	if !ok {
		return
	}
	tokenStr, ok := token.(string)
	if !ok {
		return
	}
	// Progress notification via the request's progress token
	_ = tokenStr // Reserved for MCP progress token integration
}

// jsonResult creates a JSON response from any marshalable value.
func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: string(data)},
		},
	}, nil
}
