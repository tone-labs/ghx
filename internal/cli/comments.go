package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tone-labs/ghx/internal/ghclient"
	"github.com/tone-labs/ghx/internal/model"
	"github.com/tone-labs/ghx/internal/provider"
	"github.com/tone-labs/ghx/internal/render"
)

// defaultBodyLines is the wrapped-line cap per comment in the compact view.
// 0 means unlimited (set by --full or --lines 0).
const defaultBodyLines = 2

func newCommentsCmd() *cobra.Command {
	var (
		repo         string
		all          bool
		hideOutdated bool
		bots         bool
		humans       bool
		author       string
		thread       int
		conversation bool
		full         bool
		lines        int
		width        int
		jsonOut      bool
		color        colorFlag
	)
	cmd := &cobra.Command{
		Use:     "comments [PR]",
		Aliases: []string{"c"},
		Short:   "Inline review threads, reviews + decision, and conversation",
		Long: "Inline review threads (with resolution state), reviews + decision,\n" +
			"and PR-level conversation. Defaults to the current branch's PR and to\n" +
			"unresolved threads only.",
		Example: "  ghx comments                  # current branch's PR, unresolved threads\n" +
			"  ghx comments 1667 --all       # include resolved threads\n" +
			"  ghx comments --bots           # only bot-authored items\n" +
			"  ghx comments --thread 2       # drill into thread #2, full text\n" +
			"  ghx comments --full           # expand bodies + conversation\n" +
			"  ghx comments --json | jq .    # machine-readable (full bodies)",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true // parsing done; runtime errors shouldn't dump usage

			c, err := ghclient.New(repo)
			if err != nil {
				return fail(err)
			}
			prNum, err := ghclient.ResolvePR(prArg(args), repo)
			if err != nil {
				return fail(err)
			}
			pr, err := provider.FetchPR(c, prNum)
			if err != nil {
				return fail(err)
			}

			commentFilter{
				all:          all,
				hideOutdated: hideOutdated,
				bots:         bots,
				humans:       humans,
				author:       author,
			}.apply(pr)

			bodyLines := lines
			showConv := conversation
			if full {
				bodyLines = 0
				showConv = true
			}
			if thread > 0 {
				if err := selectThread(pr, thread); err != nil {
					return fail(err)
				}
				bodyLines = 0 // a drilled-in thread always shows full text
			}

			if jsonOut {
				if err := render.JSON(os.Stdout, pr); err != nil {
					return fail(err)
				}
				return nil
			}
			render.Comments(os.Stdout, pr, render.Options{
				Width:            width,
				BodyLines:        bodyLines,
				ShowConversation: showConv,
				Color:            color.mode,
			})
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVar(&all, "all", false, "include resolved threads (default: unresolved only)")
	f.BoolVar(&hideOutdated, "hide-outdated", false, "exclude outdated threads")
	f.BoolVar(&bots, "bots", false, "only bot-authored items")
	f.BoolVar(&humans, "humans", false, "only human-authored items")
	f.StringVar(&author, "author", "", "only items authored by this login (overrides --bots/--humans)")
	f.IntVar(&thread, "thread", 0, "drill into thread number N (from the listing), in full")
	f.BoolVar(&conversation, "conversation", false, "expand PR-level conversation (default: collapsed)")
	f.BoolVar(&full, "full", false, "expand everything: full bodies + conversation")
	f.IntVar(&lines, "lines", defaultBodyLines, "max wrapped lines per comment body (0 = unlimited)")
	f.IntVar(&width, "width", 0, "wrap width (0 = detect terminal width)")
	f.BoolVar(&jsonOut, "json", false, "machine-readable JSON output")
	f.Var(&color, "color", "when to use color: auto, always, never")
	f.StringVarP(&repo, "repo", "R", "", "target repo as owner/repo (default: current repo)")
	return cmd
}

// colorFlag is a pflag.Value for --color, parsing auto|always|never into a
// render.ColorMode. Invalid values fail at parse time (→ usage error, exit 2).
type colorFlag struct{ mode render.ColorMode }

func (c *colorFlag) String() string {
	switch c.mode {
	case render.ColorAlways:
		return "always"
	case render.ColorNever:
		return "never"
	default:
		return "auto"
	}
}

func (c *colorFlag) Set(s string) error {
	switch s {
	case "auto":
		c.mode = render.ColorAuto
	case "always":
		c.mode = render.ColorAlways
	case "never":
		c.mode = render.ColorNever
	default:
		return fmt.Errorf(`invalid color %q: want "auto", "always", or "never"`, s)
	}
	return nil
}

func (c *colorFlag) Type() string { return "auto|always|never" }

// selectThread reduces pr to the single thread at 1-based index n (its position
// in the filtered listing) and drops reviews/conversation for a focused view.
func selectThread(pr *model.PR, n int) error {
	if n < 1 || n > len(pr.Threads) {
		return fmt.Errorf("--thread %d out of range: the listing shows %d thread(s)", n, len(pr.Threads))
	}
	pr.Threads = []model.Thread{pr.Threads[n-1]}
	pr.Reviews = nil
	pr.Conversation = nil
	return nil
}

type commentFilter struct {
	all          bool
	hideOutdated bool
	bots         bool
	humans       bool
	author       string
}

// apply mutates pr in place per the active filters.
func (f commentFilter) apply(pr *model.PR) {
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
