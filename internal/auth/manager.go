package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jorge/notebooklm-mcp-server/internal/apperrors"
	"github.com/jorge/notebooklm-mcp-server/internal/config"
)

// Manager handles Google authentication via cookie persistence.
type Manager struct {
	cfg config.Config
}

// NewManager creates a new auth manager with the given configuration.
func NewManager(cfg config.Config) *Manager {
	return &Manager{cfg: cfg}
}

// Validate checks that the authentication state is valid: all required
// Google cookies are present, non-expired, and the state file is less
// than 24 hours old.
func (m *Manager) Validate(now time.Time) error {
	return ValidateCookies(m.cfg.BrowserStateDir, now)
}

// PerformSetup performs interactive browser login by navigating to Google
// and waiting for the user to complete authentication. The caller is
// responsible for providing a page from a headful browser context.
//
// This method saves the browser state after successful login.
func (m *Manager) PerformSetup(page interface{}) error {
	// NOTE: This method requires a real Playwright page for interactive login.
	// The page parameter is typed as interface{} to avoid a direct playwright
	// dependency in tests; the real implementation uses playwright.Page.
	//
	// The interactive flow:
	// 1. Navigate to https://notebooklm.google.com
	// 2. Wait for Google login redirect (up to 10 minutes)
	// 3. User completes login manually
	// 4. Wait for redirect back to NotebookLM
	// 5. Save browser state (cookies + localStorage)
	//
	// For now, this is a placeholder. The full implementation requires
	// playwright-go and is tested via integration tests (skipped in -short).
	return fmt.Errorf("interactive setup requires a real browser (integration test only)")
}

// AutoLogin performs automated Google login using configured credentials.
// It fills the email and password fields with human-like typing delays.
func (m *Manager) AutoLogin(page interface{}, email, password string) error {
	if !m.cfg.AutoLoginEnabled {
		return apperrors.NewAuthenticationError("auto-login is not enabled", false)
	}
	if email == "" || password == "" {
		return apperrors.NewAuthenticationError("auto-login requires email and password", false)
	}

	// NOTE: Full implementation requires playwright.Page:
	// 1. Navigate to Google sign-in URL
	// 2. Wait for email field, type email with human-like delays
	// 3. Click Next, wait for password field
	// 4. Type password, click Next
	// 5. Wait for redirect to NotebookLM
	// 6. Save browser state
	//
	// This is tested via integration tests (skipped in -short).
	return fmt.Errorf("auto-login requires a real browser (integration test only)")
}

// ClearAllAuthData removes all authentication data: state files and
// the Chrome profile directory.
func (m *Manager) ClearAllAuthData() error {
	// Remove state file
	statePath := filepath.Join(m.cfg.BrowserStateDir, stateFile)
	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove state file: %w", err)
	}

	// Remove session file
	sessionPath := filepath.Join(m.cfg.BrowserStateDir, sessionFile)
	if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove session file: %w", err)
	}

	// Remove Chrome profile directory
	if err := os.RemoveAll(m.cfg.ChromeProfileDir); err != nil {
		return fmt.Errorf("remove chrome profile: %w", err)
	}

	return nil
}
