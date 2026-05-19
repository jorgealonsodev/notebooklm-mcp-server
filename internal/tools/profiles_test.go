package tools

import (
	"testing"
)

func TestResolveProfile(t *testing.T) {
	tests := []struct {
		name     string
		profile  ToolProfile
		wantLen  int
		wantName string // a tool name that should be in the profile
	}{
		{
			name:     "minimal has 4 tools",
			profile:  ProfileMinimal,
			wantLen:  4,
			wantName: "ask_question",
		},
		{
			name:     "standard has 16 tools",
			profile:  ProfileStandard,
			wantLen:  16,
			wantName: "list_sessions",
		},
		{
			name:     "full has 20 tools",
			profile:  ProfileFull,
			wantLen:  20,
			wantName: "cleanup_data",
		},
		{
			name:     "unknown profile returns full",
			profile:  ToolProfile("unknown"),
			wantLen:  20,
			wantName: "add_notebook",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := ResolveProfile(tt.profile)
			if len(names) != tt.wantLen {
				t.Errorf("ResolveProfile(%q) returned %d tools, want %d", tt.profile, len(names), tt.wantLen)
			}

			found := false
			for _, n := range names {
				if n == tt.wantName {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("ResolveProfile(%q) missing tool %q", tt.profile, tt.wantName)
			}
		})
	}
}

func TestToolProfiles_MinimalSubsetOfStandard(t *testing.T) {
	minimal := ResolveProfile(ProfileMinimal)
	standard := ResolveProfile(ProfileStandard)

	minimalSet := make(map[string]bool)
	for _, n := range minimal {
		minimalSet[n] = true
	}

	for _, n := range minimal {
		found := false
		for _, s := range standard {
			if s == n {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("minimal tool %q not in standard profile", n)
		}
	}
}

func TestToolProfiles_StandardSubsetOfFull(t *testing.T) {
	standard := ResolveProfile(ProfileStandard)
	full := ResolveProfile(ProfileFull)

	for _, n := range standard {
		found := false
		for _, f := range full {
			if f == n {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("standard tool %q not in full profile", n)
		}
	}
}
