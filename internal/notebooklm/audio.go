package notebooklm

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/playwright-community/playwright-go"
)

// AudioStatus represents the state of an audio overview.
type AudioStatus string

const (
	AudioNotStarted AudioStatus = "not_started"
	AudioInProgress AudioStatus = "in_progress"
	AudioReady      AudioStatus = "ready"
)

// AudioResult holds the result of an audio generation request.
type AudioResult struct {
	Status    AudioStatus
	ElapsedMs int64
}

// DownloadResult holds the result of an audio download.
type DownloadResult struct {
	FilePath  string
	FileSize  int64
	ElapsedMs int64
}

// GenerateAudio generates an audio overview for the current notebook.
// If a custom prompt is provided, it opens the customize dialog.
// This operation is idempotent: if audio already exists, it returns immediately.
func GenerateAudio(page playwright.Page, customPrompt string, timeoutMs int) (*AudioResult, error) {
	start := time.Now()

	// Check if audio already exists (idempotent)
	status, err := GetAudioStatus(page)
	if err != nil {
		return nil, fmt.Errorf("check audio status: %w", err)
	}
	if status == AudioReady {
		return &AudioResult{Status: AudioReady, ElapsedMs: time.Since(start).Milliseconds()}, nil
	}

	// Check if generation is already in progress
	if status == AudioInProgress {
		return &AudioResult{Status: AudioInProgress, ElapsedMs: time.Since(start).Milliseconds()}, nil
	}

	// Click the Audio Overview button
	audioBtn, err := page.QuerySelector(AudioOverviewButton)
	if err != nil || audioBtn == nil {
		return nil, fmt.Errorf("audio overview button not found")
	}
	if err := audioBtn.(playwright.ElementHandle).Click(); err != nil {
		return nil, fmt.Errorf("click audio overview: %w", err)
	}

	// If custom prompt provided, use the customize dialog
	if customPrompt != "" {
		// Wait for customize dialog
		if _, err := page.WaitForSelector(Dialog, playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(10000),
		}); err != nil {
			return nil, fmt.Errorf("customize dialog did not appear: %w", err)
		}

		// Fill the custom prompt
		promptEl, err := page.QuerySelector("textarea[placeholder*=\"prompt\" i], textarea")
		if err != nil || promptEl == nil {
			return nil, fmt.Errorf("prompt textarea not found")
		}
		if err := promptEl.(playwright.ElementHandle).Fill(customPrompt); err != nil {
			return nil, fmt.Errorf("fill custom prompt: %w", err)
		}

		// Click Generate button in dialog
		genBtn, err := page.QuerySelector("button:has-text(\"Generate\")")
		if err != nil || genBtn == nil {
			return nil, fmt.Errorf("generate button not found in dialog")
		}
		if err := genBtn.(playwright.ElementHandle).Click(); err != nil {
			return nil, fmt.Errorf("click generate in dialog: %w", err)
		}
	}

	// Wait for dialog to close (if it opened)
	if customPrompt != "" {
		if _, err := page.WaitForSelector(Dialog, playwright.PageWaitForSelectorOptions{
			State:   playwright.WaitForSelectorStateHidden,
			Timeout: playwright.Float(30000),
		}); err != nil {
			return nil, fmt.Errorf("dialog did not close: %w", err)
		}
	}

	// Optionally block until ready
	if timeoutMs > 0 {
		deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
		for time.Now().Before(deadline) {
			s, err := GetAudioStatus(page)
			if err == nil && s == AudioReady {
				return &AudioResult{Status: AudioReady, ElapsedMs: time.Since(start).Milliseconds()}, nil
			}
			time.Sleep(2 * time.Second)
		}
	}

	// Return current status (likely in_progress)
	status, _ = GetAudioStatus(page)
	return &AudioResult{Status: status, ElapsedMs: time.Since(start).Milliseconds()}, nil
}

// GetAudioStatus checks the current state of the audio overview.
// Returns AudioReady if a play button tile exists,
// AudioInProgress if a spinner/loading indicator is visible,
// AudioNotStarted otherwise.
func GetAudioStatus(page playwright.Page) (AudioStatus, error) {
	// Check for ready state: audio tile with play button
	elements, err := page.QuerySelectorAll(AudioPlayerTile)
	if err == nil && len(elements) > 0 {
		return AudioReady, nil
	}

	// Check for in-progress: spinner text
	spinnerTexts := []string{"Generating", "generating", "Creating", "creating"}
	for _, text := range spinnerTexts {
		els, err := page.QuerySelectorAll("text=" + text)
		if err == nil && len(els) > 0 {
			return AudioInProgress, nil
		}
	}

	// Also check for common spinner indicators
	spinnerSelectors := []string{
		".spinner",
		"[aria-busy=\"true\"]",
		".loading",
	}
	for _, sel := range spinnerSelectors {
		els, err := page.QuerySelectorAll(sel)
		if err == nil && len(els) > 0 {
			return AudioInProgress, nil
		}
	}

	return AudioNotStarted, nil
}

// DownloadAudio downloads the audio overview file to the specified directory.
// It verifies the audio is ready, clicks the three-dot menu, selects Download,
// and captures the downloaded file.
func DownloadAudio(page playwright.Page, destDir string) (*DownloadResult, error) {
	start := time.Now()

	// Verify audio is ready
	status, err := GetAudioStatus(page)
	if err != nil {
		return nil, fmt.Errorf("check audio status: %w", err)
	}
	if status != AudioReady {
		return nil, fmt.Errorf("audio not ready (status: %s)", status)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("create dest dir: %w", err)
	}

	// Click the audio tile to open it
	tile, err := page.QuerySelector(AudioPlayerTile)
	if err != nil || tile == nil {
		return nil, fmt.Errorf("audio tile not found")
	}
	if err := tile.(playwright.ElementHandle).Click(); err != nil {
		return nil, fmt.Errorf("click audio tile: %w", err)
	}

	// Wait for the audio player to be visible
	if _, err := page.WaitForSelector(".audio-player, [role=\"dialog\"]", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(10000),
	}); err != nil {
		return nil, fmt.Errorf("audio player did not appear: %w", err)
	}

	// Click three-dot menu
	menuBtn, err := page.QuerySelector("button[aria-label*=\"more\" i], .more-vert, [data-icon=\"more_vert\"]")
	if err != nil || menuBtn == nil {
		// Fallback: try any button with "more" text
		menuBtn, err = page.QuerySelector("button:has-text(\"more\")")
		if err != nil || menuBtn == nil {
			return nil, fmt.Errorf("menu button not found")
		}
	}
	if err := menuBtn.(playwright.ElementHandle).Click(); err != nil {
		return nil, fmt.Errorf("click menu: %w", err)
	}

	// Click Download option
	downloadBtn, err := page.QuerySelector("text=Download, button:has-text(\"Download\")")
	if err != nil || downloadBtn == nil {
		return nil, fmt.Errorf("download button not found")
	}

	// Wait for download event
	download, err := page.WaitForEvent("download", playwright.PageWaitForEventOptions{
		Predicate: nil,
		Timeout:   playwright.Float(30000),
	})
	if err != nil {
		// Fallback: click and hope download starts
		if err := downloadBtn.(playwright.ElementHandle).Click(); err != nil {
			return nil, fmt.Errorf("click download: %w", err)
		}
		return nil, fmt.Errorf("download event not captured: %w", err)
	}

	dl := download.(playwright.Download)

	// Save to destination
	destPath := filepath.Join(destDir, "audio-overview.mp3")
	if err := dl.SaveAs(destPath); err != nil {
		return nil, fmt.Errorf("save download: %w", err)
	}

	// Get file size
	info, err := os.Stat(destPath)
	if err != nil {
		return nil, fmt.Errorf("stat downloaded file: %w", err)
	}

	return &DownloadResult{
		FilePath:  destPath,
		FileSize:  info.Size(),
		ElapsedMs: time.Since(start).Milliseconds(),
	}, nil
}
