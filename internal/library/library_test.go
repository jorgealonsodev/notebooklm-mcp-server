package library_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jorge/notebooklm-mcp-server/internal/library"
)

func tmpLib(t *testing.T) *library.NotebookLibrary {
	t.Helper()
	dir := t.TempDir()
	lib, err := library.New(filepath.Join(dir, "library.json"))
	if err != nil {
		t.Fatalf("library.New: %v", err)
	}
	return lib
}

// ---- Add ----

func TestAddNotebook_ReturnsEntry(t *testing.T) {
	lib := tmpLib(t)
	nb, err := lib.Add(library.AddInput{
		URL:         "https://notebooklm.google.com/notebook/abc",
		Name:        "Test NB",
		Description: "A test notebook",
		Topics:      []string{"go", "testing"},
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if nb.ID == "" {
		t.Error("ID should not be empty")
	}
	if nb.URL != "https://notebooklm.google.com/notebook/abc" {
		t.Errorf("URL mismatch: %q", nb.URL)
	}
	if nb.UseCount != 0 {
		t.Errorf("UseCount should be 0, got %d", nb.UseCount)
	}
}

func TestAddNotebook_IDIsSlugLike(t *testing.T) {
	lib := tmpLib(t)
	nb, _ := lib.Add(library.AddInput{
		URL: "https://notebooklm.google.com/notebook/xyz", Name: "My Great Notebook",
		Description: "x", Topics: []string{"x"},
	})
	for _, c := range nb.ID {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			t.Errorf("ID %q contains unexpected character %q", nb.ID, string(c))
		}
	}
}

func TestAddNotebook_DuplicateURLReturnsError(t *testing.T) {
	lib := tmpLib(t)
	inp := library.AddInput{
		URL: "https://notebooklm.google.com/notebook/dup", Name: "A",
		Description: "x", Topics: []string{"x"},
	}
	lib.Add(inp)
	_, err := lib.Add(inp)
	if err == nil {
		t.Error("adding duplicate URL should return error")
	}
}

func TestAddNotebook_DefaultContentTypes(t *testing.T) {
	lib := tmpLib(t)
	nb, _ := lib.Add(library.AddInput{
		URL: "https://notebooklm.google.com/notebook/ct", Name: "A",
		Description: "x", Topics: []string{"x"},
	})
	if len(nb.ContentTypes) == 0 {
		t.Error("ContentTypes should not be empty when not provided")
	}
}

// ---- List ----

func TestListNotebooks_EmptyByDefault(t *testing.T) {
	lib := tmpLib(t)
	nbs := lib.List()
	if len(nbs) != 0 {
		t.Errorf("expected 0 notebooks, got %d", len(nbs))
	}
}

func TestListNotebooks_ReturnsAllAdded(t *testing.T) {
	lib := tmpLib(t)
	lib.Add(library.AddInput{URL: "https://notebooklm.google.com/notebook/a", Name: "A", Description: "x", Topics: []string{"x"}})
	lib.Add(library.AddInput{URL: "https://notebooklm.google.com/notebook/b", Name: "B", Description: "x", Topics: []string{"x"}})
	if len(lib.List()) != 2 {
		t.Errorf("expected 2 notebooks")
	}
}

// ---- Get ----

func TestGetNotebook_ExistingID(t *testing.T) {
	lib := tmpLib(t)
	nb, _ := lib.Add(library.AddInput{
		URL: "https://notebooklm.google.com/notebook/g", Name: "G",
		Description: "x", Topics: []string{"x"},
	})
	got, err := lib.Get(nb.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != nb.ID {
		t.Errorf("ID mismatch")
	}
}

func TestGetNotebook_MissingID(t *testing.T) {
	lib := tmpLib(t)
	_, err := lib.Get("does-not-exist")
	if err == nil {
		t.Error("Get with missing ID should return error")
	}
}

// ---- Select / Active ----

func TestSelectNotebook_SetsActive(t *testing.T) {
	lib := tmpLib(t)
	nb, _ := lib.Add(library.AddInput{
		URL: "https://notebooklm.google.com/notebook/s", Name: "S",
		Description: "x", Topics: []string{"x"},
	})
	if err := lib.Select(nb.ID); err != nil {
		t.Fatalf("Select: %v", err)
	}
	active := lib.Active()
	if active == nil || active.ID != nb.ID {
		t.Error("Active() should return the selected notebook")
	}
}

func TestSelectNotebook_MissingID(t *testing.T) {
	lib := tmpLib(t)
	if err := lib.Select("nope"); err == nil {
		t.Error("Select with missing ID should return error")
	}
}

func TestActive_NilWhenNoneSelected(t *testing.T) {
	lib := tmpLib(t)
	if lib.Active() != nil {
		t.Error("Active should be nil when no notebook is selected")
	}
}

// ---- Update ----

func TestUpdateNotebook_ChangesName(t *testing.T) {
	lib := tmpLib(t)
	nb, _ := lib.Add(library.AddInput{
		URL: "https://notebooklm.google.com/notebook/u", Name: "Old",
		Description: "x", Topics: []string{"x"},
	})
	updated, err := lib.Update(library.UpdateInput{ID: nb.ID, Name: strPtr("New")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "New" {
		t.Errorf("name should be 'New', got %q", updated.Name)
	}
}

func TestUpdateNotebook_MissingID(t *testing.T) {
	lib := tmpLib(t)
	_, err := lib.Update(library.UpdateInput{ID: "ghost"})
	if err == nil {
		t.Error("Update with missing ID should return error")
	}
}

// ---- Remove ----

func TestRemoveNotebook_RemovesFromList(t *testing.T) {
	lib := tmpLib(t)
	nb, _ := lib.Add(library.AddInput{
		URL: "https://notebooklm.google.com/notebook/r", Name: "R",
		Description: "x", Topics: []string{"x"},
	})
	if err := lib.Remove(nb.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(lib.List()) != 0 {
		t.Error("library should be empty after remove")
	}
}

func TestRemoveNotebook_ClearsActiveWhenItMatches(t *testing.T) {
	lib := tmpLib(t)
	nb, _ := lib.Add(library.AddInput{
		URL: "https://notebooklm.google.com/notebook/ra", Name: "RA",
		Description: "x", Topics: []string{"x"},
	})
	lib.Select(nb.ID)
	lib.Remove(nb.ID)
	if lib.Active() != nil {
		t.Error("Active should be nil after active notebook is removed")
	}
}

func TestRemoveNotebook_MissingID(t *testing.T) {
	lib := tmpLib(t)
	if err := lib.Remove("ghost"); err == nil {
		t.Error("Remove with missing ID should return error")
	}
}

// ---- IncrementUseCount ----

func TestIncrementUseCount(t *testing.T) {
	lib := tmpLib(t)
	nb, _ := lib.Add(library.AddInput{
		URL: "https://notebooklm.google.com/notebook/inc", Name: "I",
		Description: "x", Topics: []string{"x"},
	})
	updated, err := lib.IncrementUseCount(nb.ID)
	if err != nil {
		t.Fatalf("IncrementUseCount: %v", err)
	}
	if updated.UseCount != 1 {
		t.Errorf("UseCount should be 1, got %d", updated.UseCount)
	}
	updated2, _ := lib.IncrementUseCount(nb.ID)
	if updated2.UseCount != 2 {
		t.Errorf("UseCount should be 2, got %d", updated2.UseCount)
	}
}

// ---- Search ----

func TestSearchNotebooks_MatchesName(t *testing.T) {
	lib := tmpLib(t)
	lib.Add(library.AddInput{
		URL: "https://notebooklm.google.com/notebook/go", Name: "Go Testing Guide",
		Description: "Go language testing", Topics: []string{"go", "testing"},
	})
	lib.Add(library.AddInput{
		URL: "https://notebooklm.google.com/notebook/py", Name: "Python Guide",
		Description: "Python language", Topics: []string{"python"},
	})
	results := lib.Search("Go")
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestSearchNotebooks_CaseInsensitive(t *testing.T) {
	lib := tmpLib(t)
	lib.Add(library.AddInput{
		URL: "https://notebooklm.google.com/notebook/ci", Name: "Docker Ops",
		Description: "Container operations", Topics: []string{"docker"},
	})
	if len(lib.Search("docker")) == 0 {
		t.Error("search should be case-insensitive")
	}
}

func TestSearchNotebooks_MatchesTopic(t *testing.T) {
	lib := tmpLib(t)
	lib.Add(library.AddInput{
		URL: "https://notebooklm.google.com/notebook/k8s", Name: "Kubernetes",
		Description: "Container orchestration", Topics: []string{"kubernetes", "docker"},
	})
	if len(lib.Search("kubernetes")) == 0 {
		t.Error("should match by topic")
	}
}

// ---- Persistence ----

func TestPersistence_ReloadFromDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "library.json")

	lib1, _ := library.New(path)
	nb, _ := lib1.Add(library.AddInput{
		URL: "https://notebooklm.google.com/notebook/p", Name: "Persist",
		Description: "x", Topics: []string{"x"},
	})
	lib1.Select(nb.ID)

	lib2, err := library.New(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(lib2.List()) != 1 {
		t.Errorf("expected 1 notebook after reload, got %d", len(lib2.List()))
	}
	if lib2.Active() == nil || lib2.Active().ID != nb.ID {
		t.Error("active notebook should survive reload")
	}
}

func TestPersistence_CreatesFileIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "library.json")
	_, err := library.New(path)
	if err != nil {
		t.Fatalf("New should create missing dirs: %v", err)
	}
	// File should exist now (created on first write)
	lib2, _ := library.New(path)
	if lib2 == nil {
		t.Error("should be able to open newly created library")
	}
}

// ---- Stats ----

func TestStats_TotalNotebooks(t *testing.T) {
	lib := tmpLib(t)
	lib.Add(library.AddInput{URL: "https://notebooklm.google.com/notebook/s1", Name: "S1", Description: "x", Topics: []string{"x"}})
	lib.Add(library.AddInput{URL: "https://notebooklm.google.com/notebook/s2", Name: "S2", Description: "x", Topics: []string{"x"}})
	stats := lib.Stats()
	if stats.TotalNotebooks != 2 {
		t.Errorf("expected 2, got %d", stats.TotalNotebooks)
	}
}

func TestStats_TotalQueries(t *testing.T) {
	lib := tmpLib(t)
	nb, _ := lib.Add(library.AddInput{URL: "https://notebooklm.google.com/notebook/q", Name: "Q", Description: "x", Topics: []string{"x"}})
	lib.IncrementUseCount(nb.ID)
	lib.IncrementUseCount(nb.ID)
	stats := lib.Stats()
	if stats.TotalQueries != 2 {
		t.Errorf("expected 2 total queries, got %d", stats.TotalQueries)
	}
}

func TestStats_LastModified_UpdatesOnChange(t *testing.T) {
	lib := tmpLib(t)
	before := time.Now().UTC().Truncate(time.Second)
	lib.Add(library.AddInput{URL: "https://notebooklm.google.com/notebook/lm", Name: "LM", Description: "x", Topics: []string{"x"}})
	stats := lib.Stats()
	modified, err := time.Parse(time.RFC3339, stats.LastModified)
	if err != nil {
		t.Fatalf("LastModified parse: %v", err)
	}
	if modified.Before(before) {
		t.Error("LastModified should be after test start")
	}
}

// ---- Concurrency ----

func TestConcurrency_ParallelAdds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "library.json")
	lib, _ := library.New(path)

	done := make(chan error, 10)
	for i := range 10 {
		go func(i int) {
			_, err := lib.Add(library.AddInput{
				URL:         "https://notebooklm.google.com/notebook/" + string(rune('a'+i)),
				Name:        "NB " + string(rune('A'+i)),
				Description: "x", Topics: []string{"x"},
			})
			done <- err
		}(i)
	}
	for range 10 {
		if err := <-done; err != nil {
			t.Errorf("concurrent Add: %v", err)
		}
	}
	if len(lib.List()) != 10 {
		t.Errorf("expected 10 notebooks after concurrent adds, got %d", len(lib.List()))
	}
}

// helpers

func strPtr(s string) *string { return &s }

