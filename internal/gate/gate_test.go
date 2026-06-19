package gate

import (
	"slices"
	"strings"
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

// TestEvaluateAnchorsOnMergeStateStatus covers the GitHub-anchored path: when a
// merge state is present, it drives the verdict (so we agree with the merge
// button), and the finer signals explain it.
func TestEvaluateAnchorsOnMergeStateStatus(t *testing.T) {
	cases := []struct {
		name        string
		pr          *model.PR
		ck          *model.Checks
		wantVerdict string
		wantMerge   bool
		wantBlocker string // substring required among blockers ("" = none)
	}{
		{"clean is mergeable", &model.PR{State: "OPEN", MergeStateStatus: "CLEAN", ReviewDecision: "APPROVED"}, checks(0, 0), VerdictMergeable, true, ""},
		{"has_hooks is mergeable", &model.PR{State: "OPEN", MergeStateStatus: "HAS_HOOKS"}, checks(0, 0), VerdictMergeable, true, ""},
		// The core fix: red but non-required checks don't block merge.
		{"unstable is mergeable despite red checks", &model.PR{State: "OPEN", MergeStateStatus: "UNSTABLE"}, checks(2, 1), VerdictMergeable, true, ""},
		{"blocked with changes requested", &model.PR{State: "OPEN", MergeStateStatus: "BLOCKED", ReviewDecision: "CHANGES_REQUESTED"}, checks(0, 0), VerdictBlocked, false, "changes requested"},
		// GitHub blocks for a rule we don't model — honest catch-all.
		{"blocked with no local signal", &model.PR{State: "OPEN", MergeStateStatus: "BLOCKED", ReviewDecision: "APPROVED"}, checks(0, 0), VerdictBlocked, false, "branch protection"},
		{"behind surfaces out-of-date branch", &model.PR{State: "OPEN", MergeStateStatus: "BEHIND", ReviewDecision: "APPROVED"}, checks(0, 0), VerdictBlocked, false, "out of date"},
		{"dirty surfaces merge conflict", &model.PR{State: "OPEN", MergeStateStatus: "DIRTY", Mergeable: "CONFLICTING", ReviewDecision: "APPROVED"}, checks(0, 0), VerdictBlocked, false, "merge conflict"},
		{"draft status blocks", &model.PR{State: "OPEN", MergeStateStatus: "DRAFT"}, checks(0, 0), VerdictBlocked, false, "draft"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := Evaluate(tc.pr, tc.ck)
			if r.Verdict != tc.wantVerdict {
				t.Errorf("verdict = %q, want %q (blockers %v)", r.Verdict, tc.wantVerdict, r.Blockers)
			}
			if r.Mergeable != tc.wantMerge {
				t.Errorf("mergeable = %v, want %v", r.Mergeable, tc.wantMerge)
			}
			if tc.wantBlocker != "" && !slices.ContainsFunc(r.Blockers, func(s string) bool { return strings.Contains(s, tc.wantBlocker) }) {
				t.Errorf("blockers %v missing %q", r.Blockers, tc.wantBlocker)
			}
		})
	}
}

// TestEvaluateUnstableChecksOK guards breakdown/headline consistency: under
// UNSTABLE the checks dimension must read non-blocking even though checks are
// red, so the ✓ checks row doesn't contradict the MERGEABLE headline.
func TestEvaluateUnstableChecksOK(t *testing.T) {
	r := Evaluate(&model.PR{State: "OPEN", MergeStateStatus: "UNSTABLE"}, checks(1, 0))
	if !r.ChecksOK {
		t.Error("ChecksOK = false under UNSTABLE, want true (reds are non-required)")
	}
	if len(r.Blockers) != 0 {
		t.Errorf("blockers = %v, want none under UNSTABLE", r.Blockers)
	}
}

// TestEvaluateUnknownFallsBackToHeuristic verifies that when GitHub hasn't
// computed a merge state, the verdict degrades to the finer-signal union — and
// still blocks on a failing check (the pre-anchor behavior).
func TestEvaluateUnknownFallsBackToHeuristic(t *testing.T) {
	clean := Evaluate(&model.PR{State: "OPEN", MergeStateStatus: "UNKNOWN", ReviewDecision: "APPROVED"}, checks(0, 0))
	if clean.Verdict != VerdictMergeable {
		t.Errorf("UNKNOWN + clean: verdict = %q, want MERGEABLE", clean.Verdict)
	}
	failing := Evaluate(&model.PR{State: "OPEN", MergeStateStatus: "UNKNOWN"}, checks(1, 0))
	if failing.Verdict != VerdictBlocked {
		t.Errorf("UNKNOWN + failing: verdict = %q, want BLOCKED", failing.Verdict)
	}
	conflict := Evaluate(&model.PR{State: "OPEN", MergeStateStatus: "UNKNOWN", Mergeable: "CONFLICTING"}, checks(0, 0))
	if !conflict.Conflict || conflict.Verdict != VerdictBlocked {
		t.Errorf("UNKNOWN + CONFLICTING: conflict=%v verdict=%q, want true/BLOCKED", conflict.Conflict, conflict.Verdict)
	}
}
