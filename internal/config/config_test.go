package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jorge/notebooklm-mcp-server/internal/config"
)

// ---- defaults ----

func TestDefaults_Headless(t *testing.T) {
	cfg := config.Load()
	if !cfg.Headless {
		t.Error("default Headless should be true")
	}
}

func TestDefaults_Viewport(t *testing.T) {
	cfg := config.Load()
	if cfg.Viewport.Width != 1920 || cfg.Viewport.Height != 1080 {
		t.Errorf("default viewport should be 1920x1080, got %dx%d", cfg.Viewport.Width, cfg.Viewport.Height)
	}
}

func TestDefaults_BrowserTimeout(t *testing.T) {
	cfg := config.Load()
	if cfg.BrowserTimeoutMs != 30_000 {
		t.Errorf("default BrowserTimeoutMs should be 30000, got %d", cfg.BrowserTimeoutMs)
	}
}

func TestDefaults_AnswerTimeout(t *testing.T) {
	cfg := config.Load()
	if cfg.AnswerTimeoutMs != 600_000 {
		t.Errorf("default AnswerTimeoutMs should be 600000, got %d", cfg.AnswerTimeoutMs)
	}
}

func TestDefaults_MaxSessions(t *testing.T) {
	cfg := config.Load()
	if cfg.MaxSessions != 10 {
		t.Errorf("default MaxSessions should be 10, got %d", cfg.MaxSessions)
	}
}

func TestDefaults_SessionTimeoutSeconds(t *testing.T) {
	cfg := config.Load()
	if cfg.SessionTimeoutSeconds != 900 {
		t.Errorf("default SessionTimeoutSeconds should be 900, got %d", cfg.SessionTimeoutSeconds)
	}
}

func TestDefaults_StealthEnabled(t *testing.T) {
	cfg := config.Load()
	if !cfg.StealthEnabled {
		t.Error("default StealthEnabled should be true")
	}
}

func TestDefaults_TypingWpm(t *testing.T) {
	cfg := config.Load()
	if cfg.TypingWPMMin != 160 || cfg.TypingWPMMax != 240 {
		t.Errorf("default typing WPM should be 160-240, got %d-%d", cfg.TypingWPMMin, cfg.TypingWPMMax)
	}
}

func TestDefaults_ProfileStrategy(t *testing.T) {
	cfg := config.Load()
	if cfg.ProfileStrategy != config.ProfileStrategyAuto {
		t.Errorf("default ProfileStrategy should be auto, got %v", cfg.ProfileStrategy)
	}
}

// ---- platform paths ----

func TestPaths_DataDir_ContainsAppName(t *testing.T) {
	cfg := config.Load()
	if !strings.Contains(cfg.DataDir, "notebooklm-mcp") {
		t.Errorf("DataDir %q should contain 'notebooklm-mcp'", cfg.DataDir)
	}
}

func TestPaths_ChromeProfileDir_UnderDataDir(t *testing.T) {
	cfg := config.Load()
	rel, err := filepath.Rel(cfg.DataDir, cfg.ChromeProfileDir)
	if err != nil || strings.HasPrefix(rel, "..") {
		t.Errorf("ChromeProfileDir %q should be under DataDir %q", cfg.ChromeProfileDir, cfg.DataDir)
	}
}

func TestPaths_NoNodejsSuffix(t *testing.T) {
	cfg := config.Load()
	if strings.Contains(cfg.DataDir, "-nodejs") {
		t.Errorf("DataDir %q must not contain '-nodejs' suffix", cfg.DataDir)
	}
}

func TestPaths_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only test")
	}
	cfg := config.Load()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".local", "share", "notebooklm-mcp")
	if cfg.DataDir != want {
		t.Errorf("Linux DataDir should be %q, got %q", want, cfg.DataDir)
	}
}

// ---- env overrides ----

func TestEnvOverride_Headless(t *testing.T) {
	t.Setenv("HEADLESS", "false")
	cfg := config.Load()
	if cfg.Headless {
		t.Error("HEADLESS=false should set Headless to false")
	}
}

func TestEnvOverride_AnswerTimeout(t *testing.T) {
	t.Setenv("ANSWER_TIMEOUT_MS", "120000")
	cfg := config.Load()
	if cfg.AnswerTimeoutMs != 120_000 {
		t.Errorf("ANSWER_TIMEOUT_MS=120000 should override, got %d", cfg.AnswerTimeoutMs)
	}
}

func TestEnvOverride_ProfileStrategy_Valid(t *testing.T) {
	t.Setenv("NOTEBOOK_PROFILE_STRATEGY", "isolated")
	cfg := config.Load()
	if cfg.ProfileStrategy != config.ProfileStrategyIsolated {
		t.Errorf("expected isolated, got %v", cfg.ProfileStrategy)
	}
}

func TestEnvOverride_ProfileStrategy_Invalid(t *testing.T) {
	t.Setenv("NOTEBOOK_PROFILE_STRATEGY", "bogus")
	cfg := config.Load()
	if cfg.ProfileStrategy != config.ProfileStrategyAuto {
		t.Errorf("invalid strategy should fall back to auto, got %v", cfg.ProfileStrategy)
	}
}

func TestEnvOverride_NotebookTopics(t *testing.T) {
	t.Setenv("NOTEBOOK_TOPICS", "go,testing,mcp")
	cfg := config.Load()
	if len(cfg.NotebookTopics) != 3 || cfg.NotebookTopics[1] != "testing" {
		t.Errorf("NOTEBOOK_TOPICS not parsed correctly: %v", cfg.NotebookTopics)
	}
}

func TestEnvOverride_LoginEmail(t *testing.T) {
	t.Setenv("LOGIN_EMAIL", "test@example.com")
	cfg := config.Load()
	if cfg.LoginEmail != "test@example.com" {
		t.Errorf("LOGIN_EMAIL override failed, got %q", cfg.LoginEmail)
	}
}

// ---- BrowserOptions override ----

func TestApplyBrowserOptions_Show(t *testing.T) {
	base := config.Load()
	base.Headless = true
	show := true
	result := base.ApplyBrowserOptions(&config.BrowserOptions{Show: &show})
	if result.Headless {
		t.Error("Show=true should set Headless=false")
	}
}

func TestApplyBrowserOptions_Headless(t *testing.T) {
	base := config.Load()
	h := false
	result := base.ApplyBrowserOptions(&config.BrowserOptions{Headless: &h})
	if result.Headless {
		t.Error("Headless=false in options should set Headless=false")
	}
}

func TestApplyBrowserOptions_Nil(t *testing.T) {
	base := config.Load()
	result := base.ApplyBrowserOptions(nil)
	if result.Headless != base.Headless {
		t.Error("nil options should return unchanged config")
	}
}

func TestApplyBrowserOptions_Viewport(t *testing.T) {
	base := config.Load()
	w, h := 1280, 720
	result := base.ApplyBrowserOptions(&config.BrowserOptions{
		Viewport: &config.ViewportOptions{Width: &w, Height: &h},
	})
	if result.Viewport.Width != 1280 || result.Viewport.Height != 720 {
		t.Errorf("viewport not applied, got %dx%d", result.Viewport.Width, result.Viewport.Height)
	}
}

func TestApplyBrowserOptions_DoesNotMutateBase(t *testing.T) {
	base := config.Load()
	base.Headless = true
	show := true
	_ = base.ApplyBrowserOptions(&config.BrowserOptions{Show: &show})
	if !base.Headless {
		t.Error("ApplyBrowserOptions must not mutate the receiver")
	}
}
