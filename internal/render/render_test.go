package render

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tone-labs/ghx/internal/model"
)

var update = flag.Bool("update", false, "update golden files")

func samplePR() *model.PR {
	return &model.PR{
		Number:         42,
		Title:          "Add xpath support",
		URL:            "https://github.com/o/r/pull/42",
		State:          "OPEN",
		IsDraft:        true,
		Author:         "alice",
		ReviewDecision: "CHANGES_REQUESTED",
		Reviews: []model.Review{
			{Author: "bob", State: "COMMENTED", SubmittedAt: "2026-01-01T00:00:00Z"},
			{Author: "bob", State: "CHANGES_REQUESTED", Body: "Needs work", SubmittedAt: "2026-01-02T00:00:00Z"},
			{Author: "ci-bot[bot]", IsBot: true, State: "APPROVED", SubmittedAt: "2026-01-01T12:00:00Z"},
		},
		Threads: []model.Thread{
			{
				ID: "T_open", Path: "src/a.ts", Line: 72, IsResolved: false,
				Comments: []model.Comment{
					{Author: "bob", Body: "Why does this return element-not-found?"},
					{Author: "alice", Body: "Good catch, fixing."},
				},
			},
			{
				ID: "T_resolved", Path: "src/b.ts", Line: 10, IsResolved: true, IsOutdated: true,
				Comments: []model.Comment{
					{Author: "lint-bot[bot]", IsBot: true, Body: "nit: prefer const here for the long explanation that should be truncated in the compact view"},
				},
			},
		},
		Conversation: []model.Comment{
			{Author: "carol", Body: "LGTM overall once threads are addressed."},
		},
	}
}

func TestCommentsGolden(t *testing.T) {
	var buf bytes.Buffer
	Comments(&buf, samplePR(), Options{BodyLines: defaultLines})
	checkGolden(t, "comments_default.golden", buf.Bytes())
}

func TestCommentsFullGolden(t *testing.T) {
	var buf bytes.Buffer
	Comments(&buf, samplePR(), Options{BodyLines: 0, ShowConversation: true})
	checkGolden(t, "comments_full.golden", buf.Bytes())
}

const defaultLines = 2

// TestNoLineOverflow guards the wrapping bug class: at realistic widths, no
// rendered line may exceed the budget in terminal cells — in particular the
// wrapped comment bodies, whose "↳" reply marker is an ambiguous-width glyph a
// terminal may render two cells wide. Single-line structural rows (the BLUF
// status line, section/thread headers) are intentionally not wrapped, so the
// widths tested stay above their natural length. Width is set via Options.Width
// to avoid TTY detection.
func TestNoLineOverflow(t *testing.T) {
	for _, width := range []int{70, 80, 100, 120} {
		var buf bytes.Buffer
		Comments(&buf, samplePR(), Options{Width: width, BodyLines: 0, ShowConversation: true})
		for line := range strings.SplitSeq(strings.TrimRight(buf.String(), "\n"), "\n") {
			if w := cellWidth(line); w > width {
				t.Errorf("width=%d: line exceeds budget (%d cells): %q", width, w, line)
			}
		}
	}
}

// TestWriteBodyNoOverflow exercises the indent arithmetic directly at narrow
// widths and with a long author label — the case TestNoLineOverflow can't reach
// because its unwrapped status line forces widths ≥ ~58. With a 31-cell label,
// width=40 drives writeBody's body-budget floor (pulling the continuation indent
// back in); width=50/70 are the normal path. Every emitted line — first and
// continuation — must stay within the budget.
func TestWriteBodyNoOverflow(t *testing.T) {
	const label = "a-very-long-reviewer-username!!" // 31 cells, still < width below
	body := "This is a multi-line review comment long enough to wrap onto several continuation lines."
	for _, width := range []int{40, 50, 70} {
		var buf bytes.Buffer
		s := newStyles(&buf, ColorAuto)
		writeBody(&buf, s, 6, label, body, width, 0)
		for line := range strings.SplitSeq(strings.TrimRight(buf.String(), "\n"), "\n") {
			if w := cellWidth(line); w > width {
				t.Errorf("width=%d: line exceeds budget (%d cells): %q", width, w, line)
			}
		}
	}
}

// TestColorMode verifies --color plumbing: writing to a (non-TTY) buffer,
// `auto` and `never` stay plain while `always` forces ANSI escapes.
func TestColorMode(t *testing.T) {
	hasANSI := func(s string) bool { return strings.Contains(s, "\x1b[") }
	cases := []struct {
		mode    ColorMode
		wantSeq bool
	}{
		{ColorAuto, false},  // non-TTY → no color
		{ColorNever, false}, // explicitly off
		{ColorAlways, true}, // forced even when piped
	}
	for _, tc := range cases {
		var buf bytes.Buffer
		Comments(&buf, samplePR(), Options{Color: tc.mode, BodyLines: defaultLines})
		if got := hasANSI(buf.String()); got != tc.wantSeq {
			t.Errorf("Color=%d: ANSI present = %v, want %v", tc.mode, got, tc.wantSeq)
		}
	}
}

func TestChecksGolden(t *testing.T) {
	ck := &model.Checks{
		Counts: map[string]int{"pass": 3, "fail": 1, "pending": 2},
		Total:  6,
		Failing: []model.Check{
			{Name: "lint", Bucket: "fail", State: "FAILURE", Workflow: "CI", Link: "https://x/runs/1"},
		},
	}
	var buf bytes.Buffer
	ChecksView(&buf, 42, ck)
	checkGolden(t, "checks.golden", buf.Bytes())
}

func checkGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run `go test -update` to create): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("output mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}
