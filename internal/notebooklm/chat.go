package notebooklm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jorge/notebooklm-mcp-server/internal/apperrors"
	"github.com/jorge/notebooklm-mcp-server/internal/config"
	"github.com/jorge/notebooklm-mcp-server/internal/stealth"
	"github.com/playwright-community/playwright-go"
)

// AskResult holds the result of asking a question to NotebookLM.
type AskResult struct {
	Answer      string
	Citations   []Citation
	NotebookURL string
	ElapsedMs   int64
}

// Citation represents a single citation reference extracted from an answer.
type Citation struct {
	SourceTitle string
	SourceURL   string
	Quote       string
}

// pageReader defines the minimal page operations needed for reading answers.
type pageReader interface {
	Goto(url string, opts ...interface{}) (interface{}, error)
	QuerySelector(selector string) (interface{}, error)
	QuerySelectorAll(selector string) ([]interface{}, error)
	InnerText(selector string, opts ...interface{}) (string, error)
	WaitForSelector(selector string, opts ...interface{}) (interface{}, error)
	IsClosed() bool
	URL() string
}

// Ask asks a question on the given NotebookLM page and waits for a stable answer.
// It navigates to the notebook URL if needed, types the question with stealth,
// and polls for streaming-stability (3 identical reads).
func Ask(ctx context.Context, page playwright.Page, question string, cfg config.Config) (*AskResult, error) {
	start := time.Now()

	// Navigate to notebook URL if not already there
	currentURL := page.URL()
	if currentURL == "" || !strings.Contains(currentURL, "notebooklm.google") {
		if cfg.NotebookURL != "" {
			_, err := page.Goto(cfg.NotebookURL, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			})
			if err != nil {
				return nil, fmt.Errorf("navigate to notebook: %w", err)
			}
		}
	}

	// Snapshot existing answers before asking
	existingAnswers := SnapshotAnswers(page)

	// Find chat input
	inputEl, err := findChatInput(page)
	if err != nil {
		return nil, fmt.Errorf("find chat input: %w", err)
	}

	// Click the input to focus
	if err := inputEl.Click(); err != nil {
		return nil, fmt.Errorf("click chat input: %w", err)
	}

	// Human-type the question
	if cfg.StealthEnabled && cfg.StealthHumanTyping {
		if err := stealth.HumanType(page, question, cfg, nil); err != nil {
			return nil, fmt.Errorf("type question: %w", err)
		}
	} else {
		if err := page.Fill(ChatInput, question); err != nil {
			return nil, fmt.Errorf("fill question: %w", err)
		}
	}

	// Press Enter to submit
	if err := page.Keyboard().Press("Enter"); err != nil {
		return nil, fmt.Errorf("submit question: %w", err)
	}

	// Wait for stable answer
	timeout := time.Duration(cfg.AnswerTimeoutMs) * time.Millisecond
	answer, err := WaitForStableAnswer(ctx, page, timeout, 750*time.Millisecond, existingAnswers)
	if err != nil {
		// Check for rate limit
		if apperrors.IsRateLimit(err) {
			return nil, err
		}
		// Return partial answer on timeout
		if partial, partialErr := ReadLatestAnswer(page); partialErr == nil && partial != "" {
			return &AskResult{
				Answer:    SanitizeAnswer(partial),
				ElapsedMs: time.Since(start).Milliseconds(),
			}, fmt.Errorf("answer timeout (partial): %w", err)
		}
		return nil, err
	}

	return &AskResult{
		Answer:    SanitizeAnswer(answer.Text),
		ElapsedMs: time.Since(start).Milliseconds(),
	}, nil
}

// Answer represents a detected stable answer from the page.
type Answer struct {
	Text      string
	RawText   string
	Timestamp time.Time
}

// WaitForStableAnswer polls the page for a new answer that stabilizes
// (3 consecutive identical reads, filtered for placeholders and echoes).
func WaitForStableAnswer(ctx context.Context, page playwright.Page, timeout, pollInterval time.Duration, existingAnswers []string) (*Answer, error) {
	deadline := time.Now().Add(timeout)
	consecutiveMatches := 0
	var lastText string
	var bestEffort string

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			if bestEffort != "" {
				return &Answer{Text: bestEffort, Timestamp: time.Now()}, ctx.Err()
			}
			return nil, ctx.Err()
		default:
		}

		// Check if page is still alive
		if page.IsClosed() {
			return nil, fmt.Errorf("page closed during answer wait")
		}

		text, err := ReadLatestAnswer(page)
		if err != nil || text == "" {
			// No answer yet, reset counter
			consecutiveMatches = 0
			lastText = ""
			time.Sleep(pollInterval)
			continue
		}

		// Filter placeholders
		if IsPlaceholder(text) {
			consecutiveMatches = 0
			lastText = ""
			time.Sleep(pollInterval)
			continue
		}

		// Filter rate limit messages
		if IsRateLimitMessage(text) {
			return nil, apperrors.NewRateLimitError(text)
		}

		// Filter echo of the question
		trimmedQuestion := strings.TrimSpace(text)
		// Check if the answer is just the question echoed back
		if strings.EqualFold(strings.TrimSpace(text), strings.TrimSpace(trimmedQuestion)) {
			consecutiveMatches = 0
			lastText = ""
			time.Sleep(pollInterval)
			continue
		}

		// Check if this is a new answer (not in existing)
		isNew := true
		for _, existing := range existingAnswers {
			if strings.TrimSpace(existing) == strings.TrimSpace(text) {
				isNew = false
				break
			}
		}
		if !isNew {
			consecutiveMatches = 0
			lastText = ""
			time.Sleep(pollInterval)
			continue
		}

		// Check stability: 3 consecutive identical reads
		if text == lastText {
			consecutiveMatches++
			if consecutiveMatches >= 3 {
				return &Answer{Text: text, Timestamp: time.Now()}, nil
			}
		} else {
			consecutiveMatches = 1
			lastText = text
			bestEffort = text
		}

		time.Sleep(pollInterval)
	}

	// Timeout: return best effort if we have anything
	if bestEffort != "" {
		return &Answer{Text: bestEffort, Timestamp: time.Now()},
			fmt.Errorf("answer did not stabilize within %v", timeout)
	}
	return nil, fmt.Errorf("no answer detected within %v", timeout)
}

// SnapshotAnswers reads all existing answer texts from the page.
func SnapshotAnswers(page playwright.Page) []string {
	elements, err := page.QuerySelectorAll(AnswerContainer)
	if err != nil {
		return nil
	}

	var answers []string
	for _, el := range elements {
		element := el.(playwright.ElementHandle)
		text, err := element.InnerText()
		if err == nil && strings.TrimSpace(text) != "" {
			answers = append(answers, strings.TrimSpace(text))
		}
	}
	return answers
}

// ReadLatestAnswer reads only the most recent answer from the page.
func ReadLatestAnswer(page playwright.Page) (string, error) {
	text, err := page.InnerText(LatestAnswer)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// findChatInput tries the primary selector first, then aria-label fallbacks.
func findChatInput(page playwright.Page) (playwright.ElementHandle, error) {
	// Try primary selector
	el, err := page.QuerySelector(ChatInput)
	if err == nil && el != nil {
		return el.(playwright.ElementHandle), nil
	}

	// Try aria-label fallbacks
	for _, label := range ChatInputAriaLabels {
		selector := fmt.Sprintf("textarea[aria-label=%q]", label)
		el, err = page.QuerySelector(selector)
		if err == nil && el != nil {
			return el.(playwright.ElementHandle), nil
		}
	}

	return nil, fmt.Errorf("chat input not found (tried %s and %d aria-labels)", ChatInput, len(ChatInputAriaLabels))
}
