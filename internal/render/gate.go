package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/tone-labs/ghx/internal/gate"
)

// GateView renders the mergeability verdict: a headline (MERGEABLE / BLOCKED /
// MERGED / CLOSED) followed by a per-dimension breakdown of review, threads,
// and checks.
func GateView(w io.Writer, r gate.Result, color ColorMode) {
	s := newStyles(w, color)

	fmt.Fprintln(w, s.bold.Render(fmt.Sprintf("#%d  %s", r.Number, r.Title)))
	if r.URL != "" {
		fmt.Fprintln(w, s.faint.Render(r.URL))
	}
	// The headline carries the PR's terminal state: merged is purple (a success
	// state, GitHub-style — not a red blocker), closed is red, and an open PR is
	// the green/red merge verdict.
	switch r.Verdict {
	case gate.VerdictMergeable:
		fmt.Fprintln(w, s.green.Render("✓ MERGEABLE"))
	case gate.VerdictMerged:
		fmt.Fprintln(w, s.purple.Render("● MERGED"))
	case gate.VerdictClosed:
		fmt.Fprintln(w, s.red.Render("✗ CLOSED"))
	default: // BLOCKED
		fmt.Fprintln(w, s.red.Render("✗ BLOCKED")+s.faint.Render("  ·  "+plural(len(r.Blockers), "blocker")))
	}
	// Show GitHub's own merge-button state (the anchor) when it gave us one, so
	// the verdict is traceable — e.g. an UNSTABLE behind a MERGEABLE headline
	// explains the red-but-non-required checks below.
	if isOpenVerdict(r.Verdict) && r.MergeStateStatus != "" && r.MergeStateStatus != "UNKNOWN" {
		fmt.Fprintln(w, s.faint.Render("merge state: "+strings.ToLower(r.MergeStateStatus)))
	}
	fmt.Fprintln(w)

	// ok-states come from the Result (set by gate.Evaluate) so the breakdown
	// can't disagree with the headline; only the detail strings are formatted here.
	row := func(ok bool, label, detail string) {
		mark := s.green.Render("✓")
		if !ok {
			mark = s.red.Render("✗")
		}
		fmt.Fprintf(w, "  %s %-8s %s\n", mark, label, s.faint.Render(detail))
	}
	if r.IsDraft {
		row(false, "draft", "marked draft")
	}
	if r.Conflict {
		row(false, "conflict", "merge conflict with base")
	}
	if r.Behind {
		row(false, "branch", "out of date with base")
	}
	row(r.ReviewOK, "review", decisionDetail(r.Decision))
	row(r.ThreadsOK, "threads", threadsDetail(r.Unresolved))
	row(r.ChecksOK, "checks", checksDetail(r.Failing, r.Pending, r.MergeStateStatus))
}

// isOpenVerdict reports whether the verdict is for an open PR (not a terminal
// merged/closed state), where GitHub's merge-state annotation is meaningful.
func isOpenVerdict(verdict string) bool {
	return verdict == gate.VerdictMergeable || verdict == gate.VerdictBlocked
}

func decisionDetail(decision string) string {
	switch decision {
	case "APPROVED":
		return "approved"
	case "CHANGES_REQUESTED":
		return "changes requested"
	case "REVIEW_REQUIRED":
		return "review required"
	default:
		return "no decision required"
	}
}

func threadsDetail(unresolved int) string {
	if unresolved == 0 {
		return "all resolved"
	}
	return plural(unresolved, "thread") + " unresolved"
}

func checksDetail(failing, pending int, status string) string {
	if failing == 0 && pending == 0 {
		return "all passing"
	}
	var parts []string
	if failing > 0 {
		parts = append(parts, plural(failing, "check")+" failing")
	}
	if pending > 0 {
		parts = append(parts, plural(pending, "check")+" running")
	}
	detail := strings.Join(parts, ", ")
	// Under UNSTABLE the reds are real but non-required, so they don't block the
	// merge — say so, otherwise a ✓ checks row next to "1 check failing" reads as
	// a contradiction.
	if status == "UNSTABLE" {
		detail += " (not required)"
	}
	return detail
}
