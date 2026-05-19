package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// AllToolDefinitions returns a map of tool name → mcp.Tool for all 19 tools.
func AllToolDefinitions() map[string]mcp.Tool {
	return map[string]mcp.Tool{
		"ask_question":     defAskQuestion(),
		"add_notebook":     defAddNotebook(),
		"list_notebooks":   defListNotebooks(),
		"get_notebook":     defGetNotebook(),
		"select_notebook":  defSelectNotebook(),
		"update_notebook":  defUpdateNotebook(),
		"remove_notebook":  defRemoveNotebook(),
		"search_notebooks": defSearchNotebooks(),
		"get_library_stats": defGetLibraryStats(),
		"list_sessions":    defListSessions(),
		"close_session":    defCloseSession(),
		"reset_session":    defResetSession(),
		"get_health":       defGetHealth(),
		"setup_auth":       defSetupAuth(),
		"re_auth":          defReAuth(),
		"cleanup_data":     defCleanupData(),
		"add_source":       defAddSource(),
		"generate_audio":   defGenerateAudio(),
		"get_audio_status": defGetAudioStatus(),
		"download_audio":   defDownloadAudio(),
	}
}

func defAskQuestion() mcp.Tool {
	return mcp.NewTool("ask_question",
		mcp.WithDescription("Ask a question to NotebookLM and get an answer with optional citations."),
		mcp.WithString("question",
			mcp.Required(),
			mcp.Description("The question to ask."),
		),
		mcp.WithString("session_id",
			mcp.Description("Optional session ID to use."),
		),
		mcp.WithString("notebook_id",
			mcp.Description("Optional notebook ID to use."),
		),
		mcp.WithString("notebook_url",
			mcp.Description("Optional notebook URL to use."),
		),
		mcp.WithString("source_format",
			mcp.Description("Citation source format."),
			mcp.Enum("none", "inline", "footnotes", "json"),
		),
		mcp.WithBoolean("show_browser",
			mcp.Description("Show browser window."),
		),
		mcp.WithObject("browser_options",
			mcp.Description("Optional browser configuration overrides."),
		),
	)
}

func defAddNotebook() mcp.Tool {
	return mcp.NewTool("add_notebook",
		mcp.WithDescription("Add a new notebook to the library."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("NotebookLM URL."),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Display name for the notebook."),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Description of the notebook."),
		),
		mcp.WithArray("topics",
			mcp.Required(),
			mcp.Description("Topics covered by the notebook."),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithArray("content_types",
			mcp.Description("Types of content in the notebook."),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithArray("use_cases",
			mcp.Description("Use cases for the notebook."),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithArray("tags",
			mcp.Description("Tags for organizing."),
			mcp.Items(map[string]any{"type": "string"}),
		),
	)
}

func defListNotebooks() mcp.Tool {
	return mcp.NewTool("list_notebooks",
		mcp.WithDescription("List all notebooks in the library."),
	)
}

func defGetNotebook() mcp.Tool {
	return mcp.NewTool("get_notebook",
		mcp.WithDescription("Get details of a specific notebook."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Notebook ID."),
		),
	)
}

func defSelectNotebook() mcp.Tool {
	return mcp.NewTool("select_notebook",
		mcp.WithDescription("Select a notebook as the active notebook."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Notebook ID to select."),
		),
	)
}

func defUpdateNotebook() mcp.Tool {
	return mcp.NewTool("update_notebook",
		mcp.WithDescription("Update fields of an existing notebook."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Notebook ID to update."),
		),
		mcp.WithString("name",
			mcp.Description("New name."),
		),
		mcp.WithString("description",
			mcp.Description("New description."),
		),
		mcp.WithArray("topics",
			mcp.Description("New topics."),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithArray("content_types",
			mcp.Description("New content types."),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithArray("use_cases",
			mcp.Description("New use cases."),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithArray("tags",
			mcp.Description("New tags."),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithString("url",
			mcp.Description("New URL."),
		),
	)
}

func defRemoveNotebook() mcp.Tool {
	return mcp.NewTool("remove_notebook",
		mcp.WithDescription("Remove a notebook from the library."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Notebook ID to remove."),
		),
	)
}

func defSearchNotebooks() mcp.Tool {
	return mcp.NewTool("search_notebooks",
		mcp.WithDescription("Search notebooks by name, description, or topics."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query string."),
		),
	)
}

func defGetLibraryStats() mcp.Tool {
	return mcp.NewTool("get_library_stats",
		mcp.WithDescription("Get aggregate library statistics."),
	)
}

func defListSessions() mcp.Tool {
	return mcp.NewTool("list_sessions",
		mcp.WithDescription("List all active browser sessions."),
	)
}

func defCloseSession() mcp.Tool {
	return mcp.NewTool("close_session",
		mcp.WithDescription("Close a browser session."),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Session ID to close."),
		),
	)
}

func defResetSession() mcp.Tool {
	return mcp.NewTool("reset_session",
		mcp.WithDescription("Reset a browser session (reload page, clear messages)."),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Session ID to reset."),
		),
	)
}

func defGetHealth() mcp.Tool {
	return mcp.NewTool("get_health",
		mcp.WithDescription("Check server health status."),
	)
}

func defSetupAuth() mcp.Tool {
	return mcp.NewTool("setup_auth",
		mcp.WithDescription("Set up authentication via interactive browser login."),
		mcp.WithBoolean("show_browser",
			mcp.Description("Show browser window."),
		),
		mcp.WithObject("browser_options",
			mcp.Description("Optional browser configuration overrides."),
		),
	)
}

func defReAuth() mcp.Tool {
	return mcp.NewTool("re_auth",
		mcp.WithDescription("Re-authenticate with Google (re-setup auth)."),
		mcp.WithBoolean("show_browser",
			mcp.Description("Show browser window."),
		),
		mcp.WithObject("browser_options",
			mcp.Description("Optional browser configuration overrides."),
		),
	)
}

func defCleanupData() mcp.Tool {
	return mcp.NewTool("cleanup_data",
		mcp.WithDescription("Clean up all authentication and browser data."),
		mcp.WithBoolean("confirm",
			mcp.Required(),
			mcp.Description("Must be true to confirm cleanup."),
		),
		mcp.WithBoolean("preserve_library",
			mcp.Description("If true, preserve the library file."),
		),
	)
}

func defAddSource() mcp.Tool {
	return mcp.NewTool("add_source",
		mcp.WithDescription("Add a source (URL or text) to a notebook."),
		mcp.WithString("type",
			mcp.Required(),
			mcp.Description("Source type: url or text."),
			mcp.Enum("url", "text"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Source content (URL or text body)."),
		),
		mcp.WithString("title",
			mcp.Description("Title for the source (used for text type)."),
		),
		mcp.WithString("session_id",
			mcp.Description("Session ID to use."),
		),
		mcp.WithString("notebook_id",
			mcp.Description("Notebook ID to add source to."),
		),
		mcp.WithString("notebook_url",
			mcp.Description("Notebook URL to add source to."),
		),
		mcp.WithBoolean("show_browser",
			mcp.Description("Show browser window."),
		),
	)
}

func defGenerateAudio() mcp.Tool {
	return mcp.NewTool("generate_audio",
		mcp.WithDescription("Generate an audio overview for a notebook."),
		mcp.WithString("custom_prompt",
			mcp.Description("Custom prompt for audio generation."),
		),
		mcp.WithNumber("timeout_ms",
			mcp.Description("Timeout in milliseconds for blocking wait."),
		),
		mcp.WithBoolean("wait_for_completion",
			mcp.Description("Block until audio generation completes."),
		),
		mcp.WithString("session_id",
			mcp.Description("Session ID to use."),
		),
		mcp.WithString("notebook_id",
			mcp.Description("Notebook ID to generate audio for."),
		),
		mcp.WithString("notebook_url",
			mcp.Description("Notebook URL to generate audio for."),
		),
		mcp.WithBoolean("show_browser",
			mcp.Description("Show browser window."),
		),
	)
}

func defGetAudioStatus() mcp.Tool {
	return mcp.NewTool("get_audio_status",
		mcp.WithDescription("Check the status of audio overview generation."),
		mcp.WithString("session_id",
			mcp.Description("Session ID to check."),
		),
		mcp.WithString("notebook_id",
			mcp.Description("Notebook ID to check."),
		),
		mcp.WithString("notebook_url",
			mcp.Description("Notebook URL to check."),
		),
		mcp.WithBoolean("show_browser",
			mcp.Description("Show browser window."),
		),
	)
}

func defDownloadAudio() mcp.Tool {
	return mcp.NewTool("download_audio",
		mcp.WithDescription("Download the audio overview file."),
		mcp.WithString("destination_dir",
			mcp.Required(),
			mcp.Description("Directory to save the audio file."),
		),
		mcp.WithString("session_id",
			mcp.Description("Session ID to use."),
		),
		mcp.WithString("notebook_id",
			mcp.Description("Notebook ID to download from."),
		),
		mcp.WithString("notebook_url",
			mcp.Description("Notebook URL to download from."),
		),
		mcp.WithBoolean("show_browser",
			mcp.Description("Show browser window."),
		),
	)
}
