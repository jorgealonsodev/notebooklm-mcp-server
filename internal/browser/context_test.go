package browser

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jorge/notebooklm-mcp-server/internal/config"
)

// ---- Mock types for Playwright interfaces ----

type mockPage struct {
	closed bool
}

func (m *mockPage) Close(opts ...interface{}) error {
	m.closed = true
	return nil
}

type mockBrowserContext struct {
	pages   []*mockPage
	closed  bool
	newPage func() (*mockPage, error)
}

func (m *mockBrowserContext) NewPage(opts ...interface{}) (*mockPage, error) {
	if m.newPage != nil {
		return m.newPage()
	}
	page := &mockPage{}
	m.pages = append(m.pages, page)
	return page, nil
}

func (m *mockBrowserContext) Close(opts ...interface{}) error {
	m.closed = true
	return nil
}

type mockBrowser struct {
	closed bool
}

func (m *mockBrowser) Close(opts ...interface{}) error {
	m.closed = true
	return nil
}

// mockBrowserType simulates channel-based launch behavior.
type mockBrowserType struct {
	launchErr map[string]error // channel -> error
	ctx       *mockBrowserContext
	browser   *mockBrowser
}

func (m *mockBrowserType) LaunchPersistentContext(userDataDir string, opts interface{}) (*mockBrowserContext, error) {
	// Extract channel from opts (simplified for testing)
	channel := ""
	if co, ok := opts.(interface{ GetChannel() string }); ok {
		channel = co.GetChannel()
	}
	if err, found := m.launchErr[channel]; found {
		return nil, err
	}
	return m.ctx, nil
}

// ---- Lock file tests ----

func TestAcquireLock_CreatesLockFile(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, ".lock")

	release, err := acquireLock(lockPath)
	if err != nil {
		t.Fatalf("acquireLock() error = %v", err)
	}
	defer release()

	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("lock file should have been created")
	}
}

func TestAcquireLock_FailsWhenAlreadyLocked(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, ".lock")

	release1, err := acquireLock(lockPath)
	if err != nil {
		t.Fatalf("first acquireLock() error = %v", err)
	}
	defer release1()

	_, err = acquireLock(lockPath)
	if err == nil {
		t.Error("second acquireLock() should fail when lock is held")
	}
}

func TestReleaseLock_RemovesLockFile(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, ".lock")

	release, err := acquireLock(lockPath)
	if err != nil {
		t.Fatalf("acquireLock() error = %v", err)
	}

	release()

	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("lock file should have been removed after release")
	}
}

// ---- NewSharedContextManager tests ----

func TestNewSharedContextManager_SetsDefaults(t *testing.T) {
	cfg := config.Config{
		DataDir:          t.TempDir(),
		BrowserStateDir:  t.TempDir(),
		ChromeProfileDir: t.TempDir(),
		ProfileStrategy:  config.ProfileStrategyAuto,
	}

	mgr := NewSharedContextManager(cfg)
	if mgr == nil {
		t.Fatal("NewSharedContextManager should not return nil")
	}
}

// ---- Integration test (requires real Playwright) ----

func TestSharedContextManager_Launch_Short(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	cfg := config.Config{
		DataDir:          dir,
		BrowserStateDir:  dir,
		ChromeProfileDir: dir,
		ProfileStrategy:  config.ProfileStrategyIsolated,
		Headless:         true,
		Viewport:         config.Viewport{Width: 1920, Height: 1080},
	}

	mgr := NewSharedContextManager(cfg)
	ctx := context.Background()

	err := mgr.Launch(ctx)
	if err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	defer mgr.Close()

	// Verify we can create a page
	page, err := mgr.NewPage(ctx)
	if err != nil {
		t.Fatalf("NewPage() error = %v", err)
	}
	if page == nil {
		t.Fatal("NewPage() returned nil page")
	}

	// Verify healthy
	if !mgr.Healthy() {
		t.Error("manager should be healthy after launch")
	}
}

func TestSharedContextManager_Close_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		DataDir:          dir,
		BrowserStateDir:  dir,
		ChromeProfileDir: dir,
		ProfileStrategy:  config.ProfileStrategyIsolated,
		Headless:         true,
	}

	mgr := NewSharedContextManager(cfg)

	// Close without launch should not panic
	err := mgr.Close()
	if err != nil {
		t.Errorf("Close() without launch should not error, got %v", err)
	}

	// Second close should also not error
	err = mgr.Close()
	if err != nil {
		t.Errorf("second Close() should not error, got %v", err)
	}
}

func TestSharedContextManager_NewPage_BeforeLaunch(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		DataDir:          dir,
		BrowserStateDir:  dir,
		ChromeProfileDir: dir,
	}

	mgr := NewSharedContextManager(cfg)
	_, err := mgr.NewPage(context.Background())
	if err == nil {
		t.Error("NewPage before launch should return an error")
	}
}

// ---- Anti-detection args tests ----

func TestAntiDetectionArgs_ContainsRequired(t *testing.T) {
	args := antiDetectionArgs()
	required := []string{
		"--disable-blink-features=AutomationControlled",
		"--disable-dev-shm-usage",
		"--no-first-run",
		"--no-default-browser-check",
	}
	for _, req := range required {
		found := false
		for _, arg := range args {
			if arg == req {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("antiDetectionArgs missing required arg: %s", req)
		}
	}
}

// ---- Channel fallback tests ----

func TestTryLaunchChannels_TriesChromeFirst(t *testing.T) {
	launchAttempts := []string{}
	tryLaunch := func(channel string) error {
		launchAttempts = append(launchAttempts, channel)
		if channel == "chrome" {
			return errors.New("chrome not available")
		}
		return nil
	}

	channels := []string{"chrome", "chromium"}
	err := tryLaunchWithFallback(channels, tryLaunch)
	if err != nil {
		t.Errorf("tryLaunchWithFallback() error = %v", err)
	}
	if len(launchAttempts) != 2 {
		t.Errorf("expected 2 launch attempts, got %d", len(launchAttempts))
	}
	if launchAttempts[0] != "chrome" {
		t.Errorf("first attempt should be chrome, got %s", launchAttempts[0])
	}
	if launchAttempts[1] != "chromium" {
		t.Errorf("second attempt should be chromium, got %s", launchAttempts[1])
	}
}

func TestTryLaunchChannels_ReturnsErrorWhenAllFail(t *testing.T) {
	attempts := 0
	tryLaunch := func(channel string) error {
		attempts++
		return errors.New("not available")
	}

	channels := []string{"chrome", "chromium"}
	err := tryLaunchWithFallback(channels, tryLaunch)
	if err == nil {
		t.Error("expected error when all channels fail")
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestTryLaunchChannels_SucceedsOnFirst(t *testing.T) {
	attempts := 0
	tryLaunch := func(channel string) error {
		attempts++
		return nil
	}

	channels := []string{"chrome", "chromium"}
	err := tryLaunchWithFallback(channels, tryLaunch)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt (success on first), got %d", attempts)
	}
}
