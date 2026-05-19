package auth

import (
	"path/filepath"
	"testing"

	"github.com/jorge/notebooklm-mcp-server/internal/config"
)

func TestAccountPaths(t *testing.T) {
	tests := []struct {
		name         string
		account      string
		dataDir      string
		wantStateDir string
		wantProfile  string
	}{
		{
			name:         "default account (empty)",
			account:      "",
			dataDir:      "/home/user/data",
			wantStateDir: "/home/user/data/browser_state",
			wantProfile:  "/home/user/data/chrome_profile",
		},
		{
			name:         "named account",
			account:      "work",
			dataDir:      "/home/user/data",
			wantStateDir: "/home/user/data/accounts/work/browser_state",
			wantProfile:  "/home/user/data/accounts/work/chrome_profile",
		},
		{
			name:         "another named account",
			account:      "personal",
			dataDir:      "/home/user/data",
			wantStateDir: "/home/user/data/accounts/personal/browser_state",
			wantProfile:  "/home/user/data/accounts/personal/chrome_profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				DataDir:          tt.dataDir,
				BrowserStateDir:  filepath.Join(tt.dataDir, "browser_state"),
				ChromeProfileDir: filepath.Join(tt.dataDir, "chrome_profile"),
			}

			got := ResolveAccountPaths(cfg, tt.account)

			if got.BrowserStateDir != tt.wantStateDir {
				t.Errorf("BrowserStateDir = %q, want %q", got.BrowserStateDir, tt.wantStateDir)
			}
			if got.ChromeProfileDir != tt.wantProfile {
				t.Errorf("ChromeProfileDir = %q, want %q", got.ChromeProfileDir, tt.wantProfile)
			}
		})
	}
}

func TestAccountPathIsolation(t *testing.T) {
	// Verify that different accounts produce different paths
	cfg := config.Config{
		DataDir:          "/data",
		BrowserStateDir:  "/data/browser_state",
		ChromeProfileDir: "/data/chrome_profile",
	}

	a := ResolveAccountPaths(cfg, "alice")
	b := ResolveAccountPaths(cfg, "bob")

	if a.BrowserStateDir == b.BrowserStateDir {
		t.Error("different accounts should have different state dirs")
	}
	if a.ChromeProfileDir == b.ChromeProfileDir {
		t.Error("different accounts should have different profile dirs")
	}
}

func TestAccountPathsDefaultMatchesBase(t *testing.T) {
	cfg := config.Config{
		DataDir:          "/data",
		BrowserStateDir:  "/data/browser_state",
		ChromeProfileDir: "/data/chrome_profile",
	}

	got := ResolveAccountPaths(cfg, "")

	// Default account should return paths matching the base config
	if got.BrowserStateDir != cfg.BrowserStateDir {
		t.Errorf("default account BrowserStateDir = %q, want %q", got.BrowserStateDir, cfg.BrowserStateDir)
	}
	if got.ChromeProfileDir != cfg.ChromeProfileDir {
		t.Errorf("default account ChromeProfileDir = %q, want %q", got.ChromeProfileDir, cfg.ChromeProfileDir)
	}
}
