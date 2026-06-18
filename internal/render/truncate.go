package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

// flatten collapses a body to a single logical line: newlines and runs of
// whitespace become single spaces. Bodies are treated as prose, then re-wrapped
// to the terminal width.
func flatten(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// wrapBody word-wraps a (flattened) body to width and caps it at maxLines,
// appending an ellipsis to the last line when content was dropped. maxLines <= 0
// means no cap. Returns at least one line (possibly empty).
func wrapBody(body string, width, maxLines int) []string {
	if width < 10 {
		width = 10
	}
	wrapped := wordwrap.String(flatten(body), width)
	lines := strings.Split(strings.TrimRight(wrapped, "\n"), "\n")
	if maxLines > 0 && len(lines) > maxLines {
		kept := lines[:maxLines]
		kept[maxLines-1] = clampWidth(kept[maxLines-1], width-1) + "…"
		return kept
	}
	return lines
}

// clampWidth hard-trims a string to a display width in runes (no wrapping).
func clampWidth(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r)) > width {
		r = r[:len(r)-1]
	}
	return strings.TrimRight(string(r), " ")
}
