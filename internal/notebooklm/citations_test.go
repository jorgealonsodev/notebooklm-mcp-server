package notebooklm

import (
	"strings"
	"testing"
)

func TestCitationFormat_Constants(t *testing.T) {
	tests := []struct {
		name   string
		format CitationFormat
		want   string
	}{
		{"none", CitationNone, "none"},
		{"inline", CitationInline, "inline"},
		{"footnotes", CitationFootnotes, "footnotes"},
		{"json", CitationJSON, "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.format) != tt.want {
				t.Errorf("CitationFormat = %q, want %q", tt.format, tt.want)
			}
		})
	}
}

func TestParseCitationIndex(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{"bracket number", "[1]", 1},
		{"bracket double digit", "[12]", 12},
		{"plain number", "3", 3},
		{"plain number with spaces", " 5 ", 5},
		{"no number", "source", 0},
		{"empty", "", 0},
		{"text with bracket", "[source]", 0},
		{"multiple brackets", "[1][2]", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCitationIndex(tt.text)
			if got != tt.want {
				t.Errorf("parseCitationIndex(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}

func TestFormatCitations_None(t *testing.T) {
	answer := "This is the answer [1] with citations [2]."
	citations := []FormattedCitation{
		{Index: 1, Quote: "Quote one"},
		{Index: 2, Quote: "Quote two"},
	}

	got := FormatCitations(answer, citations, CitationNone)
	if got != answer {
		t.Errorf("FormatCitations(none) modified the answer:\ngot:  %q\nwant: %q", got, answer)
	}
}

func TestFormatCitations_Inline(t *testing.T) {
	answer := "This is the answer [1] with citations [2]."
	citations := []FormattedCitation{
		{Index: 1, Quote: "Quote one"},
		{Index: 2, Quote: "Quote two"},
	}

	got := FormatCitations(answer, citations, CitationInline)
	want := "This is the answer (Quote one) with citations (Quote two)."
	if got != want {
		t.Errorf("FormatCitations(inline) =\ngot:  %q\nwant: %q", got, want)
	}
}

func TestFormatCitations_Inline_NoCitations(t *testing.T) {
	answer := "This is the answer with no citations."
	citations := []FormattedCitation{}

	got := FormatCitations(answer, citations, CitationInline)
	if got != answer {
		t.Errorf("FormatCitations(inline) with no citations modified answer:\ngot:  %q\nwant: %q", got, answer)
	}
}

func TestFormatCitations_Footnotes(t *testing.T) {
	answer := "This is the answer [1] with citations [2]."
	citations := []FormattedCitation{
		{Index: 1, Quote: "Quote one"},
		{Index: 2, Quote: "Quote two"},
	}

	got := FormatCitations(answer, citations, CitationFootnotes)

	// Should contain original answer
	if !strings.Contains(got, answer) {
		t.Errorf("footnotes format should contain original answer")
	}
	// Should contain footnote block separator
	if !strings.Contains(got, "---") {
		t.Errorf("footnotes format should contain separator")
	}
	// Should contain footnote entries
	if !strings.Contains(got, "[1] Quote one") {
		t.Errorf("footnotes format should contain [1] footnote")
	}
	if !strings.Contains(got, "[2] Quote two") {
		t.Errorf("footnotes format should contain [2] footnote")
	}
}

func TestFormatCitations_Footnotes_NoCitations(t *testing.T) {
	answer := "This is the answer with no citations."
	citations := []FormattedCitation{}

	got := FormatCitations(answer, citations, CitationFootnotes)
	if got != answer {
		t.Errorf("FormatCitations(footnotes) with no citations modified answer:\ngot:  %q\nwant: %q", got, answer)
	}
}

func TestFormatCitations_JSON(t *testing.T) {
	citations := []FormattedCitation{
		{Index: 1, Quote: "Quote one"},
		{Index: 2, Quote: "Quote two"},
	}

	got := FormatCitations("", citations, CitationJSON)

	// Should be valid-looking JSON array
	if !strings.HasPrefix(got, "[") {
		t.Errorf("JSON format should start with [")
	}
	if !strings.HasSuffix(got, "]") {
		t.Errorf("JSON format should end with ]")
	}
	// Should contain citation data
	if !strings.Contains(got, `"index": 1`) {
		t.Errorf("JSON format should contain index 1")
	}
	if !strings.Contains(got, `"quote": "Quote one"`) {
		t.Errorf("JSON format should contain quote one")
	}
	if !strings.Contains(got, `"index": 2`) {
		t.Errorf("JSON format should contain index 2")
	}
}

func TestFormatCitations_JSON_NoCitations(t *testing.T) {
	citations := []FormattedCitation{}

	got := FormatCitations("", citations, CitationJSON)
	if got != "[]" {
		t.Errorf("FormatCitations(json) with no citations = %q, want []", got)
	}
}

func TestFormatCitations_JSON_Escaping(t *testing.T) {
	citations := []FormattedCitation{
		{Index: 1, Quote: `Quote with "quotes" and \backslash`},
	}

	got := FormatCitations("", citations, CitationJSON)

	// Should properly escape quotes and backslashes
	if strings.Contains(got, `"Quote with "quotes"`) {
		t.Errorf("JSON format should escape quotes in quote text")
	}
}

func TestFormatCitations_JSON_NewlineEscaping(t *testing.T) {
	citations := []FormattedCitation{
		{Index: 1, Quote: "Line one\nLine two"},
	}

	got := FormatCitations("", citations, CitationJSON)

	// Should escape newlines
	if strings.Contains(got, "\nLine two") {
		t.Errorf("JSON format should escape newlines in quote text")
	}
	if !strings.Contains(got, `\n`) {
		t.Errorf("JSON format should contain escaped newline")
	}
}

func TestFormattedCitation_StructFields(t *testing.T) {
	c := FormattedCitation{
		Index:    3,
		Quote:    "Important excerpt",
		SourceID: "src-123",
	}

	if c.Index != 3 {
		t.Errorf("Index = %d, want 3", c.Index)
	}
	if c.Quote != "Important excerpt" {
		t.Errorf("Quote = %q, want %q", c.Quote, "Important excerpt")
	}
	if c.SourceID != "src-123" {
		t.Errorf("SourceID = %q, want %q", c.SourceID, "src-123")
	}
}

// TestIntegration_ExtractCitations is guarded by testing.Short().
func TestIntegration_ExtractCitations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

// TestAllCitationFormats tests all 4 format modes with the same input.
func TestAllCitationFormats(t *testing.T) {
	answer := "The answer [1] is clear [2] and [3] definitive."
	citations := []FormattedCitation{
		{Index: 1, Quote: "Source A says X"},
		{Index: 2, Quote: "Source B says Y"},
		{Index: 3, Quote: "Source C says Z"},
	}

	formats := []CitationFormat{CitationNone, CitationInline, CitationFootnotes, CitationJSON}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			result := FormatCitations(answer, citations, format)

			switch format {
			case CitationNone:
				if result != answer {
					t.Errorf("none format should return original answer")
				}
			case CitationInline:
				if strings.Contains(result, "[1]") {
					t.Errorf("inline format should replace [1]")
				}
				if !strings.Contains(result, "(Source A says X)") {
					t.Errorf("inline format should contain inline quote")
				}
			case CitationFootnotes:
				if !strings.Contains(result, "[1] Source A says X") {
					t.Errorf("footnotes format should contain footnote entry")
				}
			case CitationJSON:
				if !strings.Contains(result, `"index": 1`) {
					t.Errorf("json format should contain index")
				}
			}
		})
	}
}

func TestEscapeJSONString(t *testing.T) {
	tests := []struct {
		name string
		input string
		want  string
	}{
		{"no escaping needed", "hello", "hello"},
		{"backslash", `hello\world`, `hello\\world`},
		{"double quote", `hello "world"`, `hello \"world\"`},
		{"newline", "hello\nworld", `hello\nworld`},
		{"tab", "hello\tworld", `hello\tworld`},
		{"carriage return", "hello\rworld", `hello\rworld`},
		{"multiple escapes", `a"b\c`, `a\"b\\c`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeJSONString(tt.input)
			if got != tt.want {
				t.Errorf("escapeJSONString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
