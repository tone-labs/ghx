package gate

import (
	"testing"

	"github.com/tone-labs/ghx/internal/model"
)

// pr builds a PR with the given decision, draft flag, and one thread per
// `threads` entry (true = resolved).
func pr(decision string, draft bool, threads ...bool) *model.PR {
	p := &model.PR{Number: 1, Title: "t", ReviewDecision: decision, IsDraft: draft}
	for _, resolved := range threads {
		p.Threads = append(p.Threads, model.Thread{IsResolved: resolved})
	}
	return p
}

func checks(failing, pending int) *model.Checks {
	ck := &model.Checks{Counts: map[string]int{}}
	for range failing {
		ck.Failing = append(ck.Failing, model.Check{Bucket: "fail"})
	}
	if pending > 0 {
		ck.Counts["pending"] = pending
	}
	return ck
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name      string
		pr        *model.PR
		ck        *model.Checks
		mergeable bool
		blockers  int
	}{
		{"approved + resolved + passing", pr("APPROVED", false, true, true), checks(0, 0), true, 0},
		{"no decision required, clean", pr("", false), checks(0, 0), true, 0},
		{"changes requested", pr("CHANGES_REQUESTED", false), checks(0, 0), false, 1},
		{"review required", pr("REVIEW_REQUIRED", false), checks(0, 0), false, 1},
		{"unresolved thread", pr("APPROVED", false, false, true), checks(0, 0), false, 1},
		{"failing checks", pr("APPROVED", false), checks(2, 0), false, 1},
		{"pending checks", pr("APPROVED", false), checks(0, 3), false, 1},
		{"draft", pr("APPROVED", true), checks(0, 0), false, 1},
		{"everything blocks", pr("CHANGES_REQUESTED", true, false), checks(1, 1), false, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(tt.pr, tt.ck)
			if got.Mergeable != tt.mergeable {
				t.Errorf("Mergeable = %v, want %v (blockers: %v)", got.Mergeable, tt.mergeable, got.Blockers)
			}
			if len(got.Blockers) != tt.blockers {
				t.Errorf("blockers = %d %v, want %d", len(got.Blockers), got.Blockers, tt.blockers)
			}
		})
	}
}

func TestEvaluateCounts(t *testing.T) {
	got := Evaluate(pr("APPROVED", false, false, false, true), checks(2, 3))
	if got.Unresolved != 2 {
		t.Errorf("Unresolved = %d, want 2", got.Unresolved)
	}
	if got.Failing != 2 {
		t.Errorf("Failing = %d, want 2", got.Failing)
	}
	if got.Pending != 3 {
		t.Errorf("Pending = %d, want 3", got.Pending)
	}
}
