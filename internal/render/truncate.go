package render

import (
	"strings"

	"github.com/mattn/go-runewidth"
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

// wrapBody word-wraps a body to width and caps it at maxLines, appending an
// ellipsis to the last line when content was dropped. maxLines <= 0 = no cap.
func wrapBody(body string, width, maxLines int) []string {
	if width < 10 {
		width = 10
	}
	lines := wrapText(body, width)
	if maxLines > 0 && len(lines) > maxLines {
		kept := lines[:maxLines]
		const ell = "…"
		kept[maxLines-1] = clampWidth(kept[maxLines-1], width-cellWidth(ell)) + ell
		return kept
	}
	return lines
}

// wrapText flattens whitespace, then greedily packs words, breaking at the last
// word boundary that fits — so a line is never one cell over the limit (the
// off-by-one that muesli/wordwrap exhibited and that a hard-clamp turned into a
// mid-word split). A single word wider than width (a long URL or space-free CJK
// run) is hard-split at cell boundaries via hardWrap rather than overflowing.
// Every returned line is guaranteed <= width cells.
func wrapText(body string, width int) []string {
	var lines []string
	cur := ""
	curw := 0
	flush := func() {
		if cur != "" {
			lines = append(lines, cur)
			cur, curw = "", 0
		}
	}
	for word := range strings.FieldsSeq(body) {
		ww := cellWidth(word)
		if ww > width {
			// ww > width, so hardWrap returns >= 2 chunks: emit all but the last,
			// keep the last as the running line.
			flush()
			chunks := hardWrap(word, width)
			lines = append(lines, chunks[:len(chunks)-1]...)
			cur, curw = chunks[len(chunks)-1], cellWidth(chunks[len(chunks)-1])
			continue
		}
		switch {
		case cur == "":
			cur, curw = word, ww
		case curw+1+ww <= width:
			cur += " " + word
			curw += 1 + ww
		default:
			flush()
			cur, curw = word, ww
		}
	}
	flush()
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
