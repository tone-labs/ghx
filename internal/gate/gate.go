// Package gate evaluates whether a PR is ready to merge. It anchors the verdict
// on GitHub's own merge-button state (mergeStateStatus) — which already accounts
// for branch protection, required reviews, and *required* checks — and uses the
// finer signals (review decision, unresolved threads, CI checks) to explain the
// verdict rather than to re-derive it. When GitHub hasn't computed a state, it
// falls back to a best-effort heuristic union of those finer signals.
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
	Number           int      `json:"number"`
	Title            string   `json:"title"`
	URL              string   `json:"url"`
	State            string   `json:"state"`            // OPEN | CLOSED | MERGED
	Verdict          string   `json:"verdict"`          // see Verdict* constants
	Mergeable        bool     `json:"mergeable"`        // true only when Verdict == MERGEABLE
	MergeStateStatus string   `json:"mergeStateStatus"` // GitHub's merge-button state — the anchor
	Blockers         []string `json:"blockers"`         // open-PR reasons; empty when mergeable or terminal
	Decision         string   `json:"decision"`         // raw review decision
	Unresolved       int      `json:"unresolved"`       // unresolved review threads
	Failing          int      `json:"failingChecks"`    // checks in the fail/cancel buckets
	Pending          int      `json:"pendingChecks"`    // checks still running
	IsDraft          bool     `json:"isDraft"`
	Conflict         bool     `json:"conflict"` // merge conflict (DIRTY / CONFLICTING)
	Behind           bool     `json:"behind"`   // head branch behind base (BEHIND)

	ReviewOK  bool `json:"-"`
	ThreadsOK bool `json:"-"`
	ChecksOK  bool `json:"-"`
}

// Evaluate computes the verdict from a fetched PR and its checks. Merged and
// closed PRs are terminal — their state is the verdict. For an open PR, GitHub's
// mergeStateStatus is authoritative: CLEAN/HAS_HOOKS/UNSTABLE mean it will merge
// (UNSTABLE = non-required checks are red but the PR is still mergeable — the
// case a naive union gets wrong); BLOCKED/BEHIND/DIRTY/DRAFT mean it won't, and
// the finer signals explain why. When mergeStateStatus is UNKNOWN (GitHub hasn't
// computed it), fall back to a heuristic union of the finer signals.
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
		Number:           pr.Number,
		Title:            pr.Title,
		URL:              pr.URL,
		State:            pr.State,
		Decision:         pr.ReviewDecision,
		MergeStateStatus: pr.MergeStateStatus,
		Unresolved:       unresolved,
		Failing:          failing,
		Pending:          pending,
		IsDraft:          pr.IsDraft,
	}
	// Natural per-dimension flags: informative for terminal states and the
	// heuristic fallback. The open-PR branches override them to stay consistent
	// with the GitHub-anchored verdict.
	r.ReviewOK = reviewOK(pr.ReviewDecision)
	r.ThreadsOK = unresolved == 0
	r.ChecksOK = failing == 0 && pending == 0

	switch pr.State {
	case "MERGED":
		r.Verdict = VerdictMerged
		return r
	case "CLOSED":
		r.Verdict = VerdictClosed
		return r
	}

	// Open: anchor on GitHub's merge-button state.
	switch pr.MergeStateStatus {
	case "CLEAN", "HAS_HOOKS", "UNSTABLE":
		r.Verdict = VerdictMergeable
		r.Mergeable = true
		r.ReviewOK, r.ThreadsOK, r.ChecksOK = true, true, true
	case "BLOCKED", "BEHIND", "DIRTY", "DRAFT":
		r.Verdict = VerdictBlocked
		r.Blockers = openBlockers(pr, &r, unresolved, failing, pending)
	default: // UNKNOWN / "" — GitHub wouldn't say; degrade to the heuristic union.
		r.Blockers = heuristicBlockers(pr, &r, unresolved, failing, pending)
		r.Mergeable = len(r.Blockers) == 0
		if r.Mergeable {
			r.Verdict = VerdictMergeable
		} else {
			r.Verdict = VerdictBlocked
		}
	}
	return r
}

// openBlockers explains a BLOCKED/BEHIND/DIRTY/DRAFT status using the finer
// signals, setting the matching breakdown flags as it goes. The catch-all keeps
// us honest when GitHub blocks for a rule we don't model (e.g. a branch-
// protection requirement with no local signal).
func openBlockers(pr *model.PR, r *Result, unresolved, failing, pending int) []string {
	var b []string
	if pr.IsDraft || pr.MergeStateStatus == "DRAFT" {
		r.IsDraft = true
		b = append(b, "PR is a draft")
	}
	if pr.MergeStateStatus == "DIRTY" || pr.Mergeable == "CONFLICTING" {
		r.Conflict = true
		b = append(b, "merge conflict")
	}
	if pr.MergeStateStatus == "BEHIND" {
		r.Behind = true
		b = append(b, "branch out of date with base")
	}
	if !reviewOK(pr.ReviewDecision) {
		r.ReviewOK = false
		b = append(b, reviewBlocker(pr.ReviewDecision))
	}
	if unresolved > 0 {
		r.ThreadsOK = false
		b = append(b, fmt.Sprintf("%s unresolved", plural(unresolved, "thread")))
	}
	if failing > 0 {
		r.ChecksOK = false
		b = append(b, fmt.Sprintf("%s failing", plural(failing, "check")))
	}
	if pending > 0 {
		r.ChecksOK = false
		b = append(b, fmt.Sprintf("%s still running", plural(pending, "check")))
	}
	if len(b) == 0 {
		b = append(b, "blocked by branch protection")
	}
	return b
}

// heuristicBlockers is the fallback union when GitHub hasn't computed a merge
// state. It's the pre-anchor behavior (plus a known-conflict signal), and may
// over-block on non-required checks — the price of GitHub not answering.
func heuristicBlockers(pr *model.PR, r *Result, unresolved, failing, pending int) []string {
	var b []string
	if pr.IsDraft {
		r.IsDraft = true
		b = append(b, "PR is a draft")
	}
	if pr.Mergeable == "CONFLICTING" {
		r.Conflict = true
		b = append(b, "merge conflict")
	}
	if !reviewOK(pr.ReviewDecision) {
		r.ReviewOK = false
		b = append(b, reviewBlocker(pr.ReviewDecision))
	}
	if unresolved > 0 {
		r.ThreadsOK = false
		b = append(b, fmt.Sprintf("%s unresolved", plural(unresolved, "thread")))
	}
	if failing > 0 {
		r.ChecksOK = false
		b = append(b, fmt.Sprintf("%s failing", plural(failing, "check")))
	}
	if pending > 0 {
		r.ChecksOK = false
		b = append(b, fmt.Sprintf("%s still running", plural(pending, "check")))
	}
	return b
}

func reviewOK(decision string) bool {
	return decision != "CHANGES_REQUESTED" && decision != "REVIEW_REQUIRED"
}

func reviewBlocker(decision string) string {
	if decision == "CHANGES_REQUESTED" {
		return "changes requested"
	}
	return "review required"
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
