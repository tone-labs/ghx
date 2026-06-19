package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tone-labs/ghx/internal/model"
)

func threadAt(id string, resolved bool) model.Thread {
	return model.Thread{ID: id, IsResolved: resolved, Path: "f.go", Line: 1,
		Comments: []model.Comment{{Author: "bob", Body: "why?"}}}
}

// TestActionableThreads pins the selection that makes `ghx resolve N` line up
// with what the verb can act on: resolve sees only unresolved threads,
// unresolve only resolved — both in original (listing) order.
func TestActionableThreads(t *testing.T) {
	threads := []model.Thread{
		threadAt("a", false), // unresolved
		threadAt("b", true),  // resolved
		threadAt("c", false), // unresolved
	}

	resolveTargets := actionableThreads(threads, true)
	if got := ids(resolveTargets); got != "a,c" {
		t.Errorf("resolve targets = %q, want \"a,c\" (unresolved, in order)", got)
	}
	unresolveTargets := actionableThreads(threads, false)
	if got := ids(unresolveTargets); got != "b" {
		t.Errorf("unresolve targets = %q, want \"b\" (resolved)", got)
	}

	if n := len(actionableThreads(nil, true)); n != 0 {
		t.Errorf("no threads: got %d targets, want 0", n)
	}
}

func TestThreadLoc(t *testing.T) {
	cases := map[string]model.Thread{
		"f.go:7":       {Path: "f.go", Line: 7},
		"f.go":         {Path: "f.go", Line: 0},
		"(file-level)": {Path: "", Line: 0},
	}
	for want, th := range cases {
		if got := threadLoc(th); got != want {
			t.Errorf("threadLoc(%+v) = %q, want %q", th, got, want)
		}
	}
}

func TestStarterSnippet(t *testing.T) {
	long := strings.Repeat("x", 80)
	th := model.Thread{Comments: []model.Comment{{Author: "ann", Body: long}}}
	got := starterSnippet(th)
	if !strings.HasPrefix(got, "ann: ") || !strings.HasSuffix(got, "…") {
		t.Errorf("snippet = %q, want truncated 'ann: …' form", got)
	}
	if len([]rune(got)) > 5+60+1 { // "ann: " + 60 chars + ellipsis
		t.Errorf("snippet not truncated: %q (%d runes)", got, len([]rune(got)))
	}

	// First line only.
	multi := model.Thread{Comments: []model.Comment{{Author: "ann", Body: "first\nsecond"}}}
	if got := starterSnippet(multi); got != "ann: first" {
		t.Errorf("snippet = %q, want \"ann: first\"", got)
	}

	// Empty thread → no snippet.
	if got := starterSnippet(model.Thread{}); got != "" {
		t.Errorf("empty thread snippet = %q, want \"\"", got)
	}
}

func TestListThreadTargets(t *testing.T) {
	var buf bytes.Buffer
	targets := []model.Thread{threadAt("a", false), threadAt("c", false)}
	listThreadTargets(&buf, targets, 42, "resolve", "unresolved")
	out := buf.String()
	for _, want := range []string{"unresolved threads on #42", "  1  f.go:1", "  2  f.go:1", "Resolve one with: ghx resolve --thread <N>"} {
		if !strings.Contains(out, want) {
			t.Errorf("listing missing %q\n--- got ---\n%s", want, out)
		}
	}

	// Empty case.
	var empty bytes.Buffer
	listThreadTargets(&empty, nil, 42, "unresolve", "resolved")
	if got := empty.String(); !strings.Contains(got, "no resolved threads on #42") {
		t.Errorf("empty listing = %q", got)
	}
}

func ids(threads []model.Thread) string {
	var s []string
	for _, t := range threads {
		s = append(s, t.ID)
	}
	return strings.Join(s, ",")
}
