// Package gate evaluates whether a PR is ready to merge by unioning its open
// state, review decision, unresolved review threads, and CI checks into one
// verdict.
package gate

import (
	"fmt"
	"strings"

	"github.com/tone-labs/ghx/internal/model"
)

// Result is the mergeability verdict for a PR. The json-tagged fields are the
// `ghx gate --json` contract; the per-dimension ok flags (json:"-") exist so the
// rendered breakdown reads the same verdict the headline does, rather than
// re-deriving the blocking predicate and risking drift.
type Result struct {
	Number     int      `json:"number"`
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	State      string   `json:"state"` // OPEN | CLOSED | MERGED
	Mergeable  bool     `json:"mergeable"`
	Blockers   []string `json:"blockers"`      // human-readable reasons; empty when mergeable
	Decision   string   `json:"decision"`      // raw review decision
	Unresolved int      `json:"unresolved"`    // unresolved review threads
	Failing    int      `json:"failingChecks"` // checks in the fail/cancel buckets
	Pending    int      `json:"pendingChecks"` // checks still running
	IsDraft    bool     `json:"isDraft"`

	OpenOK    bool `json:"-"`
	ReviewOK  bool `json:"-"`
	ThreadsOK bool `json:"-"`
	ChecksOK  bool `json:"-"`
}

// Evaluate computes the verdict from a fetched PR and its checks. A PR is
// mergeable only when nothing blocks it: open (not closed/merged), not a draft,
// review not requesting changes or still required, no unresolved threads, and
// all checks complete and passing. Threads are read raw (not filtered) so the
// unresolved count is true. An empty State is treated as open (benefit of the
// doubt — the current-branch PR is always open).
func Evaluate(pr *model.PR, ck *model.Checks) Result {
	unresolved := 0
	for _, t := range pr.Threads {
		if !t.IsResolved {
			unresolved++
		}
	}
	failing := len(ck.Failing)
	pending := ck.Counts["pending"]

	openOK := pr.State == "" || pr.State == "OPEN"
	reviewOK := pr.ReviewDecision != "CHANGES_REQUESTED" && pr.ReviewDecision != "REVIEW_REQUIRED"
	threadsOK := unresolved == 0
	checksOK := failing == 0 && pending == 0

	var blockers []string
	if !openOK {
		blockers = append(blockers, "PR is "+strings.ToLower(pr.State))
	}
	if pr.IsDraft {
		blockers = append(blockers, "PR is a draft")
	}
	if !reviewOK {
		if pr.ReviewDecision == "CHANGES_REQUESTED" {
			blockers = append(blockers, "changes requested")
		} else {
			blockers = append(blockers, "review required")
		}
	}
	if !threadsOK {
		blockers = append(blockers, fmt.Sprintf("%s unresolved", plural(unresolved, "thread")))
	}
	if failing > 0 {
		blockers = append(blockers, fmt.Sprintf("%s failing", plural(failing, "check")))
	}
	if pending > 0 {
		blockers = append(blockers, fmt.Sprintf("%s still running", plural(pending, "check")))
	}

	return Result{
		Number:     pr.Number,
		Title:      pr.Title,
		URL:        pr.URL,
		State:      pr.State,
		Mergeable:  len(blockers) == 0,
		Blockers:   blockers,
		Decision:   pr.ReviewDecision,
		Unresolved: unresolved,
		Failing:    failing,
		Pending:    pending,
		IsDraft:    pr.IsDraft,
		OpenOK:     openOK,
		ReviewOK:   reviewOK,
		ThreadsOK:  threadsOK,
		ChecksOK:   checksOK,
	}
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
