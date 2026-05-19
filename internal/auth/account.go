package auth

import (
	"path/filepath"

	"github.com/jorge/notebooklm-mcp-server/internal/config"
)

// AccountPaths holds the resolved directory paths for a specific account.
// When account is empty, paths match the base config (default account).
type AccountPaths struct {
	BrowserStateDir  string
	ChromeProfileDir string
}

// ResolveAccountPaths resolves directory paths for the given account slug.
// An empty account slug returns the base config paths.
// A named account uses <dataDir>/accounts/<slug>/ as the root.
func ResolveAccountPaths(cfg config.Config, account string) AccountPaths {
	if account == "" {
		return AccountPaths{
			BrowserStateDir:  cfg.BrowserStateDir,
			ChromeProfileDir: cfg.ChromeProfileDir,
		}
	}

	accountRoot := filepath.Join(cfg.DataDir, "accounts", account)
	return AccountPaths{
		BrowserStateDir:  filepath.Join(accountRoot, "browser_state"),
		ChromeProfileDir: filepath.Join(accountRoot, "chrome_profile"),
	}
}
