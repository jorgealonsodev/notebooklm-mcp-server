package auth

import (
	"os"
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
