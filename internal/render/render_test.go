package render

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tone-labs/ghx/internal/gate"
	"github.com/tone-labs/ghx/internal/model"
)

var update = flag.Bool("update", false, "update golden files")

// sampleNow is a fixed reference time for relative timestamps so the comment
// goldens stay stable. The sample's timestamps sit 1-2 days before it.
var sampleNow = time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

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
					{Author: "bob", CreatedAt: "2026-01-01T10:00:00Z", Body: "Why does this return element-not-found?"},
					{Author: "alice", CreatedAt: "2026-01-02T11:00:00Z", Body: "Good catch, fixing."},
				},
			},
			{
				ID: "T_resolved", Path: "src/b.ts", Line: 10, IsResolved: true, IsOutdated: true,
				Comments: []model.Comment{
					{Author: "lint-bot[bot]", IsBot: true, CreatedAt: "2026-01-01T09:00:00Z", Body: "nit: prefer const here for the long explanation that should be truncated in the compact view"},
				},
			},
		},
		Conversation: []model.Comment{
			{Author: "carol", CreatedAt: "2026-01-02T12:00:00Z", Body: "LGTM overall once threads are addressed."},
		},
	}
}

func TestCommentsGolden(t *testing.T) {
	var buf bytes.Buffer
	Comments(&buf, samplePR(), Options{BodyLines: defaultLines, Now: sampleNow})
	checkGolden(t, "comments_default.golden", buf.Bytes())
}

func TestCommentsFullGolden(t *testing.T) {
	var buf bytes.Buffer
	Comments(&buf, samplePR(), Options{BodyLines: 0, ShowConversation: true, Now: sampleNow})
	checkGolden(t, "comments_full.golden", buf.Bytes())
}

func TestRelativeTime(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	// empty, unparseable, and future timestamps all yield "".
	for _, ts := range []string{"", "not-a-time", "2026-01-11T00:00:00Z"} {
		if got := relativeTime(ts, now); got != "" {
			t.Errorf("relativeTime(%q) = %q, want \"\"", ts, got)
		}
	}
	// a past timestamp yields a humanized "… ago" string.
	if got := relativeTime("2026-01-08T12:00:00Z", now); !strings.HasSuffix(got, "ago") {
		t.Errorf("relativeTime(past) = %q, want a '… ago' string", got)
	}
}

func TestDisplayAuthor(t *testing.T) {
	if got := displayAuthor("github-actions[bot]"); got != "github-actions" {
		t.Errorf("displayAuthor(bot) = %q, want github-actions", got)
	}
	if got := displayAuthor("alice"); got != "alice" {
		t.Errorf("displayAuthor(human) = %q, want alice", got)
	}
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

// TestGateGolden feeds real gate.Evaluate output into GateView, so the goldens
// pin the Evaluate→GateView path end-to-end (not a hand-authored Result).
func TestGateGolden(t *testing.T) {
	blockedPR := &model.PR{
		Number: 42, Title: "Add xpath support", URL: "https://github.com/o/r/pull/42",
		State: "OPEN", ReviewDecision: "CHANGES_REQUESTED", MergeStateStatus: "BLOCKED",
		Threads: []model.Thread{{IsResolved: false}, {IsResolved: false}},
	}
	blockedCk := &model.Checks{Counts: map[string]int{}, Failing: []model.Check{{Bucket: "fail"}}}
	var buf bytes.Buffer
	GateView(&buf, gate.Evaluate(blockedPR, blockedCk), ColorAuto)
	checkGolden(t, "gate_blocked.golden", buf.Bytes())

	mergeablePR := &model.PR{
		Number: 42, Title: "Add xpath support", URL: "https://github.com/o/r/pull/42",
		State: "OPEN", ReviewDecision: "APPROVED", MergeStateStatus: "CLEAN",
	}
	var buf2 bytes.Buffer
	GateView(&buf2, gate.Evaluate(mergeablePR, &model.Checks{Counts: map[string]int{}}), ColorAuto)
	checkGolden(t, "gate_mergeable.golden", buf2.Bytes())

	// UNSTABLE is the key fix: non-required checks are red but the PR still
	// merges, so the verdict is MERGEABLE with the reds flagged "not required".
	unstablePR := &model.PR{
		Number: 42, Title: "Add xpath support", URL: "https://github.com/o/r/pull/42",
		State: "OPEN", ReviewDecision: "APPROVED", MergeStateStatus: "UNSTABLE",
	}
	unstableCk := &model.Checks{Counts: map[string]int{"pending": 1}, Failing: []model.Check{{Bucket: "fail"}}}
	var bufU bytes.Buffer
	GateView(&bufU, gate.Evaluate(unstablePR, unstableCk), ColorAuto)
	checkGolden(t, "gate_unstable.golden", bufU.Bytes())

	// DIRTY surfaces the conflict row; CONFLICTING mergeable corroborates it.
	conflictPR := &model.PR{
		Number: 42, Title: "Add xpath support", URL: "https://github.com/o/r/pull/42",
		State: "OPEN", ReviewDecision: "APPROVED", MergeStateStatus: "DIRTY", Mergeable: "CONFLICTING",
	}
	var bufC bytes.Buffer
	GateView(&bufC, gate.Evaluate(conflictPR, &model.Checks{Counts: map[string]int{}}), ColorAuto)
	checkGolden(t, "gate_conflict.golden", bufC.Bytes())

	// Merged is terminal: purple MERGED headline, breakdown still shown.
	mergedPR := &model.PR{
		Number: 42, Title: "Add xpath support", URL: "https://github.com/o/r/pull/42",
		State: "MERGED", ReviewDecision: "APPROVED",
		Threads: []model.Thread{{IsResolved: false}},
	}
	var buf3 bytes.Buffer
	GateView(&buf3, gate.Evaluate(mergedPR, &model.Checks{Counts: map[string]int{}}), ColorAuto)
	checkGolden(t, "gate_merged.golden", buf3.Bytes())

	// Closed (unmerged) is terminal too: red CLOSED headline, breakdown shown.
	closedPR := &model.PR{
		Number: 42, Title: "Add xpath support", URL: "https://github.com/o/r/pull/42",
		State: "CLOSED", ReviewDecision: "CHANGES_REQUESTED",
	}
	var buf4 bytes.Buffer
	GateView(&buf4, gate.Evaluate(closedPR, &model.Checks{Counts: map[string]int{}}), ColorAuto)
	checkGolden(t, "gate_closed.golden", buf4.Bytes())
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
	ChecksView(&buf, 42, ck, ColorAuto)
	checkGolden(t, "checks.golden", buf.Bytes())
}

func TestChecksColor(t *testing.T) {
	ck := &model.Checks{
		Counts: map[string]int{"pass": 1, "fail": 1}, Total: 2,
		Failing: []model.Check{{Name: "lint", Bucket: "fail"}},
	}
	var on, off bytes.Buffer
	ChecksView(&on, 1, ck, ColorAlways)
	ChecksView(&off, 1, ck, ColorNever)
	if !strings.Contains(on.String(), "\x1b[") {
		t.Error("ColorAlways should emit ANSI")
	}
	if strings.Contains(off.String(), "\x1b[") {
		t.Error("ColorNever should not emit ANSI")
	}
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
