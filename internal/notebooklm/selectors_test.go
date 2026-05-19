package notebooklm

import (
	"strings"
	"testing"
)

func TestSelectorConstants(t *testing.T) {
	// Verify all selector constants are non-empty and look like valid CSS selectors.
	selectors := map[string]string{
		"ChatInput":         ChatInput,
		"AnswerContainer":   AnswerContainer,
		"LatestAnswer":      LatestAnswer,
		"AddSourceButton":   AddSourceButton,
		"SourceRow":         SourceRow,
		"SourceCountHeader": SourceCountHeader,
		"Dialog":            Dialog,
		"AudioOverviewButton": AudioOverviewButton,
		"AudioPlayerTile":   AudioPlayerTile,
		"CitationMarker":    CitationMarker,
		"HighlightedText":   HighlightedText,
	}

	for name, sel := range selectors {
		t.Run(name, func(t *testing.T) {
			if sel == "" {
				t.Errorf("selector %s is empty", name)
			}
			// Should start with a valid CSS selector prefix
			if !strings.HasPrefix(sel, ".") &&
				!strings.HasPrefix(sel, "[") &&
				!strings.HasPrefix(sel, "textarea") &&
				!strings.HasPrefix(sel, "button") &&
				!strings.HasPrefix(sel, "mat-icon") {
				t.Errorf("selector %s=%q does not look like a valid CSS selector", name, sel)
			}
		})
	}
}

func TestChatInputAriaLabels(t *testing.T) {
	if len(ChatInputAriaLabels) < 10 {
		t.Errorf("expected at least 10 aria-label fallbacks, got %d", len(ChatInputAriaLabels))
	}
	for i, label := range ChatInputAriaLabels {
		if label == "" {
			t.Errorf("ChatInputAriaLabels[%d] is empty", i)
		}
	}
}

func TestIsPlaceholder(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		// English placeholders
		{"thinking", "Thinking", true},
		{"thinking dots", "Thinking...", true},
		{"generating", "Generating", true},
		{"generating response", "Generating response", true},
		{"processing", "Processing", true},
		{"loading", "Loading", true},
		{"analyzing sources", "Analyzing sources", true},
		{"composing answer", "Composing answer", true},
		{"working on it", "Working on it", true},
		{"just a moment", "Just a moment", true},
		// Spanish placeholders
		{"pensando", "Pensando", true},
		{"generando respuesta", "Generando respuesta", true},
		{"cargando", "Cargando", true},
		{"un momento", "Un momento", true},
		// French placeholders
		{"réflexion", "Réflexion", true},
		{"génération de la réponse", "Génération de la réponse", true},
		// German placeholders
		{"denken", "Denken", true},
		{"antwort generieren", "Antwort generieren", true},
		// Portuguese placeholders
		{"gerando resposta", "Gerando resposta", true},
		// Japanese placeholders
		{"考え中", "考え中", true},
		{"回答を生成中", "回答を生成中", true},
		// Chinese placeholders
		{"思考中", "思考中", true},
		{"正在生成回答", "正在生成回答", true},
		// Non-placeholders (real answers)
		{"real answer", "NotebookLM is a tool by Google", false},
		{"short answer", "Yes, that's correct", false},
		{"empty", "", false},
		{"partial match", "I was Thinking about this", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPlaceholder(tt.text)
			if got != tt.want {
				t.Errorf("IsPlaceholder(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestIsPlaceholder_AllPhrases(t *testing.T) {
	// Verify every phrase in the list is recognized as a placeholder.
	for _, phrase := range answerPlaceholderPhrases {
		t.Run(phrase, func(t *testing.T) {
			if !IsPlaceholder(phrase) {
				t.Errorf("phrase %q should be recognized as a placeholder", phrase)
			}
		})
	}
}

func TestIsPlaceholder_WhitespaceVariants(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"leading space", " Thinking", true},
		{"trailing space", "Thinking ", true},
		{"extra spaces", "  Thinking  ", true},
		{"tabs", "\tThinking\t", true},
		{"newlines", "\nThinking\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPlaceholder(tt.text)
			if got != tt.want {
				t.Errorf("IsPlaceholder(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestIsRateLimitMessage(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"rate limit", "You have reached the rate limit", true},
		{"too many requests", "Too many requests, please try again", true},
		{"daily limit", "Daily limit reached", true},
		{"quota exceeded", "Quota exceeded for this account", true},
		{"come back tomorrow", "Please come back tomorrow", true},
		{"you've reached", "You've reached your limit for today", true},
		{"50 queries", "50 queries per day limit reached", true},
		{"slow down", "Please slow down", true},
		{"try again tomorrow", "Please try again tomorrow", true},
		{"temporarily unavailable", "Service temporarily unavailable", true},
		// Non-rate-limit messages
		{"normal error", "An unexpected error occurred", false},
		{"network error", "Network connection lost", false},
		{"empty", "", false},
		{"real answer", "The answer to your question is...", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRateLimitMessage(tt.text)
			if got != tt.want {
				t.Errorf("IsRateLimitMessage(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestIsRateLimitMessage_AllPatterns(t *testing.T) {
	// Verify every pattern is recognized.
	for _, pattern := range rateLimitPatterns {
		t.Run(pattern, func(t *testing.T) {
			if !IsRateLimitMessage(pattern) {
				t.Errorf("pattern %q should be recognized as a rate limit message", pattern)
			}
			// Also test with surrounding context
			if !IsRateLimitMessage("Error: " + pattern + ". Please retry.") {
				t.Errorf("pattern %q with context should be recognized", pattern)
			}
		})
	}
}

func TestSanitizeAnswer(t *testing.T) {
	tests := []struct {
		name string
		input string
		want  string
	}{
		{
			"no icons",
			"NotebookLM is a research tool.",
			"NotebookLM is a research tool.",
		},
		{
			"single icon at end",
			"NotebookLM is a research tool. content_copy",
			"NotebookLM is a research tool.",
		},
		{
			"multiple icons",
			"Here is the answer. content_copy thumb_up share",
			"Here is the answer.",
		},
		{
			"icon in middle",
			"Answer content_copy more text",
			"Answer more text",
		},
		{
			"only icons",
			"content_copy thumb_up",
			"",
		},
		{
			"empty",
			"",
			"",
		},
		{
			"play_arrow icon",
			"Listen to the audio play_arrow now",
			"Listen to the audio now",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeAnswer(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeAnswer(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStripWhitespace(t *testing.T) {
	tests := []struct {
		name string
		input string
		want  string
	}{
		{"no change", "hello", "hello"},
		{"leading", "  hello", "hello"},
		{"trailing", "hello  ", "hello"},
		{"both", "  hello  ", "hello"},
		{"internal collapse", "hello   world", "hello world"},
		{"tabs", "\thello\tworld\t", "hello world"},
		{"newlines", "\nhello\nworld\n", "hello world"},
		{"empty", "", ""},
		{"only spaces", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripWhitespace(tt.input)
			if got != tt.want {
				t.Errorf("stripWhitespace(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name string
		s    string
		sub  string
		want bool
	}{
		{"exact match", "hello", "hello", true},
		{"case mismatch", "HELLO", "hello", true},
		{"substring", "hello world", "world", true},
		{"case substring", "Hello World", "WORLD", true},
		{"not found", "hello", "xyz", false},
		{"empty substr", "hello", "", true},
		{"empty string", "", "hello", false},
		{"both empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsIgnoreCase(tt.s, tt.sub)
			if got != tt.want {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.sub, got, tt.want)
			}
		})
	}
}

func TestPlaceholderCount(t *testing.T) {
	// Verify we have at least 40 placeholder phrases as required by the spec.
	if len(answerPlaceholderPhrases) < 40 {
		t.Errorf("expected at least 40 placeholder phrases, got %d", len(answerPlaceholderPhrases))
	}
}

func TestRateLimitPatternCount(t *testing.T) {
	// Verify we have at least 8 rate limit patterns as required.
	if len(rateLimitPatterns) < 8 {
		t.Errorf("expected at least 8 rate limit patterns, got %d", len(rateLimitPatterns))
	}
}

// Golden test for answer sanitization
func TestSanitizeAnswerGolden(t *testing.T) {
	// Simulated real answer with leaked Material icon labels
	input := "Based on the sources provided, NotebookLM is an AI-powered research and study tool developed by Google. It allows users to upload documents and ask questions about them. content_copy thumb_up thumb_down share"
	want := "Based on the sources provided, NotebookLM is an AI-powered research and study tool developed by Google. It allows users to upload documents and ask questions about them."

	got := SanitizeAnswer(input)
	if got != want {
		t.Errorf("golden test failed:\ngot:  %q\nwant: %q", got, want)
	}
}
