package render

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/wordwrap"
)

// eastAsian measures display width treating ambiguous-width glyphs (arrows like
// "↳", the "…" ellipsis, etc.) as two cells. Many terminals render them that
// way; counting the wider case means we never *under*-reserve and overflow the
// right edge. The cost on terminals that render them as one cell is a harmless
// extra column of indent.
var eastAsian = func() *runewidth.Condition {
	c := runewidth.NewCondition()
	c.EastAsianWidth = true
	return c
}()

// cellWidth is the display width of s in terminal cells (ambiguous = 2).
func cellWidth(s string) int { return eastAsian.StringWidth(s) }

// flatten collapses a body to a single logical line: newlines and runs of
// whitespace become single spaces. Bodies are treated as prose, then re-wrapped
// to the terminal width.
func flatten(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// wrapBody word-wraps a (flattened) body to width and caps it at maxLines,
// appending an ellipsis to the last line when content was dropped. Every
// returned line is hard-clamped to width as a backstop, so unbreakable tokens
// (long URLs) and glyph-width slop can never overflow. maxLines <= 0 = no cap.
func wrapBody(body string, width, maxLines int) []string {
	if width < 10 {
		width = 10
	}
	wrapped := wordwrap.String(flatten(body), width)
	lines := strings.Split(strings.TrimRight(wrapped, "\n"), "\n")
	for i, ln := range lines {
		lines[i] = clampWidth(ln, width)
	}
	if maxLines > 0 && len(lines) > maxLines {
		kept := lines[:maxLines]
		const ell = "…"
		kept[maxLines-1] = clampWidth(kept[maxLines-1], width-cellWidth(ell)) + ell
		return kept
	}
	return lines
}

// clampWidth hard-trims a string to a display width in cells (no wrapping).
func clampWidth(s string, width int) string {
	if cellWidth(s) <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && cellWidth(string(r)) > width {
		r = r[:len(r)-1]
	}
	return strings.TrimRight(string(r), " ")
}
