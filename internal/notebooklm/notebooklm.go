package notebooklm

import (
	"context"

	"github.com/jorge/notebooklm-mcp-server/internal/config"
	"github.com/playwright-community/playwright-go"
)

// pageOps defines the minimal page operations needed by the NotebookLM controller.
// playwright.Page satisfies this interface implicitly.
type pageOps interface {
	Goto(url string, opts ...interface{}) (interface{}, error)
	QuerySelector(selector string) (interface{}, error)
	QuerySelectorAll(selector string) ([]interface{}, error)
	InnerText(selector string, opts ...interface{}) (string, error)
	WaitForSelector(selector string, opts ...interface{}) (interface{}, error)
	Fill(selector, value string, opts ...interface{}) error
	IsClosed() bool
	URL() string
	Keyboard() playwright.Keyboard
	WaitForEvent(event string, opts ...interface{}) (interface{}, error)
}

// Controller provides the NotebookLM domain operations: chat, sources,
// audio overviews, and citation extraction.
type Controller struct {
	cfg config.Config
}

// NewController creates a new NotebookLM controller.
func NewController(cfg config.Config) *Controller {
	return &Controller{cfg: cfg}
}

// Ask asks a question on the given NotebookLM page and waits for a stable answer.
func (c *Controller) Ask(ctx context.Context, page playwright.Page, question string) (*AskResult, error) {
	return Ask(ctx, page, question, c.cfg)
}

// AddSource adds a source to the current notebook.
func (c *Controller) AddSource(page playwright.Page, sourceType string, content string, title string) (*SourceResult, error) {
	return AddSource(page, sourceType, content, title)
}

// CountSources counts the number of sources on the page.
func (c *Controller) CountSources(page playwright.Page) (int, error) {
	return CountSources(page)
}

// GenerateAudio generates an audio overview for the current notebook.
func (c *Controller) GenerateAudio(page playwright.Page, customPrompt string, timeoutMs int) (*AudioResult, error) {
	return GenerateAudio(page, customPrompt, timeoutMs)
}

// GetAudioStatus checks the current state of the audio overview.
func (c *Controller) GetAudioStatus(page playwright.Page) (AudioStatus, error) {
	return GetAudioStatus(page)
}

// DownloadAudio downloads the audio overview file to the specified directory.
func (c *Controller) DownloadAudio(page playwright.Page, destDir string) (*DownloadResult, error) {
	return DownloadAudio(page, destDir)
}

// ExtractCitations extracts citations from the latest answer.
func (c *Controller) ExtractCitations(page playwright.Page, format CitationFormat) ([]FormattedCitation, error) {
	return ExtractCitations(page, format)
}
