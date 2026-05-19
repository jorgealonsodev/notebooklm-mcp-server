package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jorge/notebooklm-mcp-server/internal/library"
	"github.com/jorge/notebooklm-mcp-server/internal/notebooklm"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/playwright-community/playwright-go"
)

// ---- Chat ----

func (r *ToolRegistry) handleAskQuestion(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	question, err := req.RequireString("question")
	if err != nil {
		return errorResult(err), nil
	}

	notebookURL, err := r.resolveNotebookURL(req)
	if err != nil {
		return errorResult(err), nil
	}

	r.sendProgress(req, 0, 1, "Creating session...")

	// Create or get a session
	sess, err := r.sessions.Create(notebookURL)
	if err != nil {
		return errorResult(fmt.Errorf("create session: %w", err)), nil
	}

	page, ok := sess.Page.(playwright.Page)
	if !ok {
		return errorResult(fmt.Errorf("session page is not available")), nil
	}

	r.sendProgress(req, 0.3, 1, "Asking question...")

	// Ask the question
	result, err := r.notebooklm.Ask(ctx, page, question)
	if err != nil {
		return errorResult(fmt.Errorf("ask question: %w", err)), nil
	}

	r.sendProgress(req, 0.8, 1, "Formatting answer...")

	// Format with citations if requested
	format := notebooklm.CitationFormat(req.GetString("source_format", "none"))
	answer := result.Answer

	if format != "none" {
		// Convert []Citation to []FormattedCitation for formatting
		var formatted []notebooklm.FormattedCitation
		for i, c := range result.Citations {
			formatted = append(formatted, notebooklm.FormattedCitation{
				Index:    i + 1,
				Quote:    c.Quote,
				SourceID: c.SourceURL,
			})
		}
		answer = notebooklm.FormatCitations(answer, formatted, format)
	}

	return textResult(answer), nil
}

// ---- Library ----

func (r *ToolRegistry) handleAddNotebook(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := req.RequireString("url")
	if err != nil {
		return errorResult(err), nil
	}
	name, err := req.RequireString("name")
	if err != nil {
		return errorResult(err), nil
	}
	description, err := req.RequireString("description")
	if err != nil {
		return errorResult(err), nil
	}
	topics, err := req.RequireStringSlice("topics")
	if err != nil {
		return errorResult(err), nil
	}

	contentTypes := req.GetStringSlice("content_types", nil)
	useCases := req.GetStringSlice("use_cases", nil)
	tags := req.GetStringSlice("tags", nil)

	entry, err := r.lib.Add(library.AddInput{
		URL:          url,
		Name:         name,
		Description:  description,
		Topics:       topics,
		ContentTypes: contentTypes,
		UseCases:     useCases,
		Tags:         tags,
	})
	if err != nil {
		return errorResult(err), nil
	}

	return jsonResult(entry)
}

func (r *ToolRegistry) handleListNotebooks(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	notebooks := r.lib.List()
	return jsonResult(notebooks)
}

func (r *ToolRegistry) handleGetNotebook(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return errorResult(err), nil
	}

	nb, err := r.lib.Get(id)
	if err != nil {
		return errorResult(err), nil
	}

	return jsonResult(nb)
}

func (r *ToolRegistry) handleSelectNotebook(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return errorResult(err), nil
	}

	if err := r.lib.Select(id); err != nil {
		return errorResult(err), nil
	}

	return textResult(fmt.Sprintf("Notebook %q selected", id)), nil
}

func (r *ToolRegistry) handleUpdateNotebook(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return errorResult(err), nil
	}

	var in library.UpdateInput
	in.ID = id

	if name, err := req.RequireString("name"); err == nil {
		in.Name = &name
	}
	if desc, err := req.RequireString("description"); err == nil {
		in.Description = &desc
	}
	if url, err := req.RequireString("url"); err == nil {
		in.URL = &url
	}

	args := req.GetArguments()
	if topics, ok := args["topics"]; ok {
		in.Topics = toStringSlice(topics)
	}
	if contentTypes, ok := args["content_types"]; ok {
		in.ContentTypes = toStringSlice(contentTypes)
	}
	if useCases, ok := args["use_cases"]; ok {
		in.UseCases = toStringSlice(useCases)
	}
	if tags, ok := args["tags"]; ok {
		in.Tags = toStringSlice(tags)
	}

	entry, err := r.lib.Update(in)
	if err != nil {
		return errorResult(err), nil
	}

	return jsonResult(entry)
}

func (r *ToolRegistry) handleRemoveNotebook(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return errorResult(err), nil
	}

	if err := r.lib.Remove(id); err != nil {
		return errorResult(err), nil
	}

	return textResult(fmt.Sprintf("Notebook %q removed", id)), nil
}

func (r *ToolRegistry) handleSearchNotebooks(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return errorResult(err), nil
	}

	results := r.lib.Search(query)
	return jsonResult(results)
}

func (r *ToolRegistry) handleGetLibraryStats(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stats := r.lib.Stats()
	return jsonResult(stats)
}

// ---- Session ----

func (r *ToolRegistry) handleListSessions(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessions := r.sessions.List()
	return jsonResult(sessions)
}

func (r *ToolRegistry) handleCloseSession(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return errorResult(err), nil
	}

	if err := r.sessions.Close(sessionID); err != nil {
		return errorResult(err), nil
	}

	return textResult(fmt.Sprintf("Session %q closed", sessionID)), nil
}

func (r *ToolRegistry) handleResetSession(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return errorResult(err), nil
	}

	if err := r.sessions.Reset(sessionID); err != nil {
		return errorResult(err), nil
	}

	return textResult(fmt.Sprintf("Session %q reset", sessionID)), nil
}

// ---- System ----

func (r *ToolRegistry) handleGetHealth(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	healthy := r.browser.Healthy()
	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}

	result := map[string]any{
		"status":  status,
		"browser": healthy,
	}
	return jsonResult(result)
}

func (r *ToolRegistry) handleSetupAuth(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Interactive setup requires a real browser — return placeholder for now
	return textResult("Interactive auth setup requires a headful browser. Use the browser to complete Google login, then cookies will be saved automatically."), nil
}

func (r *ToolRegistry) handleReAuth(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Re-auth is same as setup
	return r.handleSetupAuth(ctx, req)
}

func (r *ToolRegistry) handleCleanupData(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	confirm, err := req.RequireBool("confirm")
	if err != nil {
		return errorResult(err), nil
	}
	if !confirm {
		return errorResult(fmt.Errorf("cleanup requires confirm=true")), nil
	}

	preserveLibrary := req.GetBool("preserve_library", false)

	// Clean auth data
	if err := r.authMgr.ClearAllAuthData(); err != nil {
		return errorResult(fmt.Errorf("cleanup auth: %w", err)), nil
	}

	// Clean browser profile
	if err := os.RemoveAll(r.cfg.ChromeProfileDir); err != nil {
		return errorResult(fmt.Errorf("cleanup browser: %w", err)), nil
	}

	// Clean instances
	if err := os.RemoveAll(r.cfg.ChromeInstancesDir); err != nil {
		return errorResult(fmt.Errorf("cleanup instances: %w", err)), nil
	}

	// Clean browser state
	if err := os.RemoveAll(r.cfg.BrowserStateDir); err != nil {
		return errorResult(fmt.Errorf("cleanup state: %w", err)), nil
	}

	// Optionally clean library
	if !preserveLibrary {
		libPath := filepath.Join(r.cfg.DataDir, "library.json")
		if err := os.Remove(libPath); err != nil && !os.IsNotExist(err) {
			return errorResult(fmt.Errorf("cleanup library: %w", err)), nil
		}
	}

	return textResult("Data cleanup completed successfully"), nil
}

// ---- Sources ----

func (r *ToolRegistry) handleAddSource(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceType, err := req.RequireString("type")
	if err != nil {
		return errorResult(err), nil
	}
	content, err := req.RequireString("content")
	if err != nil {
		return errorResult(err), nil
	}

	title := req.GetString("title", "")

	notebookURL, err := r.resolveNotebookURL(req)
	if err != nil {
		return errorResult(err), nil
	}

	r.sendProgress(req, 0, 1, "Creating session...")

	// Create or get a session
	sess, err := r.sessions.Create(notebookURL)
	if err != nil {
		return errorResult(fmt.Errorf("create session: %w", err)), nil
	}

	page, ok := sess.Page.(playwright.Page)
	if !ok {
		return errorResult(fmt.Errorf("session page is not available")), nil
	}

	r.sendProgress(req, 0.3, 1, "Adding source...")

	result, err := r.notebooklm.AddSource(page, sourceType, content, title)
	if err != nil {
		return errorResult(fmt.Errorf("add source: %w", err)), nil
	}

	r.sendProgress(req, 1, 1, "Source added")

	return jsonResult(result)
}

// ---- Audio ----

func (r *ToolRegistry) handleGenerateAudio(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	notebookURL, err := r.resolveNotebookURL(req)
	if err != nil {
		return errorResult(err), nil
	}

	r.sendProgress(req, 0, 1, "Creating session...")

	// Create or get a session
	sess, err := r.sessions.Create(notebookURL)
	if err != nil {
		return errorResult(fmt.Errorf("create session: %w", err)), nil
	}

	page, ok := sess.Page.(playwright.Page)
	if !ok {
		return errorResult(fmt.Errorf("session page is not available")), nil
	}

	customPrompt := req.GetString("custom_prompt", "")
	timeoutMs := req.GetInt("timeout_ms", 0)
	waitForCompletion := req.GetBool("wait_for_completion", false)

	if waitForCompletion && timeoutMs == 0 {
		timeoutMs = r.cfg.AnswerTimeoutMs
	}

	r.sendProgress(req, 0.2, 1, "Generating audio overview...")

	result, err := r.notebooklm.GenerateAudio(page, customPrompt, timeoutMs)
	if err != nil {
		return errorResult(fmt.Errorf("generate audio: %w", err)), nil
	}

	if waitForCompletion {
		r.sendProgress(req, 0.5, 1, "Waiting for audio generation to complete...")
	}

	r.sendProgress(req, 1, 1, "Audio generation triggered")

	return jsonResult(result)
}

func (r *ToolRegistry) handleGetAudioStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	notebookURL, err := r.resolveNotebookURL(req)
	if err != nil {
		return errorResult(err), nil
	}

	// Create or get a session
	sess, err := r.sessions.Create(notebookURL)
	if err != nil {
		return errorResult(fmt.Errorf("create session: %w", err)), nil
	}

	page, ok := sess.Page.(playwright.Page)
	if !ok {
		return errorResult(fmt.Errorf("session page is not available")), nil
	}

	status, err := r.notebooklm.GetAudioStatus(page)
	if err != nil {
		return errorResult(fmt.Errorf("get audio status: %w", err)), nil
	}

	return jsonResult(map[string]any{
		"status": string(status),
	})
}

func (r *ToolRegistry) handleDownloadAudio(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	destDir, err := req.RequireString("destination_dir")
	if err != nil {
		return errorResult(err), nil
	}

	notebookURL, err := r.resolveNotebookURL(req)
	if err != nil {
		return errorResult(err), nil
	}

	// Create or get a session
	sess, err := r.sessions.Create(notebookURL)
	if err != nil {
		return errorResult(fmt.Errorf("create session: %w", err)), nil
	}

	page, ok := sess.Page.(playwright.Page)
	if !ok {
		return errorResult(fmt.Errorf("session page is not available")), nil
	}

	result, err := r.notebooklm.DownloadAudio(page, destDir)
	if err != nil {
		return errorResult(fmt.Errorf("download audio: %w", err)), nil
	}

	return jsonResult(result)
}

// ---- Helpers ----

func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}
