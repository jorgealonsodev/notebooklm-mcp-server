// Package config provides environment-driven configuration for the
// notebooklm-mcp-server. Priority: defaults → env vars → per-call BrowserOptions.
package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// ProfileStrategy controls how Chrome profiles are managed across instances.
type ProfileStrategy string

const (
	ProfileStrategyAuto     ProfileStrategy = "auto"
	ProfileStrategyIsolated ProfileStrategy = "isolated"
	ProfileStrategySingle   ProfileStrategy = "single"
)

// Viewport holds browser window dimensions.
type Viewport struct {
	Width  int
	Height int
}

// Config is the resolved, immutable configuration for a single server run (or
// a single tool invocation after ApplyBrowserOptions).
type Config struct {
	// NotebookLM
	NotebookURL string

	// Browser
	Headless         bool
	BrowserTimeoutMs int
	AnswerTimeoutMs  int
	Viewport         Viewport

	// Session
	MaxSessions           int
	SessionTimeoutSeconds int

	// Auth
	AutoLoginEnabled    bool
	LoginEmail          string
	LoginPassword       string
	AutoLoginTimeoutMs  int
	SetupTimeoutMs      int

	// Stealth
	StealthEnabled       bool
	StealthRandomDelays  bool
	StealthHumanTyping   bool
	StealthMouseMovements bool
	TypingWPMMin         int
	TypingWPMMax         int
	MinDelayMs           int
	MaxDelayMs           int

	// Paths
	DataDir             string
	BrowserStateDir     string
	ChromeProfileDir    string
	ChromeInstancesDir  string

	// Library defaults
	NotebookDescription  string
	NotebookTopics       []string
	NotebookContentTypes []string
	NotebookUseCases     []string

	// Profile management
	ProfileStrategy              ProfileStrategy
	CloneProfileOnIsolated       bool
	CleanupInstancesOnStartup    bool
	CleanupInstancesOnShutdown   bool
	InstanceProfileTTLHours      int
	InstanceProfileMaxCount      int
}

// Load returns a Config populated from defaults, then overridden by environment
// variables. It is safe to call repeatedly (each call re-reads the environment).
func Load() Config {
	data := platformDataDir()
	cfg := Config{
		NotebookURL:      "",
		Headless:         true,
		BrowserTimeoutMs: 30_000,
		AnswerTimeoutMs:  600_000,
		Viewport:         Viewport{Width: 1920, Height: 1080},

		MaxSessions:           10,
		SessionTimeoutSeconds: 900,

		AutoLoginEnabled:   false,
		LoginEmail:         "",
		LoginPassword:      "",
		AutoLoginTimeoutMs: 120_000,
		SetupTimeoutMs:     600_000, // 10 minutes

		StealthEnabled:        true,
		StealthRandomDelays:   true,
		StealthHumanTyping:    true,
		StealthMouseMovements: true,
		TypingWPMMin:          160,
		TypingWPMMax:          240,
		MinDelayMs:            100,
		MaxDelayMs:            400,

		DataDir:            data,
		BrowserStateDir:    filepath.Join(data, "browser_state"),
		ChromeProfileDir:   filepath.Join(data, "chrome_profile"),
		ChromeInstancesDir: filepath.Join(data, "chrome_profile_instances"),

		NotebookDescription:  "General knowledge base",
		NotebookTopics:       []string{"General topics"},
		NotebookContentTypes: []string{"documentation", "examples"},
		NotebookUseCases:     []string{"General research"},

		ProfileStrategy:            ProfileStrategyAuto,
		CloneProfileOnIsolated:     false,
		CleanupInstancesOnStartup:  true,
		CleanupInstancesOnShutdown: true,
		InstanceProfileTTLHours:    72,
		InstanceProfileMaxCount:    20,
	}

	applyEnvOverrides(&cfg)
	return cfg
}

// platformDataDir returns the platform-appropriate data directory for the app,
// without the "-nodejs" suffix that some helpers add.
//
//   - Linux:   ~/.local/share/notebooklm-mcp
//   - macOS:   ~/Library/Application Support/notebooklm-mcp
//   - Windows: %APPDATA%\notebooklm-mcp
func platformDataDir() string {
	const app = "notebooklm-mcp"
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", app)
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			appdata = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appdata, app)
	default: // linux + others
		xdg := os.Getenv("XDG_DATA_HOME")
		if xdg == "" {
			xdg = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(xdg, app)
	}
}

func applyEnvOverrides(c *Config) {
	if v := os.Getenv("NOTEBOOK_URL"); v != "" {
		c.NotebookURL = v
	}
	c.Headless = envBool("HEADLESS", c.Headless)
	c.BrowserTimeoutMs = envInt("BROWSER_TIMEOUT", c.BrowserTimeoutMs)
	c.AnswerTimeoutMs = envInt("ANSWER_TIMEOUT_MS", c.AnswerTimeoutMs)
	c.MaxSessions = envInt("MAX_SESSIONS", c.MaxSessions)
	c.SessionTimeoutSeconds = envInt("SESSION_TIMEOUT", c.SessionTimeoutSeconds)
	c.AutoLoginEnabled = envBool("AUTO_LOGIN_ENABLED", c.AutoLoginEnabled)
	if v := os.Getenv("LOGIN_EMAIL"); v != "" {
		c.LoginEmail = v
	}
	if v := os.Getenv("LOGIN_PASSWORD"); v != "" {
		c.LoginPassword = v
	}
	c.AutoLoginTimeoutMs = envInt("AUTO_LOGIN_TIMEOUT_MS", c.AutoLoginTimeoutMs)
	c.SetupTimeoutMs = envInt("SETUP_TIMEOUT_MS", c.SetupTimeoutMs)
	c.StealthEnabled = envBool("STEALTH_ENABLED", c.StealthEnabled)
	c.StealthRandomDelays = envBool("STEALTH_RANDOM_DELAYS", c.StealthRandomDelays)
	c.StealthHumanTyping = envBool("STEALTH_HUMAN_TYPING", c.StealthHumanTyping)
	c.StealthMouseMovements = envBool("STEALTH_MOUSE_MOVEMENTS", c.StealthMouseMovements)
	c.TypingWPMMin = envInt("TYPING_WPM_MIN", c.TypingWPMMin)
	c.TypingWPMMax = envInt("TYPING_WPM_MAX", c.TypingWPMMax)
	c.MinDelayMs = envInt("MIN_DELAY_MS", c.MinDelayMs)
	c.MaxDelayMs = envInt("MAX_DELAY_MS", c.MaxDelayMs)
	if v := os.Getenv("NOTEBOOK_DESCRIPTION"); v != "" {
		c.NotebookDescription = v
	}
	c.NotebookTopics = envSlice("NOTEBOOK_TOPICS", c.NotebookTopics)
	c.NotebookContentTypes = envSlice("NOTEBOOK_CONTENT_TYPES", c.NotebookContentTypes)
	c.NotebookUseCases = envSlice("NOTEBOOK_USE_CASES", c.NotebookUseCases)
	c.ProfileStrategy = envProfileStrategy("NOTEBOOK_PROFILE_STRATEGY", c.ProfileStrategy)
	c.CloneProfileOnIsolated = envBool("NOTEBOOK_CLONE_PROFILE", c.CloneProfileOnIsolated)
	c.CleanupInstancesOnStartup = envBool("NOTEBOOK_CLEANUP_ON_STARTUP", c.CleanupInstancesOnStartup)
	c.CleanupInstancesOnShutdown = envBool("NOTEBOOK_CLEANUP_ON_SHUTDOWN", c.CleanupInstancesOnShutdown)
	c.InstanceProfileTTLHours = envInt("NOTEBOOK_INSTANCE_TTL_HOURS", c.InstanceProfileTTLHours)
	c.InstanceProfileMaxCount = envInt("NOTEBOOK_INSTANCE_MAX_COUNT", c.InstanceProfileMaxCount)
}

// ---- BrowserOptions (per-call overrides) ----

// ViewportOptions holds optional viewport dimensions for per-call overrides.
type ViewportOptions struct {
	Width  *int
	Height *int
}

// BrowserOptions are optional per-call overrides passed through MCP tool args.
type BrowserOptions struct {
	Show     *bool
	Headless *bool
	TimeoutMs *int
	Stealth  *StealthOptions
	Viewport *ViewportOptions
}

// StealthOptions mirrors the stealth sub-object in the TypeScript BrowserOptions.
type StealthOptions struct {
	Enabled        *bool
	RandomDelays   *bool
	HumanTyping    *bool
	MouseMovements *bool
	TypingWPMMin   *int
	TypingWPMMax   *int
	DelayMinMs     *int
	DelayMaxMs     *int
}

// ApplyBrowserOptions returns a copy of c with per-call BrowserOptions applied.
// It never mutates the receiver.
func (c Config) ApplyBrowserOptions(opts *BrowserOptions) Config {
	out := c // shallow copy (slices are read-only after Load, so this is safe)
	if opts == nil {
		return out
	}
	if opts.Show != nil {
		out.Headless = !*opts.Show
	}
	if opts.Headless != nil {
		out.Headless = *opts.Headless
	}
	if opts.TimeoutMs != nil {
		out.BrowserTimeoutMs = *opts.TimeoutMs
	}
	if opts.Viewport != nil {
		vp := out.Viewport
		if opts.Viewport.Width != nil {
			vp.Width = *opts.Viewport.Width
		}
		if opts.Viewport.Height != nil {
			vp.Height = *opts.Viewport.Height
		}
		out.Viewport = vp
	}
	if opts.Stealth != nil {
		s := opts.Stealth
		if s.Enabled != nil {
			out.StealthEnabled = *s.Enabled
		}
		if s.RandomDelays != nil {
			out.StealthRandomDelays = *s.RandomDelays
		}
		if s.HumanTyping != nil {
			out.StealthHumanTyping = *s.HumanTyping
		}
		if s.MouseMovements != nil {
			out.StealthMouseMovements = *s.MouseMovements
		}
		if s.TypingWPMMin != nil {
			out.TypingWPMMin = *s.TypingWPMMin
		}
		if s.TypingWPMMax != nil {
			out.TypingWPMMax = *s.TypingWPMMax
		}
		if s.DelayMinMs != nil {
			out.MinDelayMs = *s.DelayMinMs
		}
		if s.DelayMaxMs != nil {
			out.MaxDelayMs = *s.DelayMaxMs
		}
	}
	return out
}

// ---- helpers ----

func envBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	return v == "true" || v == "1"
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envSlice(key string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func envProfileStrategy(key string, fallback ProfileStrategy) ProfileStrategy {
	v := os.Getenv(key)
	switch ProfileStrategy(v) {
	case ProfileStrategyAuto, ProfileStrategyIsolated, ProfileStrategySingle:
		return ProfileStrategy(v)
	}
	return fallback
}
