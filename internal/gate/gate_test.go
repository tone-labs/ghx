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
		name     string
		pr       *model.PR
		ck       *model.Checks
		verdict  string
		blockers int
	}{
		{"approved + resolved + passing", pr("APPROVED", false, true, true), checks(0, 0), VerdictMergeable, 0},
		{"no decision required, clean", pr("", false), checks(0, 0), VerdictMergeable, 0},
		{"changes requested", pr("CHANGES_REQUESTED", false), checks(0, 0), VerdictBlocked, 1},
		{"review required", pr("REVIEW_REQUIRED", false), checks(0, 0), VerdictBlocked, 1},
		{"unresolved thread", pr("APPROVED", false, false, true), checks(0, 0), VerdictBlocked, 1},
		{"failing checks", pr("APPROVED", false), checks(2, 0), VerdictBlocked, 1},
		{"pending checks", pr("APPROVED", false), checks(0, 3), VerdictBlocked, 1},
		{"draft", pr("APPROVED", true), checks(0, 0), VerdictBlocked, 1},
		// Terminal states short-circuit: the state IS the verdict, no blockers,
		// not mergeable — even when review/threads/checks would otherwise pass.
		{"merged is terminal", &model.PR{State: "MERGED", ReviewDecision: "APPROVED"}, checks(0, 0), VerdictMerged, 0},
		{"closed is terminal", &model.PR{State: "CLOSED"}, checks(0, 0), VerdictClosed, 0},
		{"everything blocks", pr("CHANGES_REQUESTED", true, false), checks(1, 1), VerdictBlocked, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(tt.pr, tt.ck)
			if got.Verdict != tt.verdict {
				t.Errorf("Verdict = %q, want %q (blockers: %v)", got.Verdict, tt.verdict, got.Blockers)
			}
			if want := tt.verdict == VerdictMergeable; got.Mergeable != want {
				t.Errorf("Mergeable = %v, want %v", got.Mergeable, want)
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
