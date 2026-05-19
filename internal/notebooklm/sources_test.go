package notebooklm

import (
	"testing"
)

func TestIsUUIDRedirect(t *testing.T) {
	tests := []struct {
		name        string
		currentURL  string
		expectedURL string
		want        bool
	}{
		{
			name:        "same UUID — no redirect",
			currentURL:  "https://notebooklm.google.com/notebook/abc12345-def0-1234-5678-abcdef123456",
			expectedURL: "https://notebooklm.google.com/notebook/abc12345-def0-1234-5678-abcdef123456",
			want:        false,
		},
		{
			name:        "different UUID — redirect detected",
			currentURL:  "https://notebooklm.google.com/notebook/11111111-2222-3333-4444-555555555555",
			expectedURL: "https://notebooklm.google.com/notebook/abcdef01-2345-6789-abcd-ef0123456789",
			want:        true,
		},
		{
			name:        "no UUID in current URL",
			currentURL:  "https://notebooklm.google.com/notebook",
			expectedURL: "https://notebooklm.google.com/notebook/abc12345-def0-1234-5678-abcdef123456",
			want:        false,
		},
		{
			name:        "no UUID in expected URL",
			currentURL:  "https://notebooklm.google.com/notebook/abc12345-def0-1234-5678-abcdef123456",
			expectedURL: "https://notebooklm.google.com/notebook",
			want:        false,
		},
		{
			name:        "no UUID in either URL",
			currentURL:  "https://example.com",
			expectedURL: "https://example.com",
			want:        false,
		},
		{
			name:        "case insensitive UUID comparison",
			currentURL:  "https://notebooklm.google.com/notebook/ABC12345-DEF0-1234-5678-ABCDEF123456",
			expectedURL: "https://notebooklm.google.com/notebook/abc12345-def0-1234-5678-abcdef123456",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUUIDRedirect(tt.currentURL, tt.expectedURL)
			if got != tt.want {
				t.Errorf("IsUUIDRedirect(%q, %q) = %v, want %v",
					tt.currentURL, tt.expectedURL, got, tt.want)
			}
		})
	}
}

func TestExtractUUID(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "valid UUID in URL",
			url:  "https://notebooklm.google.com/notebook/abc12345-def0-1234-5678-abcdef123456",
			want: "abc12345-def0-1234-5678-abcdef123456",
		},
		{
			name: "no UUID",
			url:  "https://example.com/page",
			want: "",
		},
		{
			name: "UUID with query params",
			url:  "https://notebooklm.google.com/notebook/11111111-2222-3333-4444-555555555555?foo=bar",
			want: "11111111-2222-3333-4444-555555555555",
		},
		{
			name: "uppercase UUID",
			url:  "https://notebooklm.google.com/notebook/AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE",
			want: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractUUID(tt.url)
			if got != tt.want {
				t.Errorf("extractUUID(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestSourceResult_StructFields(t *testing.T) {
	result := &SourceResult{
		SourceCount: 5,
		SourceType:  "url",
		ElapsedMs:   3000,
	}

	if result.SourceCount != 5 {
		t.Errorf("SourceCount = %d, want 5", result.SourceCount)
	}
	if result.SourceType != "url" {
		t.Errorf("SourceType = %q, want %q", result.SourceType, "url")
	}
	if result.ElapsedMs != 3000 {
		t.Errorf("ElapsedMs = %d, want 3000", result.ElapsedMs)
	}
}

// TestIntegration_AddSourceURL is guarded by testing.Short().
func TestIntegration_AddSourceURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

// TestIntegration_AddSourceText is guarded by testing.Short().
func TestIntegration_AddSourceText(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

// TestIntegration_CountSources is guarded by testing.Short().
func TestIntegration_CountSources(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}
