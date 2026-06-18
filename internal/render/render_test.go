package render

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/cbuchan/ghx/internal/model"
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
	Comments(&buf, samplePR(), Options{Width: 40})
	checkGolden(t, "comments_default.golden", buf.Bytes())
}

func TestCommentsFullGolden(t *testing.T) {
	var buf bytes.Buffer
	Comments(&buf, samplePR(), Options{Full: true})
	checkGolden(t, "comments_full.golden", buf.Bytes())
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
