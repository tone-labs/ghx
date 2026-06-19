// Package gate evaluates whether a PR is ready to merge by unioning its state,
// review decision, unresolved review threads, and CI checks into one verdict.
package gate

import (
	"fmt"

	"github.com/tone-labs/ghx/internal/model"
)

// Verdict is the headline state of a PR with respect to merging.
const (
	VerdictMergeable = "MERGEABLE" // open and nothing blocks it
	VerdictBlocked   = "BLOCKED"   // open but held up (see Blockers)
	VerdictMerged    = "MERGED"    // already merged — terminal
	VerdictClosed    = "CLOSED"    // closed without merging — terminal
)

// Result is the mergeability verdict for a PR. The json-tagged fields are the
// `ghx gate --json` contract; the per-dimension ok flags (json:"-") exist so the
// rendered breakdown reads the same facts the verdict does, not a re-derivation.
type Result struct {
	Number     int      `json:"number"`
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	State      string   `json:"state"`         // OPEN | CLOSED | MERGED
	Verdict    string   `json:"verdict"`       // see Verdict* constants
	Mergeable  bool     `json:"mergeable"`     // true only when Verdict == MERGEABLE
	Blockers   []string `json:"blockers"`      // open-PR reasons; empty when mergeable or terminal
	Decision   string   `json:"decision"`      // raw review decision
	Unresolved int      `json:"unresolved"`    // unresolved review threads
	Failing    int      `json:"failingChecks"` // checks in the fail/cancel buckets
	Pending    int      `json:"pendingChecks"` // checks still running
	IsDraft    bool     `json:"isDraft"`

	ReviewOK  bool `json:"-"`
	ThreadsOK bool `json:"-"`
	ChecksOK  bool `json:"-"`
}

// Evaluate computes the verdict from a fetched PR and its checks. Merged and
// closed PRs are terminal — their state is the verdict, with no merge gate to
// compute. An open PR is mergeable only when nothing blocks it: not a draft,
// review not requesting changes or still required, no unresolved threads, and
// all checks complete and passing. Threads are read raw (not filtered) so the
// unresolved count is true. The per-dimension ok flags are always set, so the
// breakdown is informative even for terminal PRs.
func Evaluate(pr *model.PR, ck *model.Checks) Result {
	unresolved := 0
	for _, t := range pr.Threads {
		if !t.IsResolved {
			unresolved++
		}
	}
	failing := len(ck.Failing)
	pending := ck.Counts["pending"]

	r := Result{
		Number:     pr.Number,
		Title:      pr.Title,
		URL:        pr.URL,
		State:      pr.State,
		Decision:   pr.ReviewDecision,
		Unresolved: unresolved,
		Failing:    failing,
		Pending:    pending,
		IsDraft:    pr.IsDraft,
		ReviewOK:   pr.ReviewDecision != "CHANGES_REQUESTED" && pr.ReviewDecision != "REVIEW_REQUIRED",
		ThreadsOK:  unresolved == 0,
		ChecksOK:   failing == 0 && pending == 0,
	}

	switch pr.State {
	case "MERGED":
		r.Verdict = VerdictMerged
		return r
	case "CLOSED":
		r.Verdict = VerdictClosed
		return r
	}

	// Open (or unknown/empty state, treated as open): compute what blocks merge.
	var blockers []string
	if pr.IsDraft {
		blockers = append(blockers, "PR is a draft")
	}
	if !r.ReviewOK {
		if pr.ReviewDecision == "CHANGES_REQUESTED" {
			blockers = append(blockers, "changes requested")
		} else {
			blockers = append(blockers, "review required")
		}
	}
	if !r.ThreadsOK {
		blockers = append(blockers, fmt.Sprintf("%s unresolved", plural(unresolved, "thread")))
	}
	if failing > 0 {
		blockers = append(blockers, fmt.Sprintf("%s failing", plural(failing, "check")))
	}
	if pending > 0 {
		blockers = append(blockers, fmt.Sprintf("%s still running", plural(pending, "check")))
	}

	r.Blockers = blockers
	r.Mergeable = len(blockers) == 0
	if r.Mergeable {
		r.Verdict = VerdictMergeable
	} else {
		r.Verdict = VerdictBlocked
	}
	return r
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
