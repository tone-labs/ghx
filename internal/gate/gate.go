// Package gate evaluates whether a PR is ready to merge by unioning the review
// decision, unresolved review threads, and CI checks into one verdict.
package gate

import (
	"fmt"

	"github.com/tone-labs/ghx/internal/model"
)

// Result is the mergeability verdict for a PR. It is the JSON contract for
// `ghx gate --json`.
type Result struct {
	Number     int      `json:"number"`
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	Mergeable  bool     `json:"mergeable"`
	Blockers   []string `json:"blockers"`      // human-readable reasons; empty when mergeable
	Decision   string   `json:"decision"`      // raw review decision
	Unresolved int      `json:"unresolved"`    // unresolved review threads
	Failing    int      `json:"failingChecks"` // checks in the fail/cancel buckets
	Pending    int      `json:"pendingChecks"` // checks still running
	IsDraft    bool     `json:"isDraft"`
}

// Evaluate computes the verdict from a fetched PR and its checks. A PR is
// mergeable only when nothing blocks it: not a draft, review not requesting
// changes or still required, no unresolved threads, and all checks complete and
// passing. Threads are read raw (not filtered) so the unresolved count is true.
func Evaluate(pr *model.PR, ck *model.Checks) Result {
	unresolved := 0
	for _, t := range pr.Threads {
		if !t.IsResolved {
			unresolved++
		}
	}
	failing := len(ck.Failing)
	pending := ck.Counts["pending"]

	var blockers []string
	if pr.IsDraft {
		blockers = append(blockers, "PR is a draft")
	}
	switch pr.ReviewDecision {
	case "CHANGES_REQUESTED":
		blockers = append(blockers, "changes requested")
	case "REVIEW_REQUIRED":
		blockers = append(blockers, "review required")
	}
	if unresolved > 0 {
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
		Mergeable:  len(blockers) == 0,
		Blockers:   blockers,
		Decision:   pr.ReviewDecision,
		Unresolved: unresolved,
		Failing:    failing,
		Pending:    pending,
		IsDraft:    pr.IsDraft,
	}
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
