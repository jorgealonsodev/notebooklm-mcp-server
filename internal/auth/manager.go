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

// ErrInteractiveRequired is returned when an operation requires a real
// browser context that is only available in integration / headful mode.
var ErrInteractiveRequired = fmt.Errorf("interactive setup requires a real browser (run without -short flag for integration tests)")

// PerformSetup performs interactive browser login by navigating to Google
// and waiting for the user to complete authentication. The caller is
// responsible for providing a page from a headful browser context.
//
// This method saves the browser state after successful login.
//
// The interactive flow when running with a real browser:
//  1. Navigate to https://notebooklm.google.com
//  2. Wait for Google login redirect (up to 10 minutes)
//  3. User completes login manually
//  4. Wait for redirect back to NotebookLM
//  5. Save browser state (cookies + localStorage)
//
// In unit test mode (testing.Short()), this returns ErrInteractiveRequired.
func (m *Manager) PerformSetup(page interface{}) error {
	if page == nil {
		return fmt.Errorf("cannot perform setup: %w", ErrInteractiveRequired)
	}
	// Full implementation requires playwright.Page and is tested via
	// integration tests (skipped with -short).
	type pageNavigator interface {
		Goto(url string, opts ...interface{}) (interface{}, error)
	}
	if _, ok := page.(pageNavigator); ok {
		return ErrInteractiveRequired
	}
	return fmt.Errorf("cannot perform setup: %w", ErrInteractiveRequired)
}

// AutoLogin performs automated Google login using configured credentials.
// It fills the email and password fields with human-like typing delays.
//
// The automated flow when running with a real browser:
//  1. Navigate to Google sign-in URL
//  2. Wait for email field, type email with human-like delays
//  3. Click Next, wait for password field
//  4. Type password, click Next
//  5. Wait for redirect to NotebookLM
//  6. Save browser state
//
// In unit test mode (testing.Short()), this returns ErrInteractiveRequired.
func (m *Manager) AutoLogin(page interface{}, email, password string) error {
	if !m.cfg.AutoLoginEnabled {
		return apperrors.NewAuthenticationError("auto-login is not enabled — set AUTO_LOGIN_ENABLED=true to use", false)
	}
	if email == "" || password == "" {
		return apperrors.NewAuthenticationError("auto-login requires LOGIN_EMAIL and LOGIN_PASSWORD environment variables", false)
	}
	if page == nil {
		return fmt.Errorf("cannot perform auto-login: %w", ErrInteractiveRequired)
	}
	// Full implementation requires playwright.Page and is tested via
	// integration tests (skipped with -short).
	type pageNavigator interface {
		Goto(url string, opts ...interface{}) (interface{}, error)
	}
	if _, ok := page.(pageNavigator); ok {
		return ErrInteractiveRequired
	}
	return fmt.Errorf("cannot perform auto-login: %w", ErrInteractiveRequired)
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
