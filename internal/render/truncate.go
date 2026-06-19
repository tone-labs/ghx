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
	var lines []string
	for ln := range strings.SplitSeq(strings.TrimRight(wrapped, "\n"), "\n") {
		// wordwrap only breaks on spaces, so an unbreakable run — a long URL or
		// space-free CJK text — can still exceed width. Hard-split those at cell
		// boundaries so the overflow reflows onto more lines instead of being
		// silently clamped away (which dropped content with no ellipsis).
		lines = append(lines, hardWrap(ln, width)...)
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

// hardWrap splits s into consecutive chunks each at most width display cells,
// breaking mid-token at rune boundaries. It's the backstop for runs wordwrap
// can't break (long URLs, space-free CJK): unlike clampWidth, which discards
// the overflow, hardWrap preserves every rune by flowing it onto another line.
// A string already within width is returned unchanged as a single chunk.
func hardWrap(s string, width int) []string {
	if cellWidth(s) <= width {
		return []string{s}
	}
	var chunks []string
	var cur strings.Builder
	curw := 0
	for _, r := range s {
		rw := cellWidth(string(r))
		if curw+rw > width {
			chunks = append(chunks, cur.String())
			cur.Reset()
			curw = 0
		}
		cur.WriteRune(r)
		curw += rw
	}
	if cur.Len() > 0 {
		chunks = append(chunks, cur.String())
	}
	return chunks
}
