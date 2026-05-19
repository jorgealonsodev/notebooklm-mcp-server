package notebooklm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/playwright-community/playwright-go"
)

// CitationFormat defines the output format for citations.
type CitationFormat string

const (
	CitationNone      CitationFormat = "none"
	CitationInline    CitationFormat = "inline"
	CitationFootnotes CitationFormat = "footnotes"
	CitationJSON      CitationFormat = "json"
)

// FormattedCitation holds a citation with its extracted text and format.
type FormattedCitation struct {
	Index    int
	Quote    string
	SourceID string
}

// ExtractCitations finds all citation markers in the latest answer, clicks each
// sequentially to read the highlighted source text, and formats the output
// according to the specified format.
func ExtractCitations(page playwright.Page, format CitationFormat) ([]FormattedCitation, error) {
	// Find all citation markers in the latest answer
	markers, err := page.QuerySelectorAll(CitationMarker)
	if err != nil {
		return nil, fmt.Errorf("find citation markers: %w", err)
	}

	if len(markers) == 0 {
		return nil, nil // No citations
	}

	var citations []FormattedCitation

	for i, marker := range markers {
		el := marker.(playwright.ElementHandle)

		// Read the citation index from the marker text
		markerText, err := el.InnerText()
		if err != nil {
			continue
		}
		idx := parseCitationIndex(markerText)
		if idx == 0 {
			idx = i + 1 // Fallback to position-based index
		}

		// Click the marker to open the source panel
		if err := el.Click(); err != nil {
			continue
		}

		// Wait for the source panel to show highlighted text
		highlighted, err := page.WaitForSelector(HighlightedText, playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})
		if err != nil {
			// Try to close and continue
			_ = page.Keyboard().Press("Escape")
			continue
		}

		// Read the highlighted text
		var quote string
		if highlighted != nil {
			hl := highlighted.(playwright.ElementHandle)
			quote, _ = hl.InnerText()
		}

		citations = append(citations, FormattedCitation{
			Index: idx,
			Quote: strings.TrimSpace(quote),
		})

		// Close the source panel
		_ = page.Keyboard().Press("Escape")
	}

	return citations, nil
}

// FormatCitations formats the answer text with citations in the specified format.
func FormatCitations(answer string, citations []FormattedCitation, format CitationFormat) string {
	switch format {
	case CitationNone:
		return answer
	case CitationInline:
		return formatInline(answer, citations)
	case CitationFootnotes:
		return formatFootnotes(answer, citations)
	case CitationJSON:
		return formatJSON(citations)
	default:
		return answer
	}
}

// formatInline replaces [N] markers with the excerpt text in parentheses.
func formatInline(answer string, citations []FormattedCitation) string {
	result := answer
	for _, c := range citations {
		marker := fmt.Sprintf("[%d]", c.Index)
		replacement := fmt.Sprintf("(%s)", c.Quote)
		result = strings.ReplaceAll(result, marker, replacement)
	}
	return result
}

// formatFootnotes keeps [N] markers and appends a footnote block.
func formatFootnotes(answer string, citations []FormattedCitation) string {
	if len(citations) == 0 {
		return answer
	}

	var sb strings.Builder
	sb.WriteString(answer)
	sb.WriteString("\n\n---\n\n")
	for _, c := range citations {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", c.Index, c.Quote))
	}
	return sb.String()
}

// formatJSON returns a JSON-like structured representation of citations.
func formatJSON(citations []FormattedCitation) string {
	if len(citations) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, c := range citations {
		sb.WriteString(fmt.Sprintf("  {\"index\": %d, \"quote\": %q}", c.Index, escapeJSONString(c.Quote)))
		if i < len(citations)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("]")
	return sb.String()
}

// parseCitationIndex extracts the numeric index from a citation marker like "[1]" or "1".
func parseCitationIndex(text string) int {
	// Try to find a number in brackets
	re := regexp.MustCompile(`\[(\d+)\]`)
	match := re.FindStringSubmatch(text)
	if len(match) >= 2 {
		n, err := strconv.Atoi(match[1])
		if err == nil {
			return n
		}
	}
	// Try plain number
	re2 := regexp.MustCompile(`^(\d+)$`)
	match2 := re2.FindStringSubmatch(strings.TrimSpace(text))
	if len(match2) >= 2 {
		n, err := strconv.Atoi(match2[1])
		if err == nil {
			return n
		}
	}
	return 0
}

// escapeJSONString escapes special characters for JSON string values.
func escapeJSONString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}
