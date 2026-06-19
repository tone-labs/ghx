package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/tone-labs/ghx/internal/gate"
)

// GateView renders the mergeability verdict: a headline (MERGEABLE / BLOCKED)
// followed by a per-dimension breakdown of review, threads, and checks.
func GateView(w io.Writer, r gate.Result, color ColorMode) {
	s := newStyles(w, color)

	fmt.Fprintln(w, s.bold.Render(fmt.Sprintf("#%d  %s", r.Number, r.Title)))
	if r.URL != "" {
		fmt.Fprintln(w, s.faint.Render(r.URL))
	}
	if r.Mergeable {
		fmt.Fprintln(w, s.green.Render("✓ MERGEABLE"))
	} else {
		fmt.Fprintln(w, s.red.Render("✗ BLOCKED")+s.faint.Render("  ·  "+plural(len(r.Blockers), "blocker")))
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
	if !r.OpenOK {
		row(false, "state", strings.ToLower(r.State))
	}
	if r.IsDraft {
		row(false, "draft", "marked draft")
	}
	row(r.ReviewOK, "review", decisionDetail(r.Decision))
	row(r.ThreadsOK, "threads", threadsDetail(r.Unresolved))
	row(r.ChecksOK, "checks", checksDetail(r.Failing, r.Pending))
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

func checksDetail(failing, pending int) string {
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
	return strings.Join(parts, ", ")
}
