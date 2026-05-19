// Package session provides concurrent-safe browser session lifecycle management
// with automatic timeout cleanup.
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/jorge/notebooklm-mcp-server/internal/config"
)

// BrowserManager defines the browser operations needed by the session manager.
type BrowserManager interface {
	Launch(ctx context.Context) error
	NewPage(ctx context.Context) (pageOps, error)
	Close() error
	Healthy() bool
}

// SessionInfo holds summary information about a session.
type SessionInfo struct {
	ID           string    `json:"id"`
	NotebookURL  string    `json:"notebook_url"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
	MessageCount int       `json:"message_count"`
}

// SessionStats holds aggregate session statistics.
type SessionStats struct {
	Active        int           `json:"active"`
	Max           int           `json:"max"`
	Timeout       time.Duration `json:"timeout"`
	Oldest        time.Time     `json:"oldest"`
	TotalMessages int           `json:"total_messages"`
}

// Manager manages browser sessions with lifecycle and timeout cleanup.
type Manager struct {
	cfg       config.Config
	browser   BrowserManager
	mu        sync.RWMutex
	sessions  map[string]*BrowserSession
	timeout   time.Duration
	stopCh    chan struct{}
	doneCh    chan struct{}
}

// NewManager creates a new session manager.
func NewManager(cfg config.Config, browser BrowserManager) *Manager {
	return &Manager{
		cfg:      cfg,
		browser:  browser,
		sessions: make(map[string]*BrowserSession),
		timeout:  time.Duration(cfg.SessionTimeoutSeconds) * time.Second,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Create opens a new page in the shared browser context and registers it
// as a new session.
func (m *Manager) Create(notebookURL string) (*BrowserSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sessions) >= m.cfg.MaxSessions {
		return nil, fmt.Errorf("max sessions reached (%d)", m.cfg.MaxSessions)
	}

	ctx := context.Background()
	page, err := m.browser.NewPage(ctx)
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}

	sess := NewBrowserSession(notebookURL)
	sess.Page = page
	m.sessions[sess.ID] = sess
	return sess, nil
}

// Get returns a session by ID.
func (m *Manager) Get(id string) (*BrowserSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sess, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %s not found", id)
	}
	return sess, nil
}

// Close closes a session's page and removes it from the registry.
func (m *Manager) Close(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}

	_ = sess.Close()
	delete(m.sessions, id)
	return nil
}

// Reset reloads the session's page and resets the message count.
func (m *Manager) Reset(id string) error {
	m.mu.RLock()
	sess, ok := m.sessions[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session %s not found", id)
	}

	return sess.Reset()
}

// List returns information about all active sessions.
func (m *Manager) List() []SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]SessionInfo, 0, len(m.sessions))
	for _, sess := range m.sessions {
		result = append(result, SessionInfo{
			ID:           sess.ID,
			NotebookURL:  sess.NotebookURL,
			CreatedAt:    sess.CreatedAt,
			LastActivity: sess.LastActivity,
			MessageCount: sess.MessageCount,
		})
	}
	return result
}

// Stats returns aggregate session statistics.
func (m *Manager) Stats() SessionStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := SessionStats{
		Active: len(m.sessions),
		Max:    m.cfg.MaxSessions,
		Timeout: m.timeout,
	}

	totalMsgs := 0
	for _, sess := range m.sessions {
		totalMsgs += sess.MessageCount
		if stats.Oldest.IsZero() || sess.CreatedAt.Before(stats.Oldest) {
			stats.Oldest = sess.CreatedAt
		}
	}
	stats.TotalMessages = totalMsgs
	return stats
}

// CloseAll closes all active sessions.
func (m *Manager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for id, sess := range m.sessions {
		if err := sess.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(m.sessions, id)
	}
	return firstErr
}

// StartCleanup starts the background goroutine that periodically removes
// expired sessions. The cleanup interval is calculated as
// max(60s, min(timeout/2, 300s)).
func (m *Manager) StartCleanup() {
	interval := cleanupInterval(m.timeout)
	go m.cleanupLoop(interval)
}

// StopCleanup stops the background cleanup goroutine.
func (m *Manager) StopCleanup() {
	close(m.stopCh)
	<-m.doneCh
}

func (m *Manager) cleanupLoop(interval time.Duration) {
	defer close(m.doneCh)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.cleanupExpired()
		}
	}
}

func (m *Manager) cleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, sess := range m.sessions {
		if now.Sub(sess.LastActivity) > m.timeout {
			_ = sess.Close()
			delete(m.sessions, id)
		}
	}
}

// cleanupInterval calculates the cleanup tick interval as
// max(60s, min(timeout/2, 300s)).
func cleanupInterval(timeout time.Duration) time.Duration {
	half := timeout / 2
	if half > 300*time.Second {
		half = 300 * time.Second
	}
	if half < 60*time.Second {
		half = 60 * time.Second
	}
	return half
}

// generateID creates a random 8-character hex string.
func generateID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
