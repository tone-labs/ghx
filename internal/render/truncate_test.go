package render

import (
	"strings"
	"testing"
)

func TestWrapBody(t *testing.T) {
	t.Run("short fits one line", func(t *testing.T) {
		lines := wrapBody("hello world", 40, 2)
		if len(lines) != 1 || lines[0] != "hello world" {
			t.Errorf("got %q", lines)
		}
	})

	t.Run("wraps to multiple lines", func(t *testing.T) {
		lines := wrapBody("one two three four five six seven", 12, 0)
		if len(lines) < 2 {
			t.Errorf("expected multiple lines, got %q", lines)
		}
	})

	t.Run("caps at maxLines with ellipsis", func(t *testing.T) {
		lines := wrapBody("one two three four five six seven eight nine ten", 10, 2)
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d: %q", len(lines), lines)
		}
		if !strings.HasSuffix(lines[1], "…") {
			t.Errorf("last capped line should end with ellipsis, got %q", lines[1])
		}
	})

	t.Run("flattens newlines before wrapping", func(t *testing.T) {
		lines := wrapBody("a\n\nb", 40, 0)
		if len(lines) != 1 || lines[0] != "a b" {
			t.Errorf("got %q", lines)
		}
	})

	t.Run("hard-wraps space-free CJK without dropping content", func(t *testing.T) {
		// 30 CJK runes × 2 cells = 60 cells, no spaces: wordwrap can't break it.
		// Uncapped, every rune must survive across multiple width-10 lines.
		body := strings.Repeat("中", 30)
		lines := wrapBody(body, 10, 0)
		if len(lines) < 2 {
			t.Fatalf("expected hard-wrap onto multiple lines, got %q", lines)
		}
		if joined := strings.Join(lines, ""); joined != body {
			t.Errorf("content lost: joined %q != body %q", joined, body)
		}
		for _, ln := range lines {
			if w := cellWidth(ln); w > 10 {
				t.Errorf("line exceeds width: %d cells in %q", w, ln)
			}
		}
	})

	t.Run("caps space-free CJK with ellipsis, not silent drop", func(t *testing.T) {
		body := strings.Repeat("中", 30)
		lines := wrapBody(body, 10, 2)
		if len(lines) != 2 {
			t.Fatalf("expected 2 capped lines, got %d: %q", len(lines), lines)
		}
		if !strings.HasSuffix(lines[1], "…") {
			t.Errorf("truncated CJK should signal with ellipsis, got %q", lines[1])
		}
	})

	t.Run("breaks at word boundary, never one cell over (no mid-word split)", func(t *testing.T) {
		// Regression: muesli/wordwrap returned a line one cell over the width,
		// and the hard-clamp split a word mid-character ("lets" -> "let"/"s").
		// wrapText must break at the last word boundary that fits.
		body := "Not a switch - it's what we'd discussed for smoothing bursts. The token-bucket lets a client spend accumulated headroom."
		lines := wrapBody(body, 82, 0)
		for _, ln := range lines {
			if w := cellWidth(ln); w > 82 {
				t.Errorf("line exceeds width: %d cells in %q", w, ln)
			}
		}
		if joined := strings.Join(lines, " "); joined != body {
			t.Errorf("word-wrap altered content:\n got  %q\n want %q", joined, body)
		}
	})

	t.Run("hard-wraps an unbreakable URL", func(t *testing.T) {
		url := "https://example.com/" + strings.Repeat("a", 50)
		lines := wrapBody(url, 20, 0)
		if joined := strings.Join(lines, ""); joined != url {
			t.Errorf("URL content lost: %q", joined)
		}
		for _, ln := range lines {
			if w := cellWidth(ln); w > 20 {
				t.Errorf("line exceeds width: %d cells in %q", w, ln)
			}
		}
	})
}
