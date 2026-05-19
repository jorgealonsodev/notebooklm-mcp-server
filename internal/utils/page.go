// Package utils provides shared utilities for the notebooklm-mcp-server:
// logging, cleanup, AI provenance, page snapshots, and URL polling.
package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

// PageSnapshot captures the current state of a NotebookLM page, including
// all visible answer texts and the page URL. Useful for pre-ask baselines
// and debugging page state.
type PageSnapshot struct {
	URL     string   `json:"url"`
	Answers []string `json:"answers"`
}

// SnapshotPage reads all existing answer texts and the current URL from
// a NotebookLM page. Returns empty answers slice if the selector fails.
func SnapshotPage(page playwright.Page) *PageSnapshot {
	snap := &PageSnapshot{
		URL: page.URL(),
	}

	elements, err := page.QuerySelectorAll(".to-user-container .message-text-content")
	if err != nil {
		return snap
	}

	for _, el := range elements {
		element := el.(playwright.ElementHandle)
		text, err := element.InnerText()
		if err == nil && strings.TrimSpace(text) != "" {
			snap.Answers = append(snap.Answers, strings.TrimSpace(text))
		}
	}

	return snap
}

// URLPoll polls the page URL at the given interval until the check function
// returns true or the context is cancelled/times out. Returns nil on success,
// or an error describing why polling stopped.
func URLPoll(ctx context.Context, page playwright.Page, check func(url string) bool, interval, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return fmt.Errorf("URLPoll: context cancelled: %w", ctx.Err())
		default:
		}

		currentURL := page.URL()
		if check(currentURL) {
			return nil
		}

		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return fmt.Errorf("URLPoll: context cancelled: %w", ctx.Err())
		}
	}

	return fmt.Errorf("URLPoll: condition not met within %v", timeout)
}
