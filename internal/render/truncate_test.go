package render

import "testing"

func TestTruncate(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		width       int
		wantText    string
		wantOmitted int
	}{
		{"under width", "hello world", 20, "hello world", 0},
		{"flatten newlines", "line one\n\n  line two", 50, "line one line two", 0},
		{"exact boundary", "abcde", 5, "abcde", 0},
		{"over width", "abcdefghij", 5, "abcde…", 5},
		{"no limit", "a long body here", 0, "a long body here", 0},
		{"multibyte counts runes", "héllo wörld", 5, "héllo…", 6},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text, omitted := truncate(c.body, c.width)
			if text != c.wantText || omitted != c.wantOmitted {
				t.Errorf("truncate(%q,%d) = (%q,%d), want (%q,%d)",
					c.body, c.width, text, omitted, c.wantText, c.wantOmitted)
			}
		})
	}
}

func TestFlatten(t *testing.T) {
	if got := flatten("  a\n\tb   c \n"); got != "a b c" {
		t.Errorf("flatten = %q, want %q", got, "a b c")
	}
}
