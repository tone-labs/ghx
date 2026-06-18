package render

import (
	"strings"
	"testing"
)

func TestFlatten(t *testing.T) {
	if got := flatten("  a\n\tb   c \n"); got != "a b c" {
		t.Errorf("flatten = %q, want %q", got, "a b c")
	}
}

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
}
