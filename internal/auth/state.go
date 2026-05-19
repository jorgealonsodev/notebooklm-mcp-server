// Package auth provides cookie-based Google authentication for NotebookLM,
// including state persistence, validation, and account isolation.
package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jorge/notebooklm-mcp-server/internal/apperrors"
)

const (
	stateFile             = "state.json"
	sessionFile           = "session.json"
	maxStateAge           = 24 * time.Hour
	googleCookieSID       = "SID"
	googleCookieHSID      = "HSID"
	googleCookieSSID      = "SSID"
	googleCookieAPISID    = "APISID"
	googleCookieSAPISID   = "SAPISID"
	googleCookieOSID      = "OSID"
	googleCookieSecureOSID   = "__Secure-OSID"
	googleCookieSecure1PSID  = "__Secure-1PSID"
	googleCookieSecure3PSID  = "__Secure-3PSID"
)

// requiredGoogleCookies lists the 9 Google authentication cookies that must
// be present and non-expired for a valid session.
var requiredGoogleCookies = []string{
	googleCookieSID,
	googleCookieHSID,
	googleCookieSSID,
	googleCookieAPISID,
	googleCookieSAPISID,
	googleCookieOSID,
	googleCookieSecureOSID,
	googleCookieSecure1PSID,
	googleCookieSecure3PSID,
}

// CookieState represents a single browser cookie as persisted by Playwright.
type CookieState struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
}

// OriginState represents a storage origin with localStorage entries.
type OriginState struct {
	Origin       string     `json:"origin"`
	LocalStorage [][2]string `json:"localStorage"`
}

// BrowserState holds the full browser storage state (cookies + localStorage).
type BrowserState struct {
	Cookies []CookieState `json:"cookies"`
	Origins []OriginState `json:"origins"`
}

// SaveState writes the browser state to state.json in the given directory.
func SaveState(dir string, state *BrowserState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	path := filepath.Join(dir, stateFile)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}
	return nil
}

// LoadState reads the browser state from state.json in the given directory.
// Returns nil, nil if the file does not exist.
func LoadState(dir string) (*BrowserState, error) {
	path := filepath.Join(dir, stateFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read state file: %w", err)
	}

	var state BrowserState
	if len(data) == 0 {
		return nil, fmt.Errorf("empty state file")
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse state file: %w", err)
	}
	return &state, nil
}

// ValidateCookies checks that all required Google cookies are present and
// non-expired, and that the state file is not older than 24 hours.
func ValidateCookies(dir string, now time.Time) error {
	state, err := LoadState(dir)
	if err != nil {
		return err
	}
	if state == nil {
		return apperrors.NewAuthenticationError("no authentication state found", false)
	}

	// Check file age
	info, err := os.Stat(filepath.Join(dir, stateFile))
	if err != nil {
		return fmt.Errorf("stat state file: %w", err)
	}
	if now.Sub(info.ModTime()) > maxStateAge {
		return apperrors.NewAuthenticationError("authentication state expired (older than 24h)", true)
	}

	// Build a set of cookie names for quick lookup
	cookieNames := make(map[string]*CookieState, len(state.Cookies))
	for i := range state.Cookies {
		c := &state.Cookies[i]
		cookieNames[c.Name] = c
	}

	// Check each required cookie
	for _, name := range requiredGoogleCookies {
		c, ok := cookieNames[name]
		if !ok {
			return apperrors.NewAuthenticationError(fmt.Sprintf("missing required cookie: %s", name), false)
		}
		// Check expiry: expires == -1 means session cookie (no expiry)
		if c.Expires > 0 && c.Expires < float64(now.Unix()) {
			return apperrors.NewAuthenticationError(fmt.Sprintf("cookie expired: %s", name), false)
		}
	}

	return nil
}
