package notebooklm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

// SourceResult holds the result of adding a source to a notebook.
type SourceResult struct {
	SourceCount int
	SourceType  string
	ElapsedMs   int64
}

// AddSource adds a source to the current notebook. It opens the add-source dialog,
// selects the source type, fills content, clicks Insert, and waits for the source
// count to increase (up to 90 seconds).
func AddSource(page playwright.Page, sourceType string, content string, title string) (*SourceResult, error) {
	start := time.Now()

	// Snapshot source count before adding
	beforeCount, err := CountSources(page)
	if err != nil {
		return nil, fmt.Errorf("count sources before: %w", err)
	}

	// Open add-source dialog
	addBtn, err := page.QuerySelector(AddSourceButton)
	if err != nil || addBtn == nil {
		return nil, fmt.Errorf("add source button not found")
	}
	if err := addBtn.Click(); err != nil {
		return nil, fmt.Errorf("click add source: %w", err)
	}

	// Wait for dialog to appear
	if _, err := page.WaitForSelector(Dialog, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(10000),
	}); err != nil {
		return nil, fmt.Errorf("dialog did not appear: %w", err)
	}

	// Pick source type (URL or text)
	var typeButton string
	switch strings.ToLower(sourceType) {
	case "url":
		typeButton = "text:has(\"URL\")"
	case "text", "note":
		typeButton = "text:has(\"Text\")"
	default:
		typeButton = "text:has(\"URL\")" // default to URL
	}

	typeEl, err := page.QuerySelector(typeButton)
	if err != nil || typeEl == nil {
		// Fallback: try generic buttons
		typeEl, err = page.QuerySelector("button:has-text(\"URL\")")
		if err != nil || typeEl == nil {
			return nil, fmt.Errorf("source type button not found for %q", sourceType)
		}
	}
	if err := typeEl.Click(); err != nil {
		return nil, fmt.Errorf("click source type: %w", err)
	}

	// Fill content
	if err := page.Fill("textarea, input[type=\"url\"]", content); err != nil {
		return nil, fmt.Errorf("fill content: %w", err)
	}

	// Fill title if text type
	if strings.ToLower(sourceType) == "text" && title != "" {
		titleEl, err := page.QuerySelector("input[placeholder*=\"title\" i], input[placeholder*=\"Title\" i]")
		if err == nil && titleEl != nil {
			if err := titleEl.(playwright.ElementHandle).Fill(title); err != nil {
				// Non-fatal: title is optional
			}
		}
	}

	// Click Insert button
	insertBtn, err := page.QuerySelector("button:has-text(\"Insert\"), button:has-text(\"Add\")")
	if err != nil || insertBtn == nil {
		return nil, fmt.Errorf("insert button not found")
	}
	if err := insertBtn.(playwright.ElementHandle).Click(); err != nil {
		return nil, fmt.Errorf("click insert: %w", err)
	}

	// Wait for dialog to close
	if _, err := page.WaitForSelector(Dialog, playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateHidden,
		Timeout: playwright.Float(30000),
	}); err != nil {
		return nil, fmt.Errorf("dialog did not close: %w", err)
	}

	// Poll source count increase up to 90s
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		count, err := CountSources(page)
		if err == nil && count > beforeCount {
			// Verify via header regex as well
			headerCount, headerErr := countFromHeader(page)
			if headerErr == nil && headerCount >= count {
				return &SourceResult{
					SourceCount: count,
					SourceType:  sourceType,
					ElapsedMs:   time.Since(start).Milliseconds(),
				}, nil
			}
			// Header check failed but container count increased — still success
			return &SourceResult{
				SourceCount: count,
				SourceType:  sourceType,
				ElapsedMs:   time.Since(start).Milliseconds(),
			}, nil
		}
		time.Sleep(1 * time.Second)
	}

	return nil, fmt.Errorf("source count did not increase after 90s (was %d)", beforeCount)
}

// CountSources counts the number of source containers on the page.
func CountSources(page playwright.Page) (int, error) {
	elements, err := page.QuerySelectorAll(SourceRow)
	if err != nil {
		return 0, fmt.Errorf("query sources: %w", err)
	}
	return len(elements), nil
}

// countFromHeader reads the source count from the header element using regex.
func countFromHeader(page playwright.Page) (int, error) {
	text, err := page.InnerText(SourceCountHeader)
	if err != nil {
		return 0, err
	}
	// Extract number from text like "5 sources" or "5 Source(s)"
	re := regexp.MustCompile(`(\d+)`)
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return 0, fmt.Errorf("no number found in header: %q", text)
	}
	count, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("parse header count: %w", err)
	}
	return count, nil
}

// IsUUIDRedirect checks if the current URL looks like a UUID redirect mismatch.
// NotebookLM sometimes redirects to a different UUID after adding a source.
func IsUUIDRedirect(currentURL, expectedURL string) bool {
	// Extract UUIDs from both URLs and compare
	currentUUID := extractUUID(currentURL)
	expectedUUID := extractUUID(expectedURL)
	if currentUUID == "" || expectedUUID == "" {
		return false
	}
	return currentUUID != expectedUUID
}

// extractUUID extracts a UUID from a URL path.
func extractUUID(url string) string {
	// UUID pattern: 8-4-4-4-12 hex chars
	re := regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	match := re.FindString(strings.ToLower(url))
	return match
}
