// Package tools provides MCP tool definitions, registration, profile-based
// filtering, and handler dispatch for the NotebookLM MCP server.
package tools

// ToolProfile defines a named set of tools exposed to MCP clients.
type ToolProfile string

const (
	ProfileMinimal  ToolProfile = "minimal"
	ProfileStandard ToolProfile = "standard"
	ProfileFull     ToolProfile = "full"
)

// ToolProfiles maps each profile to the set of tool names it includes.
var ToolProfiles = map[ToolProfile][]string{
	ProfileMinimal: {
		"ask_question",
		"get_health",
		"list_notebooks",
		"get_notebook",
	},
	ProfileStandard: {
		"ask_question",
		"get_health",
		"list_notebooks",
		"get_notebook",
		"select_notebook",
		"update_notebook",
		"search_notebooks",
		"get_library_stats",
		"list_sessions",
		"close_session",
		"reset_session",
		"setup_auth",
		"add_source",
		"generate_audio",
		"get_audio_status",
		"download_audio",
	},
	ProfileFull: {
		"ask_question",
		"get_health",
		"list_notebooks",
		"get_notebook",
		"select_notebook",
		"update_notebook",
		"remove_notebook",
		"search_notebooks",
		"get_library_stats",
		"list_sessions",
		"close_session",
		"reset_session",
		"setup_auth",
		"re_auth",
		"cleanup_data",
		"add_source",
		"generate_audio",
		"get_audio_status",
		"download_audio",
		"add_notebook",
	},
}

// ResolveProfile returns the tool names for a given profile. If the profile
// is unknown, it returns the full profile.
func ResolveProfile(p ToolProfile) []string {
	if names, ok := ToolProfiles[p]; ok {
		return names
	}
	return ToolProfiles[ProfileFull]
}
