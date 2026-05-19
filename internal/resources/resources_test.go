package resources

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jorge/notebooklm-mcp-server/internal/library"
	"github.com/mark3labs/mcp-go/mcp"
)

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	libPath := t.TempDir() + "/library.json"
	lib, err := library.New(libPath)
	if err != nil {
		t.Fatalf("library.New: %v", err)
	}
	return NewRegistry(lib)
}

func TestRegistry_HandleLibrary(t *testing.T) {
	r := newTestRegistry(t)

	// Add some notebooks
	_, err := r.lib.Add(library.AddInput{
		URL:   "https://notebooklm.google.com/notebook/abc",
		Name:  "Notebook A",
		Description: "Description A",
		Topics: []string{"topic1"},
	})
	if err != nil {
		t.Fatalf("lib.Add: %v", err)
	}

	_, err = r.lib.Add(library.AddInput{
		URL:   "https://notebooklm.google.com/notebook/def",
		Name:  "Notebook B",
		Description: "Description B",
		Topics: []string{"topic2"},
	})
	if err != nil {
		t.Fatalf("lib.Add: %v", err)
	}

	req := mcp.ReadResourceRequest{}
	req.Params.URI = URILibrary

	results, err := r.handleLibrary(context.Background(), req)
	if err != nil {
		t.Fatalf("handleLibrary error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	textResult, ok := results[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatal("expected TextResourceContents")
	}

	if textResult.URI != URILibrary {
		t.Errorf("URI = %q, want %q", textResult.URI, URILibrary)
	}

	// Parse the JSON
	var notebooks []library.NotebookEntry
	if err := json.Unmarshal([]byte(textResult.Text), &notebooks); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if len(notebooks) != 2 {
		t.Errorf("got %d notebooks, want 2", len(notebooks))
	}
}

func TestRegistry_HandleLibrary_Empty(t *testing.T) {
	r := newTestRegistry(t)

	req := mcp.ReadResourceRequest{}
	req.Params.URI = URILibrary

	results, err := r.handleLibrary(context.Background(), req)
	if err != nil {
		t.Fatalf("handleLibrary error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	textResult := results[0].(mcp.TextResourceContents)

	var notebooks []library.NotebookEntry
	if err := json.Unmarshal([]byte(textResult.Text), &notebooks); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if len(notebooks) != 0 {
		t.Errorf("got %d notebooks, want 0", len(notebooks))
	}
}

func TestRegistry_HandleNotebookByID(t *testing.T) {
	r := newTestRegistry(t)

	_, err := r.lib.Add(library.AddInput{
		URL:   "https://notebooklm.google.com/notebook/abc",
		Name:  "Test Notebook",
		Description: "A test notebook",
		Topics: []string{"test"},
	})
	if err != nil {
		t.Fatalf("lib.Add: %v", err)
	}

	req := mcp.ReadResourceRequest{}
	req.Params.URI = "notebooklm://library/test-notebook"

	results, err := r.handleNotebookByID(context.Background(), req)
	if err != nil {
		t.Fatalf("handleNotebookByID error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	textResult, ok := results[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatal("expected TextResourceContents")
	}

	if textResult.URI != req.Params.URI {
		t.Errorf("URI = %q, want %q", textResult.URI, req.Params.URI)
	}

	// Parse the JSON
	var nb library.NotebookEntry
	if err := json.Unmarshal([]byte(textResult.Text), &nb); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if nb.Name != "Test Notebook" {
		t.Errorf("name = %q, want %q", nb.Name, "Test Notebook")
	}
}

func TestRegistry_HandleNotebookByID_NotFound(t *testing.T) {
	r := newTestRegistry(t)

	req := mcp.ReadResourceRequest{}
	req.Params.URI = "notebooklm://library/nonexistent"

	_, err := r.handleNotebookByID(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for nonexistent notebook")
	}
}

func TestRegistry_HandleNotebookByID_MissingID(t *testing.T) {
	r := newTestRegistry(t)

	req := mcp.ReadResourceRequest{}
	req.Params.URI = "notebooklm://library/"

	_, err := r.handleNotebookByID(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing notebook ID")
	}
}

func TestRegistry_Completions(t *testing.T) {
	r := newTestRegistry(t)

	// Add notebooks
	_, err := r.lib.Add(library.AddInput{
		URL:   "https://notebooklm.google.com/notebook/1",
		Name:  "Alpha",
		Description: "First",
		Topics: []string{"a"},
	})
	if err != nil {
		t.Fatalf("lib.Add: %v", err)
	}

	_, err = r.lib.Add(library.AddInput{
		URL:   "https://notebooklm.google.com/notebook/2",
		Name:  "Beta",
		Description: "Second",
		Topics: []string{"b"},
	})
	if err != nil {
		t.Fatalf("lib.Add: %v", err)
	}

	completions := r.Completions("")

	if len(completions) != 2 {
		t.Errorf("got %d completions, want 2", len(completions))
	}

	// Check that both IDs are present
	idSet := make(map[string]bool)
	for _, id := range completions {
		idSet[id] = true
	}

	if !idSet["alpha"] {
		t.Error("missing completion: alpha")
	}
	if !idSet["beta"] {
		t.Error("missing completion: beta")
	}
}

func TestRegistry_Completions_Empty(t *testing.T) {
	r := newTestRegistry(t)

	completions := r.Completions("")
	if len(completions) != 0 {
		t.Errorf("got %d completions, want 0", len(completions))
	}
}

func TestURILibraryConstants(t *testing.T) {
	if URILibrary != "notebooklm://library" {
		t.Errorf("URILibrary = %q, want %q", URILibrary, "notebooklm://library")
	}
	if URILibraryPrefix != "notebooklm://library/" {
		t.Errorf("URILibraryPrefix = %q, want %q", URILibraryPrefix, "notebooklm://library/")
	}
}
