package cli

import (
	"flag"
	"os"

	"github.com/tone-labs/ghx/internal/ghclient"
	"github.com/tone-labs/ghx/internal/model"
	"github.com/tone-labs/ghx/internal/provider"
	"github.com/tone-labs/ghx/internal/render"
)

func runComments(args []string) int {
	fs := flag.NewFlagSet("comments", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		repo         string
		all          = fs.Bool("all", false, "include resolved threads (default: unresolved only)")
		hideOutdated = fs.Bool("hide-outdated", false, "exclude outdated threads")
		bots         = fs.Bool("bots", false, "only bot-authored items")
		humans       = fs.Bool("humans", false, "only human-authored items")
		author       = fs.String("author", "", "only items authored by this login (overrides --bots/--humans)")
		thread       = fs.String("thread", "", "show only this thread id, in full (drill-in)")
		full         = fs.Bool("full", false, "full comment bodies, no truncation")
		width        = fs.Int("truncate", 200, "truncation width for bodies (0 = no limit)")
		jsonOut      = fs.Bool("json", false, "machine-readable JSON output")
	)
	fs.StringVar(&repo, "repo", "", "target repo as owner/repo (default: current repo)")
	fs.StringVar(&repo, "R", "", "shorthand for --repo")
	fs.Usage = func() {
		fs.Output().Write([]byte("Usage: ghx comments [PR] [flags]\n\n" +
			"Inline review threads (with resolution state), reviews + decision,\n" +
			"and PR-level conversation. Defaults to the current branch's PR and to\n" +
			"unresolved threads only.\n\nFlags:\n"))
		fs.PrintDefaults()
	}

	prArg, rest := splitPR(args)
	if err := fs.Parse(rest); err != nil {
		return 2
	}

	c, err := ghclient.New(repo)
	if err != nil {
		return fail(err)
	}
	prNum, err := ghclient.ResolvePR(prArg, repo)
	if err != nil {
		return fail(err)
	}
	pr, err := provider.FetchPR(c, prNum)
	if err != nil {
		return fail(err)
	}

	opts := commentFilter{
		all:          *all,
		hideOutdated: *hideOutdated,
		bots:         *bots,
		humans:       *humans,
		author:       *author,
		thread:       *thread,
	}
	full2 := *full
	opts.apply(pr, &full2)

	if *jsonOut {
		if err := render.JSON(os.Stdout, pr); err != nil {
			return fail(err)
		}
		return 0
	}
	render.AutoColor(os.Stdout.Fd())
	render.Comments(os.Stdout, pr, render.Options{Width: *width, Full: full2})
	return 0
}

type commentFilter struct {
	all          bool
	hideOutdated bool
	bots         bool
	humans       bool
	author       string
	thread       string
}

// apply mutates pr in place per the active filters. When a single thread is
// requested it becomes a focused, full view (full is forced on).
func (f commentFilter) apply(pr *model.PR, full *bool) {
	if f.thread != "" {
		var only []model.Thread
		for _, t := range pr.Threads {
			if t.ID == f.thread {
				only = append(only, t)
			}
		}
		pr.Threads = only
		pr.Reviews = nil
		pr.Conversation = nil
		*full = true
		return
	}

	var threads []model.Thread
	for _, t := range pr.Threads {
		if !f.all && t.IsResolved {
			continue
		}
		if f.hideOutdated && t.IsOutdated {
			continue
		}
		if !f.threadMatchesAuthor(t) {
			continue
		}
		threads = append(threads, t)
	}
	pr.Threads = threads

	pr.Reviews = filterReviews(pr.Reviews, f.keepAuthor)
	pr.Conversation = filterComments(pr.Conversation, f.keepAuthor)
}

func (f commentFilter) threadMatchesAuthor(t model.Thread) bool {
	for _, c := range t.Comments {
		if f.keepAuthor(c.Author, c.IsBot) {
			return true
		}
	}
	return len(t.Comments) == 0 && f.noAuthorFilter()
}

func (f commentFilter) noAuthorFilter() bool {
	return f.author == "" && !f.bots && !f.humans
}

// keepAuthor reports whether an item by (login, isBot) survives the
// author-type filters. --author wins over --bots/--humans.
func (f commentFilter) keepAuthor(login string, isBot bool) bool {
	if f.author != "" {
		return login == f.author
	}
	if f.bots && !isBot {
		return false
	}
	if f.humans && isBot {
		return false
	}
	return true
}

func filterReviews(in []model.Review, keep func(string, bool) bool) []model.Review {
	var out []model.Review
	for _, r := range in {
		if keep(r.Author, r.IsBot) {
			out = append(out, r)
		}
	}
	return out
}

func filterComments(in []model.Comment, keep func(string, bool) bool) []model.Comment {
	var out []model.Comment
	for _, c := range in {
		if keep(c.Author, c.IsBot) {
			out = append(out, c)
		}
	}
	return out
}
