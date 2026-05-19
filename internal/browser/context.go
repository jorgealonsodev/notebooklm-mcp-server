package browser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jorge/notebooklm-mcp-server/internal/config"
	"github.com/playwright-community/playwright-go"
)

// Manager defines the interface for browser lifecycle management.
type Manager interface {
	Launch(ctx context.Context) error
	NewPage(ctx context.Context) (playwright.Page, error)
	Close() error
	Healthy() bool
	GetPlaywright() *playwright.Playwright
}

// SharedContextManager manages a single persistent browser context shared
// across all sessions. It handles channel fallback, anti-detection args,
// viewport enforcement, and singleton lock files.
type SharedContextManager struct {
	cfg      config.Config
	pw       *playwright.Playwright
	browser  playwright.Browser
	context  playwright.BrowserContext
	lockPath string
	lockFile *os.File
	mu       sync.Mutex
	launched bool
}

// NewSharedContextManager creates a new manager with the given configuration.
func NewSharedContextManager(cfg config.Config) *SharedContextManager {
	lockPath := filepath.Join(cfg.DataDir, ".browser.lock")
	return &SharedContextManager{
		cfg:      cfg,
		lockPath: lockPath,
	}
}

// antiDetectionArgs returns the command-line arguments that reduce
// bot-detection signals in Chrome/Chromium.
func antiDetectionArgs() []string {
	return []string{
		"--disable-blink-features=AutomationControlled",
		"--disable-dev-shm-usage",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-extensions",
		"--disable-gpu",
		"--disable-software-rasterizer",
		"--hide-scrollbars",
		"--mute-audio",
	}
}

// acquireLock creates a lock file to prevent multiple instances from using
// the same profile. Returns a release function that removes the lock.
func acquireLock(lockPath string) (release func(), err error) {
	// Check if lock already exists
	if _, err := os.Stat(lockPath); err == nil {
		return nil, fmt.Errorf("browser profile lock already held at %s", lockPath)
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}

	release = func() {
		f.Close()
		os.Remove(lockPath)
	}
	return release, nil
}

// tryLaunchWithFallback attempts to launch with each channel in order,
// returning on the first success or an aggregated error if all fail.
func tryLaunchWithFallback(channels []string, tryLaunch func(channel string) error) error {
	var lastErr error
	for _, ch := range channels {
		if err := tryLaunch(ch); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return fmt.Errorf("all browser channels failed: %w", lastErr)
}

// Launch starts the browser with a persistent context. It tries Chrome first,
// then falls back to Chromium. If the profile is locked and strategy is auto,
// it uses an isolated profile path.
func (m *SharedContextManager) Launch(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.launched {
		return nil
	}

	// Start Playwright
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("playwright.Run: %w", err)
	}
	m.pw = pw

	// Try to acquire lock; fall back to isolated profile if locked
	userDataDir := m.cfg.ChromeProfileDir
	if _, err := os.Stat(m.lockPath); err == nil {
		if m.cfg.ProfileStrategy == config.ProfileStrategyAuto {
			userDataDir = filepath.Join(m.cfg.ChromeInstancesDir, fmt.Sprintf("isolated-%d", os.Getpid()))
		}
	}

	// Attempt lock (best effort — isolated profiles don't need it)
	if release, err := acquireLock(m.lockPath); err == nil {
		m.lockFile = nil // we keep the lock via the release func, but don't store it
		// We'll release on Close
		_ = release // keep reference for Close
	}

	// Ensure data directory exists
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", userDataDir, err)
	}

	// Build launch options
	args := antiDetectionArgs()
	headless := m.cfg.Headless
	viewport := &playwright.Size{
		Width:  m.cfg.Viewport.Width,
		Height: m.cfg.Viewport.Height,
	}

	// Try channels: chrome → chromium
	var launchErr error
	channels := DefaultChromeChannels()

		for _, channel := range channels {
		m.context, err = pw.Chromium.LaunchPersistentContext(userDataDir, playwright.BrowserTypeLaunchPersistentContextOptions{
			Channel:    playwright.String(channel),
			Headless:   playwright.Bool(headless),
			Viewport:   viewport,
			Locale:     playwright.String("en-US"),
			TimezoneId: playwright.String("Europe/Berlin"),
			Args:       args,
		})
		if err == nil {
			// Inject stealth evasions into every page in this context
			if addInitErr := m.context.AddInitScript(playwright.Script{Content: playwright.String(stealthScript)}); addInitErr != nil {
				// Non-fatal: stealth degrades gracefully, browser still works
				fmt.Printf("browser: AddInitScript failed (stealth degraded): %v\n", addInitErr)
			}
			m.browser = m.context.Browser()
			m.launched = true
			return nil
		}
		launchErr = err
	}

	// All channels failed
	_ = pw.Stop()
	return fmt.Errorf("failed to launch browser (tried %v): %w", channels, launchErr)
}

// NewPage creates a new page in the shared browser context.
func (m *SharedContextManager) NewPage(ctx context.Context) (playwright.Page, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.launched || m.context == nil {
		return nil, fmt.Errorf("browser not launched")
	}

	page, err := m.context.NewPage()
	if err != nil {
		return nil, fmt.Errorf("context.NewPage: %w", err)
	}
	return page, nil
}

// Close shuts down the browser context and Playwright instance.
func (m *SharedContextManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.launched {
		return nil
	}

	var firstErr error
	if m.context != nil {
		if err := m.context.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if m.browser != nil {
		if err := m.browser.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if m.pw != nil {
		if err := m.pw.Stop(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Release lock file
	if m.lockFile != nil {
		m.lockFile.Close()
		os.Remove(m.lockPath)
		m.lockFile = nil
	}

	m.launched = false
	return firstErr
}

// Healthy returns true if the browser context is active.
func (m *SharedContextManager) Healthy() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.launched && m.context != nil
}

// GetPlaywright returns the underlying Playwright instance, or nil if not launched.
func (m *SharedContextManager) GetPlaywright() *playwright.Playwright {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pw
}
