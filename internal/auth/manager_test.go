package auth

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jorge/notebooklm-mcp-server/internal/config"
)

func TestNewManager(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		DataDir:          dir,
		BrowserStateDir:  dir,
		ChromeProfileDir: dir,
	}

	mgr := NewManager(cfg)
	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestManagerValidate_NoStateFile(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		DataDir:         dir,
		BrowserStateDir: dir,
	}

	mgr := NewManager(cfg)
	err := mgr.Validate(time.Now())
	if err == nil {
		t.Error("Validate() should error when no state file exists")
	}
}

func TestManagerValidate_ValidState(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		DataDir:         dir,
		BrowserStateDir: dir,
	}

	// Write valid state with all 9 Google cookies
	state := &BrowserState{
		Cookies: makeGoogleCookies(time.Now().Add(1 * time.Hour)),
		Origins: []OriginState{},
	}
	if err := SaveState(dir, state); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(cfg)
	err := mgr.Validate(time.Now())
	if err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestManagerValidate_ExpiredState(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		DataDir:         dir,
		BrowserStateDir: dir,
	}

	// Write valid cookies but with old file timestamp
	state := &BrowserState{
		Cookies: makeGoogleCookies(time.Now().Add(1 * time.Hour)),
		Origins: []OriginState{},
	}
	if err := SaveState(dir, state); err != nil {
		t.Fatal(err)
	}

	// Make file older than 24h
	oldTime := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(dir+"/state.json", oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(cfg)
	err := mgr.Validate(time.Now())
	if err == nil {
		t.Error("Validate() should error for state older than 24h")
	}
}

func TestManagerClearAllAuthData(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		DataDir:          dir,
		BrowserStateDir:  dir,
		ChromeProfileDir: dir,
	}

	// Write state file
	state := &BrowserState{
		Cookies: makeGoogleCookies(time.Now().Add(1 * time.Hour)),
		Origins:  []OriginState{},
	}
	if err := SaveState(dir, state); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(cfg)
	err := mgr.ClearAllAuthData()
	if err != nil {
		t.Errorf("ClearAllAuthData() error: %v", err)
	}

	// Verify state file is deleted
	statePath := dir + "/state.json"
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Error("state file should be deleted after ClearAllAuthData")
	}
}

func TestManagerValidate_MissingCookies(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		DataDir:         dir,
		BrowserStateDir: dir,
	}

	// Write state with only 3 cookies (not enough)
	state := &BrowserState{
		Cookies: []CookieState{
			{Name: "SID", Value: "x", Domain: ".google.com", Path: "/", Expires: float64(time.Now().Add(1 * time.Hour).Unix())},
			{Name: "HSID", Value: "x", Domain: ".google.com", Path: "/", Expires: float64(time.Now().Add(1 * time.Hour).Unix())},
			{Name: "SSID", Value: "x", Domain: ".google.com", Path: "/", Expires: float64(time.Now().Add(1 * time.Hour).Unix())},
		},
		Origins: []OriginState{},
	}
	if err := SaveState(dir, state); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(cfg)
	err := mgr.Validate(time.Now())
	if err == nil {
		t.Error("Validate() should error when not all 9 Google cookies are present")
	}
}

// mockPage implements pageLike for unit testing URL polling.
type mockPage struct {
	mu      sync.Mutex
	urls    []string
	idx     int
	gotoURL string
}

func (m *mockPage) URL() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.idx >= len(m.urls) {
		return m.urls[len(m.urls)-1]
	}
	u := m.urls[m.idx]
	m.idx++
	return u
}

func (m *mockPage) Goto(url string, opts ...interface{}) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gotoURL = url
	return nil, nil
}

func TestWaitForNotebookLM_Success(t *testing.T) {
	page := &mockPage{
		urls: []string{
			"https://accounts.google.com/signin",
			"https://notebooklm.google.com/",
		},
	}

	ctx := context.Background()
	err := waitForNotebookLM(ctx, page, 5*time.Second)
	if err != nil {
		t.Errorf("waitForNotebookLM() unexpected error: %v", err)
	}
}

func TestWaitForNotebookLM_Timeout(t *testing.T) {
	page := &mockPage{
		urls: []string{
			"https://accounts.google.com/signin",
			"https://accounts.google.com/signin/v2",
		},
	}

	ctx := context.Background()
	err := waitForNotebookLM(ctx, page, 200*time.Millisecond)
	if err == nil {
		t.Error("waitForNotebookLM() expected timeout error, got nil")
	}
}

func TestWaitForNotebookLM_ContextCancelled(t *testing.T) {
	page := &mockPage{
		urls: []string{
			"https://accounts.google.com/signin",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := waitForNotebookLM(ctx, page, 10*time.Second)
	if err == nil {
		t.Error("waitForNotebookLM() expected cancellation error, got nil")
	}
}

func TestWaitForNotebookLM_NotebookLMWithAccountsParam(t *testing.T) {
	// URL that contains both "notebooklm.google" and "accounts.google" should NOT match
	page := &mockPage{
		urls: []string{
			"https://notebooklm.google.com/?authuser=accounts.google.com",
			"https://notebooklm.google.com/",
		},
	}

	ctx := context.Background()
	err := waitForNotebookLM(ctx, page, 5*time.Second)
	if err != nil {
		t.Errorf("waitForNotebookLM() unexpected error: %v", err)
	}
}

func TestPerformSetup_NilPlaywright_LaunchesOwnInstance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	cfg := config.Config{
		BrowserStateDir: dir,
		Viewport:        config.Viewport{Width: 1280, Height: 720},
		SetupTimeoutMs:  5000, // 5 seconds for faster test
	}

	mgr := NewManager(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// This will launch a browser and wait for user login.
	// Without user interaction, it should timeout.
	err := mgr.PerformSetup(ctx, nil, cfg)
	if err == nil {
		t.Error("PerformSetup() should timeout without user interaction")
	}
}

func TestPerformSetup_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	cfg := config.Config{
		BrowserStateDir: dir,
		Viewport:        config.Viewport{Width: 1280, Height: 720},
		SetupTimeoutMs:  600_000, // 10 minutes for real user interaction
	}

	mgr := NewManager(cfg)
	ctx := context.Background()

	t.Log("Launching headful browser for interactive setup...")
	t.Log("Please complete Google login in the browser window.")

	err := mgr.PerformSetup(ctx, nil, cfg)
	if err != nil {
		t.Fatalf("PerformSetup() error: %v", err)
	}

	// Verify state file was created
	statePath := dir + "/state.json"
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("state.json should exist after successful setup")
	}
}
