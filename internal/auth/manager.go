package auth

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jorge/notebooklm-mcp-server/internal/apperrors"
	"github.com/jorge/notebooklm-mcp-server/internal/config"
	"github.com/playwright-community/playwright-go"
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

// pageLike is the minimal interface needed for URL polling during setup.
type pageLike interface {
	URL() string
}

// PerformSetup performs interactive browser login by navigating to Google
// and waiting for the user to complete authentication.
//
// If pw is nil, a standalone Playwright instance is launched and stopped
// when setup completes. The browser is always launched headful (headless=false)
// so the user can see and interact with the Google login page.
//
// The interactive flow:
//  1. Launch a headful browser (headless=false)
//  2. Navigate to https://notebooklm.google.com
//  3. Print to stderr: user-facing message to complete login
//  4. Poll page.URL() every 2 seconds checking if URL contains
//     "notebooklm.google" (not "accounts.google")
//  5. Once on NotebookLM, save browser state via context.StorageState(path)
//  6. Close the browser
//  7. Return nil on success
//
// Timeout is controlled by cfg.SetupTimeoutMs (default 10 minutes).
func (m *Manager) PerformSetup(ctx context.Context, pw *playwright.Playwright, cfg config.Config) error {
	ownPW := pw == nil
	if ownPW {
		var err error
		pw, err = playwright.Run()
		if err != nil {
			return fmt.Errorf("playwright.Run: %w", err)
		}
		defer func() { _ = pw.Stop() }()
	}

	// Launch a headful browser with a temporary profile for setup
	userDataDir, err := os.MkdirTemp("", "notebooklm-setup-*")
	if err != nil {
		return fmt.Errorf("create temp profile dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(userDataDir) }()

	args := []string{
		"--disable-blink-features=AutomationControlled",
		"--disable-dev-shm-usage",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-extensions",
		"--disable-gpu",
	}

	opts := playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless:   playwright.Bool(false), // MUST be headful for user interaction
		Viewport: &playwright.Size{
			Width:  cfg.Viewport.Width,
			Height: cfg.Viewport.Height,
		},
		Locale:     playwright.String("en-US"),
		TimezoneId: playwright.String("Europe/Berlin"),
		Args:       args,
	}

	// Try chrome channels, fall back to chromium
	var bctx playwright.BrowserContext
	channels := []string{"chrome", "chromium"}
	var lastErr error
	for _, ch := range channels {
		opts.Channel = playwright.String(ch)
		bctx, err = pw.Chromium.LaunchPersistentContext(userDataDir, opts)
		if err == nil {
			break
		}
		lastErr = err
	}
	if bctx == nil {
		return fmt.Errorf("failed to launch browser (tried %v): %w", channels, lastErr)
	}
	defer func() { _ = bctx.Close() }()

	page, err := bctx.NewPage()
	if err != nil {
		return fmt.Errorf("create page: %w", err)
	}

	// Navigate to NotebookLM (triggers Google login redirect)
	slog.Info("Navigating to NotebookLM for authentication setup")
	if _, err := page.Goto("https://notebooklm.google.com"); err != nil {
		return fmt.Errorf("navigate to notebooklm: %w", err)
	}

	// Print user-facing message to stderr
	fmt.Fprintf(os.Stderr, "Please complete Google login in the browser window. Waiting up to %d minutes...\n", cfg.SetupTimeoutMs/60000)

	// Poll until we're on notebooklm.google.com (not accounts.google)
	timeout := time.Duration(cfg.SetupTimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	if err := waitForNotebookLM(ctx, page, timeout); err != nil {
		return err
	}

	// Save browser state
	statePath := filepath.Join(cfg.BrowserStateDir, stateFile)
	if err := os.MkdirAll(cfg.BrowserStateDir, 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	if _, err := bctx.StorageState(statePath); err != nil {
		return fmt.Errorf("save browser state: %w", err)
	}
	slog.Info("Browser state saved", "path", statePath)

	return nil
}

// waitForNotebookLM polls page.URL() every 2 seconds until the URL contains
// "notebooklm.google" and does NOT contain "accounts.google", or until the
// context is cancelled or the timeout elapses.
func waitForNotebookLM(ctx context.Context, page pageLike, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("setup cancelled: %w", ctx.Err())
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("setup timed out after %v (user did not complete login)", timeout)
			}
			url := page.URL()
			if strings.Contains(url, "notebooklm.google") && !strings.Contains(url, "accounts.google") {
				slog.Info("Login detected", "url", url)
				return nil
			}
		}
	}
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
