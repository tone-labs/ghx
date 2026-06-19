// Package model holds the normalized, provider-agnostic representation of a
// pull request's review state. Data sources (the GraphQL provider today, a
// different one tomorrow) map into these types; render and any future
// consumer read only from here. This is the seam that keeps a provider swap
// from rippling through the tool.
package model

import "strings"

// PR is the full review-state snapshot of one pull request.
type PR struct {
	Number         int    `json:"number"`
	Title          string `json:"title"`
	URL            string `json:"url"`
	State          string `json:"state"` // OPEN | CLOSED | MERGED
	IsDraft        bool   `json:"isDraft"`
	Author         string `json:"author"`
	ReviewDecision string `json:"reviewDecision"` // APPROVED | CHANGES_REQUESTED | REVIEW_REQUIRED | "" (none)
	// Mergeable and MergeStateStatus are GitHub's own merge-button signals — the
	// authoritative answer that already accounts for branch protection, required
	// reviews, and *required* checks. The gate anchors its verdict on them rather
	// than re-deriving mergeability from the finer signals below.
	Mergeable        string    `json:"mergeable"`        // MERGEABLE | CONFLICTING | UNKNOWN
	MergeStateStatus string    `json:"mergeStateStatus"` // CLEAN | HAS_HOOKS | UNSTABLE | BLOCKED | BEHIND | DIRTY | DRAFT | UNKNOWN
	Reviews          []Review  `json:"reviews"`
	Threads          []Thread  `json:"threads"`
	Conversation     []Comment `json:"conversation"`
}

// Review is a submitted PR review (the approval-gate signal).
type Review struct {
	Author      string `json:"author"`
	IsBot       bool   `json:"isBot"`
	State       string `json:"state"` // APPROVED | CHANGES_REQUESTED | COMMENTED | DISMISSED | PENDING
	Body        string `json:"body"`
	SubmittedAt string `json:"submittedAt"`
}

// Thread is an inline review thread with its resolution state — the piece the
// REST-based ghprc could never surface.
type Thread struct {
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	Line       int       `json:"line"`
	IsResolved bool      `json:"isResolved"`
	IsOutdated bool      `json:"isOutdated"`
	Comments   []Comment `json:"comments"`
}

// Comment is one inline reply or one PR-level conversation comment.
type Comment struct {
	Author    string `json:"author"`
	IsBot     bool   `json:"isBot"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	URL       string `json:"url"`
}

// Checks is the CI status-check rollup for a PR: bucket counts plus the
// detail of any failing/cancelled checks.
type Checks struct {
	Counts  map[string]int `json:"counts"` // bucket (pass|fail|pending|skipping|cancel) -> count
	Failing []Check        `json:"failing"`
	Total   int            `json:"total"`
}

// Check is a single status check.
type Check struct {
	Name     string `json:"name"`
	Bucket   string `json:"bucket"`
	State    string `json:"state"`
	Workflow string `json:"workflow"`
	Link     string `json:"link"`
}

// IsBotActor applies the single bot-classification rule used everywhere:
// GitHub's GraphQL Actor __typename of "Bot", with a login "[bot]" suffix
// fallback for cases the typename doesn't cover.
func IsBotActor(login, typename string) bool {
	return typename == "Bot" || strings.HasSuffix(login, "[bot]")
}

// Starter returns the thread's first (root) comment, or nil if empty.
func (t Thread) Starter() *Comment {
	if len(t.Comments) == 0 {
		return nil
	}
	return &t.Comments[0]
}
