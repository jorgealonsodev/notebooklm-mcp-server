// Package resources provides MCP resource handlers for the NotebookLM library.
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jorge/notebooklm-mcp-server/internal/library"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	// URILibrary is the base URI for the full library.
	URILibrary = "notebooklm://library"
	// URILibraryPrefix is the URI prefix for individual notebooks.
	URILibraryPrefix = "notebooklm://library/"
)

// Registry manages MCP resource registration and handling.
type Registry struct {
	lib *library.NotebookLibrary
}

// NewRegistry creates a new resource registry.
func NewRegistry(lib *library.NotebookLibrary) *Registry {
	return &Registry{lib: lib}
}

// RegisterWithServer registers all resources with the MCP server.
func (r *Registry) RegisterWithServer(s *server.MCPServer) {
	// notebooklm://library — all notebooks
	s.AddResource(mcp.NewResource(URILibrary, "NotebookLM Library",
		mcp.WithResourceDescription("Returns all notebooks in the library as JSON."),
		mcp.WithMIMEType("application/json"),
	), r.handleLibrary)

	// notebooklm://library/{id} — single notebook (template)
	s.AddResourceTemplate(mcp.NewResourceTemplate(
		URILibraryPrefix+"{id}", "NotebookLM Notebook",
		mcp.WithTemplateDescription("Returns a single notebook by ID as JSON."),
		mcp.WithTemplateMIMEType("application/json"),
	), r.handleNotebookByID)
}

// handleLibrary returns all notebooks as JSON.
func (r *Registry) handleLibrary(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	notebooks := r.lib.List()

	data, err := json.MarshalIndent(notebooks, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal library: %w", err)
	}

	uri := URILibrary
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

// handleNotebookByID returns a single notebook by ID.
func (r *Registry) handleNotebookByID(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract ID from URI: "notebooklm://library/{id}"
	id := strings.TrimPrefix(req.Params.URI, URILibraryPrefix)
	if id == "" {
		return nil, fmt.Errorf("missing notebook ID in URI")
	}

	nb, err := r.lib.Get(id)
	if err != nil {
		return nil, fmt.Errorf("notebook %q: %w", id, err)
	}

	data, err := json.MarshalIndent(nb, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal notebook: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

// Completions returns notebook ID completions for the given URI prefix.
func (r *Registry) Completions(uri string) []string {
	notebooks := r.lib.List()
	var ids []string
	for _, nb := range notebooks {
		ids = append(ids, nb.ID)
	}
	return ids
}
