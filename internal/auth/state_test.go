package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveState(t *testing.T) {
	dir := t.TempDir()
	state := &BrowserState{
		Cookies: []CookieState{
			{Name: "SID", Value: "test-sid", Domain: ".google.com", Path: "/", Expires: -1},
		},
		Origins: []OriginState{
			{Origin: "https://notebooklm.google.com", LocalStorage: [][2]string{{"key1", "val1"}}},
		},
	}

	err := SaveState(dir, state)
	if err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	// Verify file exists and content is correct
	data, err := os.ReadFile(filepath.Join(dir, stateFile))
	if err != nil {
		t.Fatalf("state.json not found: %v", err)
	}

	var loaded BrowserState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(loaded.Cookies) != 1 || loaded.Cookies[0].Name != "SID" {
		t.Errorf("expected SID cookie, got %+v", loaded.Cookies)
	}
}

func TestLoadState(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid state",
			content: `{"cookies":[{"name":"SID","value":"x","domain":".google.com","path":"/","expires":-1}],"origins":[]}`,
			wantErr: false,
		},
		{
			name:    "empty file",
			content: "",
			wantErr: true,
		},
		{
			name:    "invalid json",
			content: `{not valid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			// Always write the file (even for empty content)
			if err := os.WriteFile(filepath.Join(dir, stateFile), []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			state, err := LoadState(dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && state == nil {
				t.Error("LoadState() returned nil state without error")
			}
		})
	}
}

func TestLoadStateMissingFile(t *testing.T) {
	dir := t.TempDir()
	state, err := LoadState(dir)
	if err != nil {
		t.Fatalf("LoadState() on missing file should return nil, err=nil; got err=%v", err)
	}
	if state != nil {
		t.Error("LoadState() on missing file should return nil state")
	}
}

func TestValidateCookies(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		cookies []CookieState
		fileAge time.Duration
		wantErr bool
	}{
		{
			name:    "all 9 google cookies present and valid",
			cookies: makeGoogleCookies(now.Add(1*time.Hour)),
			fileAge: 1 * time.Hour,
			wantErr: false,
		},
		{
			name:    "missing SID cookie",
			cookies: makeGoogleCookies(now.Add(1*time.Hour))[1:], // remove first
			fileAge: 1 * time.Hour,
			wantErr: true,
		},
		{
			name:    "expired cookie",
			cookies: makeGoogleCookies(now.Add(-1*time.Hour)), // all expired
			fileAge: 1 * time.Hour,
			wantErr: true,
		},
		{
			name:    "empty cookies",
			cookies: []CookieState{},
			fileAge: 1 * time.Hour,
			wantErr: true,
		},
		{
			name:    "state file older than 24h",
			cookies: makeGoogleCookies(now.Add(1*time.Hour)),
			fileAge: 25 * time.Hour,
			wantErr: true,
		},
		{
			name:    "state file exactly at 24h boundary",
			cookies: makeGoogleCookies(now.Add(1*time.Hour)),
			fileAge: 24*time.Hour + 1*time.Second,
			wantErr: true,
		},
		{
			name:    "some google cookies missing",
			cookies: []CookieState{makeGoogleCookies(now.Add(1*time.Hour))[0], makeGoogleCookies(now.Add(1*time.Hour))[1]},
			fileAge: 1 * time.Hour,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			state := &BrowserState{
				Cookies: tt.cookies,
				Origins: []OriginState{},
			}
			if err := SaveState(dir, state); err != nil {
				t.Fatal(err)
			}

			// Manipulate file modification time to simulate file age
			modTime := now.Add(-tt.fileAge)
			if err := os.Chtimes(filepath.Join(dir, stateFile), modTime, modTime); err != nil {
				t.Fatal(err)
			}

			err := ValidateCookies(dir, now)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCookies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func makeGoogleCookies(expires time.Time) []CookieState {
	names := []string{"SID", "HSID", "SSID", "APISID", "SAPISID", "OSID", "__Secure-OSID", "__Secure-1PSID", "__Secure-3PSID"}
	cookies := make([]CookieState, len(names))
	for i, name := range names {
		var exp float64
		if !expires.IsZero() {
			exp = float64(expires.Unix())
		}
		cookies[i] = CookieState{
			Name:    name,
			Value:   "value-" + name,
			Domain:  ".google.com",
			Path:    "/",
			Expires: exp,
		}
	}
	return cookies
}
