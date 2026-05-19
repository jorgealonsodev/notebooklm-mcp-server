package browser

import (
	"os"
	"runtime"
)

// DefaultChromeChannels returns the fallback order for browser channels.
// Tries system Chrome first, then bundled Chromium.
func DefaultChromeChannels() []string {
	return []string{"chrome", "chromium"}
}

// commonChromePaths returns OS-specific common Chrome executable paths.
func commonChromePaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	case "windows":
		return []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		}
	default: // linux
		return []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium-browser",
			"/usr/bin/chromium",
			"/snap/bin/chromium",
			"/snap/chromium/current/chrome-wrapper",
		}
	}
}

// ResolveChromePath searches common paths for a Chrome executable.
// Returns the first found path, or empty string if none exists.
func ResolveChromePath() string {
	for _, path := range commonChromePaths() {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}
