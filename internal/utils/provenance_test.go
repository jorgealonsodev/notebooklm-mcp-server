package utils

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultProvenance(t *testing.T) {
	p := DefaultProvenance()

	if p.Provider != "Google" {
		t.Errorf("Provider = %q, want %q", p.Provider, "Google")
	}
	if p.Model != "NotebookLM" {
		t.Errorf("Model = %q, want %q", p.Model, "NotebookLM")
	}
	if p.Via != "notebooklm-mcp-server" {
		t.Errorf("Via = %q, want %q", p.Via, "notebooklm-mcp-server")
	}
	if !p.AIGenerated {
		t.Error("AIGenerated should be true")
	}
}

func TestMarkAnswer_AIGenerated(t *testing.T) {
	provenance := ProvenanceInfo{
		Provider:    "Google",
		Model:       "NotebookLM",
		Via:         "notebooklm-mcp-server",
		AIGenerated: true,
	}

	answer := "The answer is 42."
	result := MarkAnswer(answer, provenance)

	if !strings.HasPrefix(result, "[AI-generated") {
		t.Errorf("expected AI marker prefix, got: %s", result)
	}
	if !strings.Contains(result, answer) {
		t.Errorf("expected original answer in result, got: %s", result)
	}
}

func TestMarkAnswer_NotAI(t *testing.T) {
	provenance := ProvenanceInfo{AIGenerated: false}
	answer := "The answer is 42."

	result := MarkAnswer(answer, provenance)
	if result != answer {
		t.Errorf("expected unchanged answer, got: %s", result)
	}
}

func TestMarkAnswer_EmptyAnswer(t *testing.T) {
	provenance := DefaultProvenance()
	result := MarkAnswer("", provenance)

	if !strings.HasPrefix(result, "[AI-generated") {
		t.Errorf("expected AI marker even for empty answer, got: %s", result)
	}
}

func TestDisclaimerEnabled_Default(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("NOTEBOOKLM_DISABLE_DISCLAIMER")

	if !DisclaimerEnabled() {
		t.Error("disclaimer should be enabled by default")
	}
}

func TestDisclaimerEnabled_Disabled(t *testing.T) {
	tests := []string{"true", "1", "yes", "TRUE", "True"}
	for _, v := range tests {
		t.Run(v, func(t *testing.T) {
			os.Setenv("NOTEBOOKLM_DISABLE_DISCLAIMER", v)
			defer os.Unsetenv("NOTEBOOKLM_DISABLE_DISCLAIMER")

			if DisclaimerEnabled() {
				t.Errorf("disclaimer should be disabled when NOTEBOOKLM_DISABLE_DISCLAIMER=%s", v)
			}
		})
	}
}

func TestDisclaimerEnabled_StillEnabled(t *testing.T) {
	tests := []string{"false", "0", "no", "FALSE", ""}
	for _, v := range tests {
		t.Run(v, func(t *testing.T) {
			os.Setenv("NOTEBOOKLM_DISABLE_DISCLAIMER", v)
			defer os.Unsetenv("NOTEBOOKLM_DISABLE_DISCLAIMER")

			if !DisclaimerEnabled() {
				t.Errorf("disclaimer should be enabled when NOTEBOOKLM_DISABLE_DISCLAIMER=%s", v)
			}
		})
	}
}

func TestWithDisclaimer(t *testing.T) {
	os.Unsetenv("NOTEBOOKLM_DISABLE_DISCLAIMER")

	answer := "The answer is 42."
	result := WithDisclaimer(answer)

	if !strings.HasPrefix(result, DefaultDisclaimerPrefix()) {
		t.Errorf("expected disclaimer prefix, got: %s", result)
	}
	if !strings.HasSuffix(result, answer) {
		t.Errorf("expected answer at end, got: %s", result)
	}
}

func TestWithDisclaimer_Disabled(t *testing.T) {
	os.Setenv("NOTEBOOKLM_DISABLE_DISCLAIMER", "true")
	defer os.Unsetenv("NOTEBOOKLM_DISABLE_DISCLAIMER")

	answer := "The answer is 42."
	result := WithDisclaimer(answer)

	if result != answer {
		t.Errorf("expected unchanged answer when disclaimer disabled, got: %s", result)
	}
}

func TestDefaultDisclaimerPrefix(t *testing.T) {
	prefix := DefaultDisclaimerPrefix()
	if !strings.Contains(prefix, "AI-GENERATED") {
		t.Errorf("expected AI-GENERATED in prefix: %s", prefix)
	}
	if !strings.Contains(prefix, "NotebookLM") {
		t.Errorf("expected NotebookLM in prefix: %s", prefix)
	}
}
