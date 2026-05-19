// Package notebooklm provides domain-level automation for NotebookLM:
// chat, sources, audio overviews, and citation extraction.
package notebooklm

// CSS selector constants for NotebookLM UI elements.
const (
	// ChatInput is the primary selector for the question input textarea.
	ChatInput = "textarea.query-box-input"

	// AnswerContainer selects all user-facing answer containers.
	AnswerContainer = ".to-user-container .message-text-content"

	// LatestAnswer selects only the most recent answer.
	LatestAnswer = ".to-user-container:last-child .message-text-content"

	// AddSourceButton selects the button to add a new source.
	AddSourceButton = "button.add-source-button"

	// SourceRow selects individual source entries in the source list.
	SourceRow = ".single-source-container"

	// SourceCountHeader selects the element showing the total source count.
	SourceCountHeader = ".cover-subtitle-source-count"

	// Dialog selects any modal dialog on the page.
	Dialog = "[role=\"dialog\"]"

	// AudioOverviewButton selects the audio overview generation button by icon text.
	AudioOverviewButton = "mat-icon:text-is(\"audio_magic_eraser\")"

	// AudioPlayerTile selects the audio artifact item in the library.
	AudioPlayerTile = ".artifact-library-item"

	// CitationMarker selects inline citation reference buttons.
	CitationMarker = "button.citation-marker"

	// HighlightedText selects the highlighted text in a source panel.
	HighlightedText = ".highlighted"
)

// ChatInputAriaLabels contains locale-specific aria-label fallbacks for the chat input.
// The primary selector (ChatInput) is tried first; if it fails, these aria-labels
// are tried in order.
var ChatInputAriaLabels = []string{
	"Ask a question",
	"Posez une question",       // French
	"Stellen Sie eine Frage",   // German
	"Haz una pregunta",         // Spanish
	"Faça uma pergunta",        // Portuguese
	"Fai una domanda",          // Italian
	"質問する",                  // Japanese
	"提出问题",                  // Chinese (Simplified)
	"提出問題",                  // Chinese (Traditional)
	" задайте вопрос",           // Russian (with leading space for robustness)
	" задай вопрос",            // Russian (informal)
}

// answerPlaceholderPhrases contains 40+ loading/thinking placeholder strings
// across 8+ languages that must be filtered out when detecting stable answers.
var answerPlaceholderPhrases = []string{
	// English
	"Thinking",
	"Thinking...",
	"Generating",
	"Generating response",
	"Generating response...",
	"Processing",
	"Processing your question",
	"Loading",
	"Loading response",
	"Loading response...",
	"Please wait",
	"Please wait...",
	"Analyzing",
	"Analyzing sources",
	"Analyzing your question",
	"Reading sources",
	"Reading your sources",
	"Composing",
	"Composing response",
	"Composing answer",
	"Working on it",
	"Working on your answer",
	"Just a moment",
	"Just a moment...",
	"One moment please",
	// Spanish
	"Pensando",
	"Pensando...",
	"Generando",
	"Generando respuesta",
	"Cargando",
	"Cargando respuesta",
	"Analizando",
	"Analizando fuentes",
	"Componiendo",
	"Componiendo respuesta",
	"Un momento",
	"Un momento...",
	// French
	"Réflexion",
	"Réflexion...",
	"Génération",
	"Génération de la réponse",
	"Chargement",
	"Chargement de la réponse",
	"Analyse",
	"Analyse des sources",
	// German
	"Denken",
	"Denken...",
	"Generieren",
	"Antwort generieren",
	"Laden",
	"Antwort laden",
	// Portuguese
	"Pensando",
	"Gerando",
	"Gerando resposta",
	"Carregando",
	"Analisando",
	// Italian
	"Elaborazione",
	"Elaborazione della risposta",
	"Caricamento",
	// Japanese
	"考え中",
	"回答を生成中",
	"読み込み中",
	// Chinese
	"思考中",
	"正在生成回答",
	"加载中",
	"正在分析",
}

// rateLimitPatterns contains DOM error message substrings that indicate
// the user has hit NotebookLM's rate limit.
var rateLimitPatterns = []string{
	"rate limit",
	"too many requests",
	"try again later",
	"try again tomorrow",
	"daily limit",
	"quota exceeded",
	"limit reached",
	"come back tomorrow",
	"you've reached",
	"you have reached",
	"limit for today",
	"limit for free",
	"free tier limit",
	"50 queries",
	"queries per day",
	"slow down",
	"please wait",
	"temporarily unavailable",
}

// IsPlaceholder returns true if the given text matches a known loading/thinking
// placeholder phrase (case-insensitive, trimmed).
func IsPlaceholder(text string) bool {
	trimmed := stripWhitespace(text)
	for _, phrase := range answerPlaceholderPhrases {
		if trimmed == phrase {
			return true
		}
	}
	return false
}

// IsRateLimitMessage returns true if the given text matches a known rate limit
// error pattern (case-insensitive).
func IsRateLimitMessage(text string) bool {
	lower := stripWhitespace(text)
	for _, pattern := range rateLimitPatterns {
		if containsIgnoreCase(lower, pattern) {
			return true
		}
	}
	return false
}

// SanitizeAnswer removes Material icon labels and other UI noise from answer text.
func SanitizeAnswer(text string) string {
	// Strip common Material icon text patterns that leak into answer text
	// These appear as standalone words in the DOM text content
	iconLabels := []string{
		"content_copy",
		"thumb_up",
		"thumb_down",
		"share",
		"more_vert",
		"close",
		"play_arrow",
		"pause",
		"volume_up",
		"volume_off",
	}

	result := text
	for _, icon := range iconLabels {
		// Remove icon labels that appear as standalone text
		result = removeStandaloneWord(result, icon)
	}
	return result
}

// stripWhitespace trims and collapses internal whitespace for comparison.
func stripWhitespace(s string) string {
	result := make([]byte, 0, len(s))
	inSpace := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if !inSpace {
				result = append(result, ' ')
				inSpace = true
			}
		} else {
			result = append(result, c)
			inSpace = false
		}
	}
	// Trim leading/trailing space
	if len(result) > 0 && result[0] == ' ' {
		result = result[1:]
	}
	if len(result) > 0 && result[len(result)-1] == ' ' {
		result = result[:len(result)-1]
	}
	return string(result)
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	// Simple case-insensitive search
	lower := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		lower[i] = c
	}
	lowerSubstr := make([]byte, len(substr))
	for i := 0; i < len(substr); i++ {
		c := substr[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		lowerSubstr[i] = c
	}
	return bytesContains(lower, lowerSubstr)
}

// bytesContains checks if s contains substr (both lowercase byte slices).
func bytesContains(s, substr []byte) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		found := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				found = false
				break
			}
		}
		if found {
			return true
		}
	}
	return false
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// removeStandaloneWord removes occurrences of word that appear as standalone
// tokens (surrounded by whitespace or at string boundaries).
func removeStandaloneWord(s, word string) string {
	// Simple approach: remove " word " patterns, then handle boundaries
	result := s
	// Replace " word " with " "
	prefix := " " + word + " "
	for contains(result, prefix) {
		result = replaceFirst(result, prefix, " ")
	}
	// Handle word at start
	if len(result) >= len(word)+1 && result[:len(word)] == word && result[len(word)] == ' ' {
		result = result[len(word)+1:]
	}
	// Handle word at end
	if len(result) >= len(word)+1 && result[len(result)-len(word)-1] == ' ' && result[len(result)-len(word):] == word {
		result = result[:len(result)-len(word)-1]
	}
	// Handle word as entire string
	if result == word {
		result = ""
	}
	return result
}

// replaceFirst replaces the first occurrence of old with new in s.
func replaceFirst(s, old, new string) string {
	idx := containsIdx(s, old)
	if idx < 0 {
		return s
	}
	return s[:idx] + new + s[idx+len(old):]
}

// containsIdx returns the index of the first occurrence of substr in s, or -1.
func containsIdx(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(s) < len(substr) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
