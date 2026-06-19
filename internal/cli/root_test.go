package cli

import (
	"reflect"
	"testing"
)

// commentValueFlags mirrors the value-taking (non-boolean) flags of `ghx
// comments`, so splitPR tests exercise the real flag arities.
var commentValueFlags = map[string]bool{
	"author": true, "thread": true, "lines": true, "width": true,
	"repo": true, "R": true,
}

func TestSplitPR(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPR   string
		wantRest []string
	}{
		{"no args", nil, "", []string{}},
		{"bare PR", []string{"123"}, "123", []string{}},
		{"hashed PR", []string{"#123"}, "#123", []string{}},
		{"PR before flags", []string{"123", "--json"}, "123", []string{"--json"}},
		{"PR after flags", []string{"--json", "123"}, "123", []string{"--json"}},
		// Regression: a numeric value of a value-taking flag must not be
		// mistaken for the PR (this broke `--width 100`, `--lines 4`, etc.).
		{"value flag space form", []string{"--width", "100"}, "", []string{"--width", "100"}},
		{"value flag then PR", []string{"--width", "100", "42"}, "42", []string{"--width", "100"}},
		{"PR then value flag", []string{"42", "--lines", "4"}, "42", []string{"--lines", "4"}},
		{"single-dash value flag", []string{"-thread", "2"}, "", []string{"-thread", "2"}},
		{"equals form keeps PR", []string{"--width=100", "42"}, "42", []string{"--width=100"}},
		// Boolean flags don't consume the next token, so a following number is
		// still the PR.
		{"bool flag then PR", []string{"--all", "42"}, "42", []string{"--all"}},
		{"only first numeric is PR", []string{"42", "99"}, "42", []string{"99"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr, rest := splitPR(tt.args, commentValueFlags)
			if pr != tt.wantPR {
				t.Errorf("pr = %q, want %q", pr, tt.wantPR)
			}
			if !reflect.DeepEqual(rest, tt.wantRest) {
				t.Errorf("rest = %#v, want %#v", rest, tt.wantRest)
			}
		})
	}
}
