// Package library manages a persistent collection of NotebookLM notebooks.
// The library is stored as a JSON file on disk and protected by an RWMutex for
// concurrent access.
package library

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// NotebookEntry is a single notebook stored in the library.
type NotebookEntry struct {
	ID           string   `json:"id"`
	URL          string   `json:"url"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Topics       []string `json:"topics"`
	ContentTypes []string `json:"content_types"`
	UseCases     []string `json:"use_cases"`
	Tags         []string `json:"tags,omitempty"`
	AddedAt      string   `json:"added_at"`
	LastUsed     string   `json:"last_used"`
	UseCount     int      `json:"use_count"`
}

// libraryFile is the on-disk JSON structure.
type libraryFile struct {
	Notebooks        []*NotebookEntry `json:"notebooks"`
	ActiveNotebookID string           `json:"active_notebook_id"`
	LastModified     string           `json:"last_modified"`
	Version          string           `json:"version"`
}

// Stats holds aggregate information about the library.
type Stats struct {
	TotalNotebooks   int
	ActiveNotebook   *string // nil if none selected
	MostUsedNotebook *string // nil if empty
	TotalQueries     int
	LastModified     string
}

// AddInput is the input for adding a new notebook.
type AddInput struct {
	URL          string
	Name         string
	Description  string
	Topics       []string
	ContentTypes []string
	UseCases     []string
	Tags         []string
}

// UpdateInput carries optional field updates for an existing notebook.
type UpdateInput struct {
	ID           string
	Name         *string
	Description  *string
	Topics       []string
	ContentTypes []string
	UseCases     []string
	Tags         []string
	URL          *string
}

// NotebookLibrary is a concurrency-safe, persistent notebook store.
type NotebookLibrary struct {
	mu   sync.RWMutex
	path string
	data libraryFile
}

// New opens (or creates) the library at the given path.
func New(path string) (*NotebookLibrary, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("library: mkdir: %w", err)
	}
	lib := &NotebookLibrary{
		path: path,
		data: libraryFile{
			Version:      "1",
			LastModified: time.Now().UTC().Format(time.RFC3339),
		},
	}
	if err := lib.load(); err != nil {
		return nil, err
	}
	return lib, nil
}

// Add inserts a new notebook. Returns an error if the URL is already present.
func (l *NotebookLibrary) Add(in AddInput) (*NotebookEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, nb := range l.data.Notebooks {
		if nb.URL == in.URL {
			return nil, fmt.Errorf("library: notebook with URL %q already exists (id=%s)", in.URL, nb.ID)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	ct := in.ContentTypes
	if len(ct) == 0 {
		ct = []string{"documentation", "examples"}
	}
	uc := in.UseCases
	if len(uc) == 0 {
		uc = []string{"General research"}
	}

	entry := &NotebookEntry{
		ID:           slugify(in.Name),
		URL:          in.URL,
		Name:         in.Name,
		Description:  in.Description,
		Topics:       in.Topics,
		ContentTypes: ct,
		UseCases:     uc,
		Tags:         in.Tags,
		AddedAt:      now,
		LastUsed:     now,
		UseCount:     0,
	}
	// Ensure uniqueness of generated ID
	entry.ID = l.uniqueID(entry.ID)

	l.data.Notebooks = append(l.data.Notebooks, entry)
	l.touch()
	return entry, l.save()
}

// List returns all notebooks (copies).
func (l *NotebookLibrary) List() []NotebookEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]NotebookEntry, len(l.data.Notebooks))
	for i, nb := range l.data.Notebooks {
		out[i] = *nb
	}
	return out
}

// Get returns a single notebook by ID.
func (l *NotebookLibrary) Get(id string) (*NotebookEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	nb := l.find(id)
	if nb == nil {
		return nil, fmt.Errorf("library: notebook not found: %s", id)
	}
	cp := *nb
	return &cp, nil
}

// Select marks a notebook as active.
func (l *NotebookLibrary) Select(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.find(id) == nil {
		return fmt.Errorf("library: notebook not found: %s", id)
	}
	l.data.ActiveNotebookID = id
	l.touch()
	return l.save()
}

// Active returns the currently-selected notebook, or nil.
func (l *NotebookLibrary) Active() *NotebookEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.data.ActiveNotebookID == "" {
		return nil
	}
	nb := l.find(l.data.ActiveNotebookID)
	if nb == nil {
		return nil
	}
	cp := *nb
	return &cp
}

// Update applies partial updates to an existing notebook.
func (l *NotebookLibrary) Update(in UpdateInput) (*NotebookEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	nb := l.find(in.ID)
	if nb == nil {
		return nil, fmt.Errorf("library: notebook not found: %s", in.ID)
	}
	if in.Name != nil {
		nb.Name = *in.Name
	}
	if in.Description != nil {
		nb.Description = *in.Description
	}
	if in.Topics != nil {
		nb.Topics = in.Topics
	}
	if in.ContentTypes != nil {
		nb.ContentTypes = in.ContentTypes
	}
	if in.UseCases != nil {
		nb.UseCases = in.UseCases
	}
	if in.Tags != nil {
		nb.Tags = in.Tags
	}
	if in.URL != nil {
		nb.URL = *in.URL
	}
	l.touch()
	cp := *nb
	return &cp, l.save()
}

// Remove deletes a notebook by ID. If it was the active notebook, active is cleared.
func (l *NotebookLibrary) Remove(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	idx := -1
	for i, nb := range l.data.Notebooks {
		if nb.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("library: notebook not found: %s", id)
	}
	l.data.Notebooks = append(l.data.Notebooks[:idx], l.data.Notebooks[idx+1:]...)
	if l.data.ActiveNotebookID == id {
		l.data.ActiveNotebookID = ""
	}
	l.touch()
	return l.save()
}

// IncrementUseCount increments the use counter and updates last_used timestamp.
func (l *NotebookLibrary) IncrementUseCount(id string) (*NotebookEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	nb := l.find(id)
	if nb == nil {
		return nil, fmt.Errorf("library: notebook not found: %s", id)
	}
	nb.UseCount++
	nb.LastUsed = time.Now().UTC().Format(time.RFC3339)
	l.touch()
	cp := *nb
	return &cp, l.save()
}

// Search returns notebooks whose name, description, or topics contain query
// (case-insensitive).
func (l *NotebookLibrary) Search(query string) []NotebookEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	q := strings.ToLower(query)
	var out []NotebookEntry
	for _, nb := range l.data.Notebooks {
		if strings.Contains(strings.ToLower(nb.Name), q) ||
			strings.Contains(strings.ToLower(nb.Description), q) {
			out = append(out, *nb)
			continue
		}
		for _, topic := range nb.Topics {
			if strings.Contains(strings.ToLower(topic), q) {
				out = append(out, *nb)
				break
			}
		}
	}
	return out
}

// Stats returns aggregate library statistics.
func (l *NotebookLibrary) Stats() Stats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	s := Stats{
		TotalNotebooks: len(l.data.Notebooks),
		LastModified:   l.data.LastModified,
	}
	if l.data.ActiveNotebookID != "" {
		id := l.data.ActiveNotebookID
		s.ActiveNotebook = &id
	}
	var best *NotebookEntry
	for _, nb := range l.data.Notebooks {
		s.TotalQueries += nb.UseCount
		if best == nil || nb.UseCount > best.UseCount {
			best = nb
		}
	}
	if best != nil && best.UseCount > 0 {
		s.MostUsedNotebook = &best.ID
	}
	return s
}

// ---- internal ----

// find returns a pointer into the slice (for mutation). Caller must hold lock.
func (l *NotebookLibrary) find(id string) *NotebookEntry {
	for _, nb := range l.data.Notebooks {
		if nb.ID == id {
			return nb
		}
	}
	return nil
}

func (l *NotebookLibrary) touch() {
	l.data.LastModified = time.Now().UTC().Format(time.RFC3339)
}

func (l *NotebookLibrary) load() error {
	data, err := os.ReadFile(l.path)
	if os.IsNotExist(err) {
		return nil // fresh library — start empty, will be created on first write
	}
	if err != nil {
		return fmt.Errorf("library: read %s: %w", l.path, err)
	}
	if err := json.Unmarshal(data, &l.data); err != nil {
		return fmt.Errorf("library: parse %s: %w", l.path, err)
	}
	return nil
}

func (l *NotebookLibrary) save() error {
	data, err := json.MarshalIndent(l.data, "", "  ")
	if err != nil {
		return fmt.Errorf("library: marshal: %w", err)
	}
	if err := os.WriteFile(l.path, data, 0o644); err != nil {
		return fmt.Errorf("library: write %s: %w", l.path, err)
	}
	return nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a name to a lowercase dash-separated slug.
func slugify(name string) string {
	s := strings.ToLower(name)
	s = nonSlug.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "notebook"
	}
	return s
}

// uniqueID ensures the generated slug does not collide with existing IDs.
// Caller must hold the write lock.
func (l *NotebookLibrary) uniqueID(base string) string {
	candidate := base
	for i := 2; ; i++ {
		collision := false
		for _, nb := range l.data.Notebooks {
			if nb.ID == candidate {
				collision = true
				break
			}
		}
		if !collision {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}
