package tools

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestAllToolDefinitions_Count(t *testing.T) {
	defs := AllToolDefinitions()
	if len(defs) != 20 {
		t.Errorf("AllToolDefinitions() returned %d tools, want 20", len(defs))
	}
}

func TestAllToolDefinitions_Names(t *testing.T) {
	wantNames := []string{
		"ask_question",
		"add_notebook",
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
		"get_health",
		"setup_auth",
		"re_auth",
		"cleanup_data",
		"add_source",
		"generate_audio",
		"get_audio_status",
		"download_audio",
	}

	defs := AllToolDefinitions()

	for _, want := range wantNames {
		def, ok := defs[want]
		if !ok {
			t.Errorf("missing tool definition: %s", want)
			continue
		}
		if def.Name != want {
			t.Errorf("tool definition name = %q, want %q", def.Name, want)
		}
		if def.Description == "" {
			t.Errorf("tool %q has empty description", want)
		}
	}
}

func TestAllToolDefinitions_RequiredFields(t *testing.T) {
	defs := AllToolDefinitions()

	tests := []struct {
		name     string
		required []string
	}{
		{"ask_question", []string{"question"}},
		{"add_notebook", []string{"url", "name", "description", "topics"}},
		{"get_notebook", []string{"id"}},
		{"select_notebook", []string{"id"}},
		{"update_notebook", []string{"id"}},
		{"remove_notebook", []string{"id"}},
		{"search_notebooks", []string{"query"}},
		{"close_session", []string{"session_id"}},
		{"reset_session", []string{"session_id"}},
		{"cleanup_data", []string{"confirm"}},
		{"add_source", []string{"type", "content"}},
		{"download_audio", []string{"destination_dir"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, ok := defs[tt.name]
			if !ok {
				t.Fatalf("missing tool: %s", tt.name)
			}
			for _, field := range tt.required {
				if !isRequired(def, field) {
					t.Errorf("field %q should be required", field)
				}
			}
		})
	}
}

func TestAllToolDefinitions_NoRequiredFields(t *testing.T) {
	defs := AllToolDefinitions()

	noRequiredTools := []string{
		"list_notebooks",
		"get_library_stats",
		"list_sessions",
		"get_health",
	}

	for _, name := range noRequiredTools {
		t.Run(name, func(t *testing.T) {
			def, ok := defs[name]
			if !ok {
				t.Fatalf("missing tool: %s", name)
			}
			if len(def.InputSchema.Required) > 0 {
				t.Errorf("has required fields %v, expected none", def.InputSchema.Required)
			}
		})
	}
}

func TestAllToolDefinitions_EnumFields(t *testing.T) {
	defs := AllToolDefinitions()

	// ask_question: source_format should have enum values
	t.Run("ask_question source_format enum", func(t *testing.T) {
		def := defs["ask_question"]
		enum := getEnumValues(def, "source_format")
		if len(enum) == 0 {
			t.Error("source_format should have enum values")
		}
	})

	// add_source: type should have enum values
	t.Run("add_source type enum", func(t *testing.T) {
		def := defs["add_source"]
		enum := getEnumValues(def, "type")
		if len(enum) == 0 {
			t.Error("type should have enum values")
		}
	})
}

// isRequired checks if a field is in the tool's required list.
func isRequired(tool mcp.Tool, field string) bool {
	for _, r := range tool.InputSchema.Required {
		if r == field {
			return true
		}
	}
	return false
}

// getEnumValues returns enum values for a property if it has them.
func getEnumValues(tool mcp.Tool, field string) []string {
	if tool.InputSchema.Properties == nil {
		return nil
	}
	prop, ok := tool.InputSchema.Properties[field]
	if !ok {
		return nil
	}
	// The property is a map[string]any
	if m, ok := prop.(map[string]any); ok {
		if enum, ok := m["enum"].([]string); ok {
			return enum
		}
	}
	return nil
}
