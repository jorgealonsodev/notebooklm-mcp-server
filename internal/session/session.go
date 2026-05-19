package session

import (
	"time"
)

// BrowserSession represents a single browser tab session.
type BrowserSession struct {
	ID           string
	NotebookURL  string
	Page         any // actual type is playwright.Page at runtime
	CreatedAt    time.Time
	LastActivity time.Time
	MessageCount int
}

// NewBrowserSession creates a new session with a generated 8-char hex ID.
func NewBrowserSession(notebookURL string) *BrowserSession {
	return &BrowserSession{
		ID:           generateID(),
		NotebookURL:  notebookURL,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		MessageCount: 0,
	}
}

// Touch updates the LastActivity timestamp to now.
func (s *BrowserSession) Touch() {
	s.LastActivity = time.Now()
}

// IncrementMessages increments the message counter.
func (s *BrowserSession) IncrementMessages() {
	s.MessageCount++
}

// Close closes the underlying browser page.
func (s *BrowserSession) Close() error {
	if s.Page == nil {
		return nil
	}
	// Page is playwright.Page at runtime
	type pageCloser interface {
		Close(opts ...interface{}) error
	}
	if c, ok := s.Page.(pageCloser); ok {
		return c.Close()
	}
	return nil
}

// Reset reloads the page and resets the message count.
func (s *BrowserSession) Reset() error {
	s.MessageCount = 0
	if s.Page == nil {
		return nil
	}
	type pageReloader interface {
		Reload(opts ...interface{}) (interface{}, error)
	}
	if r, ok := s.Page.(pageReloader); ok {
		_, err := r.Reload()
		return err
	}
	return nil
}
