// Package gate evaluates whether a PR is ready to merge. GitHub is the boundary:
// the verdict anchors on GitHub's own merge-button state (mergeStateStatus),
// which already accounts for branch protection, required reviews, and *required*
// checks. ghx's value is inside that boundary — surfacing the finer signals
// (review decision, unresolved threads, CI checks) in a way that maps to how
// people actually work. Some of those signals are firm gates GitHub confirms
// (blockers); others are squishy and related but not boundary-setting — notably
// unresolved threads, which a human may have simply forgotten to resolve. Those
// are surfaced as *advisory*: always shown, never changing the verdict, left for
// a smarter consumer (a human, or an automated agent) to interpret.
package gate

import (
	"fmt"
	"strings"

	"github.com/tone-labs/ghx/internal/model"
)

// Verdict is the headline state of a PR with respect to merging.
const (
	VerdictMergeable = "MERGEABLE" // open and nothing blocks it
	VerdictBlocked   = "BLOCKED"   // open but held up (see Blockers)
	VerdictMerged    = "MERGED"    // already merged — terminal
	VerdictClosed    = "CLOSED"    // closed without merging — terminal
)

// Signal is a per-dimension display state in the breakdown. It is independent of
// the headline verdict: a dimension can be Note ("present and related, but not
// blocking") under a MERGEABLE verdict — that's the whole point.
const (
	SignalClear = "clear" // ✓ nothing to flag
	SignalBlock = "block" // ✗ a firm gate GitHub confirms
	SignalNote  = "note"  // ○ advisory: surfaced, but not boundary-setting
)

// Result is the mergeability verdict for a PR. The json-tagged fields are the
// `ghx gate --json` contract; the per-dimension signal flags (json:"-") let the
// rendered breakdown read the same facts the verdict does, not a re-derivation.
type Result struct {
	Number           int      `json:"number"`
	Title            string   `json:"title"`
	URL              string   `json:"url"`
	State            string   `json:"state"`            // OPEN | CLOSED | MERGED
	Verdict          string   `json:"verdict"`          // see Verdict* constants
	Mergeable        bool     `json:"mergeable"`        // true only when Verdict == MERGEABLE
	MergeStateStatus string   `json:"mergeStateStatus"` // GitHub's merge-button state — the anchor
	Blockers         []string `json:"blockers"`         // firm gates (drive the verdict); empty when mergeable/terminal
	Advisory         []string `json:"advisory"`         // related-but-not-blocking signals, for a consumer to weigh
	Decision         string   `json:"decision"`         // raw review decision
	Unresolved       int      `json:"unresolved"`       // unresolved review threads
	Failing          int      `json:"failingChecks"`    // checks in the fail/cancel buckets
	Pending          int      `json:"pendingChecks"`    // checks still running
	IsDraft          bool     `json:"isDraft"`
	Conflict         bool     `json:"conflict"` // merge conflict (DIRTY / CONFLICTING)
	Behind           bool     `json:"behind"`   // head branch behind base (BEHIND)

	ReviewState  string `json:"-"`
	ThreadsState string `json:"-"`
	ChecksState  string `json:"-"`
}

// Evaluate computes the verdict from a fetched PR and its checks. Merged and
// closed PRs are terminal. For an open PR, GitHub's mergeStateStatus is
// authoritative for the verdict; the finer signals explain it. Unresolved
// threads are always advisory (never a blocker), because ghx can't know whether
// an open thread is the reason an approval is pending, a settled note someone
// forgot to resolve, or a long tangent that never mattered — that judgment
// belongs to the consumer. When mergeStateStatus is UNKNOWN, the verdict falls
// back to a heuristic union of the firm signals.
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

	// Path-independent dimensions, always surfaced. Review decision is a firm
	// gate GitHub confirms; unresolved threads are advisory, never boundary.
	if reviewOK(pr.ReviewDecision) {
		r.ReviewState = SignalClear
	} else {
		r.ReviewState = SignalBlock
	}
	if unresolved > 0 {
		r.ThreadsState = SignalNote
		r.Advisory = append(r.Advisory, fmt.Sprintf("%s unresolved", plural(unresolved, "thread")))
	} else {
		r.ThreadsState = SignalClear
	}
	// Provisional checks state (used as-is for terminal PRs; the open-PR branches
	// below refine it using GitHub's required-vs-non-required signal).
	r.ChecksState = checksProvisional(failing, pending)

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
		// Mergeable per GitHub. UNSTABLE = checks are red but non-required, so they
		// don't block — surface them as advisory, not as a failed gate.
		r.Verdict = VerdictMergeable
		r.Mergeable = true
		if failing > 0 || pending > 0 {
			r.ChecksState = SignalNote
			r.Advisory = append(r.Advisory, checksDescription(failing, pending)+" (not required)")
		} else {
			r.ChecksState = SignalClear
		}
	case "BLOCKED", "BEHIND", "DIRTY", "DRAFT":
		r.Verdict = VerdictBlocked
		r.Blockers = firmBlockers(pr, &r, failing, pending)
	default: // UNKNOWN / "" — GitHub wouldn't say; degrade to the heuristic union.
		r.Blockers = heuristicBlockers(pr, &r, failing, pending)
		r.Mergeable = len(r.Blockers) == 0
		if r.Mergeable {
			r.Verdict = VerdictMergeable
		} else {
			r.Verdict = VerdictBlocked
		}
	}
	return r
}

// firmBlockers explains a BLOCKED/BEHIND/DIRTY/DRAFT status with the firm gates,
// setting their signal states. Unresolved threads are intentionally absent — they
// stay advisory even here. The catch-all keeps us honest when GitHub blocks for a
// rule we can't see locally (e.g. a required-conversation-resolution setting),
// where the advisory list often hints at the cause.
func firmBlockers(pr *model.PR, r *Result, failing, pending int) []string {
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
	if r.ReviewState == SignalBlock {
		b = append(b, reviewBlocker(pr.ReviewDecision))
	}
	// Under a BLOCKED status, red checks are (probably) required, so they are a
	// firm gate here — unlike UNSTABLE, where the same reds are advisory.
	if failing > 0 || pending > 0 {
		r.ChecksState = SignalBlock
		b = append(b, checkBlockers(failing, pending)...)
	} else {
		r.ChecksState = SignalClear
	}
	if len(b) == 0 {
		b = append(b, "blocked by branch protection")
	}
	return b
}

// heuristicBlockers is the fallback union when GitHub hasn't computed a merge
// state. Threads stay advisory; without GitHub's required-vs-non-required signal
// we conservatively treat red checks as firm gates (the pre-anchor behavior).
func heuristicBlockers(pr *model.PR, r *Result, failing, pending int) []string {
	var b []string
	if pr.IsDraft {
		r.IsDraft = true
		b = append(b, "PR is a draft")
	}
	if pr.Mergeable == "CONFLICTING" {
		r.Conflict = true
		b = append(b, "merge conflict")
	}
	if r.ReviewState == SignalBlock {
		b = append(b, reviewBlocker(pr.ReviewDecision))
	}
	if failing > 0 || pending > 0 {
		r.ChecksState = SignalBlock
		b = append(b, checkBlockers(failing, pending)...)
	} else {
		r.ChecksState = SignalClear
	}
	return b
}

func checkBlockers(failing, pending int) []string {
	var b []string
	if failing > 0 {
		b = append(b, fmt.Sprintf("%s failing", plural(failing, "check")))
	}
	if pending > 0 {
		b = append(b, fmt.Sprintf("%s still running", plural(pending, "check")))
	}
	return b
}

// checksDescription is the short noun phrase for red checks ("1 check failing, 2
// checks running"); checksProvisional is the pre-anchor signal state.
func checksDescription(failing, pending int) string {
	var parts []string
	if failing > 0 {
		parts = append(parts, plural(failing, "check")+" failing")
	}
	if pending > 0 {
		parts = append(parts, plural(pending, "check")+" running")
	}
	if len(parts) == 0 {
		return "all passing"
	}
	return strings.Join(parts, ", ")
}

func checksProvisional(failing, pending int) string {
	if failing > 0 || pending > 0 {
		return SignalNote
	}
	return SignalClear
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
