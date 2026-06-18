package render

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/cbuchan/ghx/internal/model"
)

// Options controls the human (non-JSON) comment view.
type Options struct {
	Width int  // body truncation width; <= 0 means no limit
	Full  bool // full bodies, multi-line (forces Width = 0)
}

func (o Options) width() int {
	if o.Full {
		return 0
	}
	return o.Width
}

// Comments renders the PR review state: header + gate, reviews, inline threads,
// and PR-level conversation. The PR passed in is assumed already filtered.
func Comments(w io.Writer, pr *model.PR, opts Options) {
	fmt.Fprintf(w, "PR #%d  %s  [%s]%s\n", pr.Number, pr.Title, pr.State, draftTag(pr.IsDraft))
	if pr.URL != "" {
		fmt.Fprintf(w, "%s\n", pr.URL)
	}
	fmt.Fprintf(w, "Review decision: %s\n", decisionLabel(pr.ReviewDecision))

	// Reviews — latest submitted review per author (the gate is the decision above).
	reviews := latestPerAuthor(pr.Reviews)
	if len(reviews) > 0 {
		fmt.Fprintf(w, "\nReviews:\n")
		for _, r := range reviews {
			line := fmt.Sprintf("  %-19s %s%s", "["+r.State+"]", r.Author, botTag(r.IsBot))
			if flatten(r.Body) != "" {
				text, omitted := truncate(r.Body, opts.width())
				line += ": " + text + hint(omitted, opts.Full)
			}
			fmt.Fprintln(w, line)
		}
	}

	// Inline threads.
	unresolved := 0
	for _, t := range pr.Threads {
		if !t.IsResolved {
			unresolved++
		}
	}
	fmt.Fprintf(w, "\nThreads: %d (%d unresolved)\n", len(pr.Threads), unresolved)
	for _, t := range pr.Threads {
		fmt.Fprintf(w, "  %s%s\n", location(t), threadBadges(t))
		for i, c := range t.Comments {
			renderComment(w, c, i == 0, opts)
		}
		fmt.Fprintf(w, "    thread: %s\n", t.ID)
	}

	// PR-level conversation.
	if len(pr.Conversation) > 0 {
		fmt.Fprintf(w, "\nConversation:\n")
		for _, c := range pr.Conversation {
			renderComment(w, c, true, opts)
		}
	}
}

func renderComment(w io.Writer, c model.Comment, root bool, opts Options) {
	prefix := "      ↪ "
	if root {
		prefix = "    "
	}
	author := c.Author + botTag(c.IsBot)
	if opts.Full {
		fmt.Fprintf(w, "%s%s:\n", prefix, author)
		fmt.Fprintln(w, indentBlock(strings.TrimSpace(c.Body), strings.Repeat(" ", len(prefix))))
		return
	}
	text, omitted := truncate(c.Body, opts.width())
	fmt.Fprintf(w, "%s%s: %s%s\n", prefix, author, text, hint(omitted, opts.Full))
}

func location(t model.Thread) string {
	if t.Path == "" {
		return "(general)"
	}
	if t.Line > 0 {
		return fmt.Sprintf("%s:%d", t.Path, t.Line)
	}
	return t.Path
}

func threadBadges(t model.Thread) string {
	badge := "  [open]"
	if t.IsResolved {
		badge = "  [resolved]"
	}
	if t.IsOutdated {
		badge += " [outdated]"
	}
	return badge
}

func draftTag(d bool) string {
	if d {
		return " (draft)"
	}
	return ""
}

func botTag(b bool) string {
	if b {
		return " (bot)"
	}
	return ""
}

func decisionLabel(d string) string {
	if d == "" {
		return "none (no review submitted)"
	}
	return d
}

func hint(omitted int, full bool) string {
	if omitted <= 0 || full {
		return ""
	}
	return " " + dim(fmt.Sprintf("(+%d chars — --full or --thread)", omitted))
}

// latestPerAuthor keeps each author's most recently submitted review.
func latestPerAuthor(reviews []model.Review) []model.Review {
	latest := map[string]model.Review{}
	for _, r := range reviews {
		if cur, ok := latest[r.Author]; !ok || r.SubmittedAt > cur.SubmittedAt {
			latest[r.Author] = r
		}
	}
	out := make([]model.Review, 0, len(latest))
	for _, r := range latest {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SubmittedAt < out[j].SubmittedAt })
	return out
}
