package render

import "strings"

// flatten collapses a comment body to a single line: newlines and runs of
// whitespace become single spaces. Used for the compact (truncated) view.
func flatten(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// truncate returns a single-line, width-limited form of body plus the number
// of runes omitted. width <= 0 means no limit (the full flattened line).
func truncate(body string, width int) (text string, omitted int) {
	flat := flatten(body)
	if width <= 0 {
		return flat, 0
	}
	r := []rune(flat)
	if len(r) <= width {
		return flat, 0
	}
	return strings.TrimRight(string(r[:width]), " ") + "…", len(r) - width
}

// indentBlock left-pads every line of s by pad spaces (used by --full).
func indentBlock(s, pad string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, ln := range lines {
		lines[i] = pad + ln
	}
	return strings.Join(lines, "\n")
}
