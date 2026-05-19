package utils

import (
	"fmt"
	"os"
	"strings"
)

// ProvenanceInfo carries metadata about how an AI-generated answer was produced.
type ProvenanceInfo struct {
	Provider     string
	Model        string
	Via          string
	Grounding    string
	AIGenerated  bool
}

// DefaultProvenance returns a ProvenanceInfo with default values for the
// NotebookLM MCP server.
func DefaultProvenance() ProvenanceInfo {
	return ProvenanceInfo{
		Provider:    "Google",
		Model:       "NotebookLM",
		Via:         "notebooklm-mcp-server",
		Grounding:   "notebook-sources",
		AIGenerated: true,
	}
}

// MarkAnswer prepends an AI provenance marker to the given answer.
// If provenance.AIGenerated is false, the answer is returned unchanged.
func MarkAnswer(answer string, provenance ProvenanceInfo) string {
	if !provenance.AIGenerated {
		return answer
	}

	marker := fmt.Sprintf("[AI-generated via %s (%s/%s)]\n\n",
		provenance.Via, provenance.Provider, provenance.Model)
	return marker + answer
}

// DisclaimerEnabled returns true if the AI disclaimer should be included
// in responses. It checks the NOTEBOOKLM_DISABLE_DISCLAIMER env var.
func DisclaimerEnabled() bool {
	v := strings.ToLower(os.Getenv("NOTEBOOKLM_DISABLE_DISCLAIMER"))
	return v != "true" && v != "1" && v != "yes"
}

// DefaultDisclaimerPrefix returns the standard disclaimer prefix.
func DefaultDisclaimerPrefix() string {
	return "[AI-GENERATED via Google NotebookLM — verify before relying]"
}

// WithDisclaimer prepends the disclaimer to the answer if disclaimers are enabled.
func WithDisclaimer(answer string) string {
	if !DisclaimerEnabled() {
		return answer
	}
	return DefaultDisclaimerPrefix() + " " + answer
}
