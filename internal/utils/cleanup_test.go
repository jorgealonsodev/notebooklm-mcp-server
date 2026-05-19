package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDeepCleanup_RemovesAuthState(t *testing.T) {
	dir := t.TempDir()

	// Create auth state files
	stateDir := filepath.Join(dir, "browser_state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "state.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "session.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DeepCleanup(dir, false)
	if err != nil {
		t.Fatalf("DeepCleanup: %v", err)
	}

	// Verify state files are deleted
	if _, err := os.Stat(filepath.Join(stateDir, "state.json")); !os.IsNotExist(err) {
		t.Error("state.json should be deleted")
	}
	if _, err := os.Stat(filepath.Join(stateDir, "session.json")); !os.IsNotExist(err) {
		t.Error("session.json should be deleted")
	}

	if len(result.DeletedPaths) == 0 {
		t.Error("expected some deleted paths")
	}
}

func TestDeepCleanup_PreservesLibrary(t *testing.T) {
	dir := t.TempDir()

	// Create library.json and other files
	libPath := filepath.Join(dir, "library.json")
	if err := os.WriteFile(libPath, []byte(`{"notebooks":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(dir, "browser_state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "state.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DeepCleanup(dir, true)
	if err != nil {
		t.Fatalf("DeepCleanup: %v", err)
	}

	// Verify library.json is preserved
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		t.Error("library.json should be preserved")
	}

	// Verify other files are deleted
	if _, err := os.Stat(filepath.Join(stateDir, "state.json")); !os.IsNotExist(err) {
		t.Error("state.json should be deleted")
	}

	_ = result
}

func TestDeepCleanup_RemovesChromeProfiles(t *testing.T) {
	dir := t.TempDir()

	// Create chrome profile directories
	profileDir := filepath.Join(dir, "chrome_profile")
	if err := os.MkdirAll(filepath.Join(profileDir, "Default"), 0755); err != nil {
		t.Fatal(err)
	}
	instanceDir := filepath.Join(dir, "chrome_profile_instances", "isolated-123")
	if err := os.MkdirAll(instanceDir, 0755); err != nil {
		t.Fatal(err)
	}

	result, err := DeepCleanup(dir, false)
	if err != nil {
		t.Fatalf("DeepCleanup: %v", err)
	}

	// Verify chrome profiles are deleted
	if _, err := os.Stat(profileDir); !os.IsNotExist(err) {
		t.Error("chrome_profile should be deleted")
	}

	_ = result
}

func TestDeepCleanup_RemovesLockFile(t *testing.T) {
	dir := t.TempDir()

	lockPath := filepath.Join(dir, ".browser.lock")
	if err := os.WriteFile(lockPath, []byte("123"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := DeepCleanup(dir, false)
	if err != nil {
		t.Fatalf("DeepCleanup: %v", err)
	}

	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error(".browser.lock should be deleted")
	}
}

func TestDeepCleanup_RemovesTempFiles(t *testing.T) {
	dir := t.TempDir()

	tmpFile := filepath.Join(dir, "test.tmp")
	if err := os.WriteFile(tmpFile, []byte("temp"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := DeepCleanup(dir, false)
	if err != nil {
		t.Fatalf("DeepCleanup: %v", err)
	}

	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error(".tmp file should be deleted")
	}
}

func TestDeepCleanup_EmptyDataDir(t *testing.T) {
	_, err := DeepCleanup("", false)
	if err == nil {
		t.Error("expected error for empty dataDir")
	}
}

func TestIsCleanupTarget(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		{"state.json", "/data/browser_state/state.json", false, true},
		{"session.json", "/data/browser_state/session.json", false, true},
		{"chrome_profile dir", "/data/chrome_profile", true, true},
		{"isolated profile", "/data/chrome_profile_instances/isolated-123", true, true},
		{"lock file", "/data/.browser.lock", false, true},
		{"tmp file", "/data/test.tmp", false, true},
		{"library.json", "/data/library.json", false, false},
		{"random file", "/data/some_file.txt", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &fakeFileInfo{name: filepath.Base(tt.path), isDir: tt.isDir}
			got := isCleanupTarget(tt.path, info)
			if got != tt.expected {
				t.Errorf("isCleanupTarget(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

type fakeFileInfo struct {
	name  string
	isDir bool
}

func (f *fakeFileInfo) Name() string       { return f.name }
func (f *fakeFileInfo) Size() int64        { return 0 }
func (f *fakeFileInfo) Mode() os.FileMode  { return 0644 }
func (f *fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f *fakeFileInfo) IsDir() bool        { return f.isDir }
func (f *fakeFileInfo) Sys() interface{}   { return nil }
