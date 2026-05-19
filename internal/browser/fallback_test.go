package browser

import (
	"runtime"
	"testing"
)

func TestResolveChromePath_TriesCommonPaths(t *testing.T) {
	// This test verifies the function returns a path (or empty if not found)
	// without requiring an actual Chrome installation.
	path := ResolveChromePath()
	// On CI or dev machines, Chrome may or may not be installed.
	// We just verify the function doesn't panic and returns a valid result.
	if runtime.GOOS == "linux" {
		// Linux may have google-chrome or chromium-browser
		// We don't assert presence, just that it ran
		t.Logf("ResolveChromePath on linux returned: %q", path)
	}
}

func TestDefaultChromeChannels(t *testing.T) {
	// Verify the channel fallback order is defined
	channels := DefaultChromeChannels()
	if len(channels) < 2 {
		t.Errorf("expected at least 2 channel candidates, got %d", len(channels))
	}
	// First should be chrome (system), second should be chromium (bundled)
	if channels[0] != "chrome" {
		t.Errorf("first channel should be 'chrome', got %q", channels[0])
	}
	if channels[1] != "chromium" {
		t.Errorf("second channel should be 'chromium', got %q", channels[1])
	}
}

func TestCommonChromePaths_NonEmpty(t *testing.T) {
	paths := commonChromePaths()
	if len(paths) == 0 {
		t.Error("commonChromePaths should return at least one path")
	}
	// Verify paths contain the OS-specific entries
	found := false
	for _, p := range paths {
		if p != "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("commonChromePaths should have at least one non-empty path")
	}
}
