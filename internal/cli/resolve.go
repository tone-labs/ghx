package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tone-labs/ghx/internal/ghclient"
	"github.com/tone-labs/ghx/internal/model"
	"github.com/tone-labs/ghx/internal/provider"
)

func newResolveCmd() *cobra.Command   { return newThreadToggleCmd(true) }
func newUnresolveCmd() *cobra.Command { return newThreadToggleCmd(false) }

// newThreadToggleCmd builds `resolve` / `unresolve`, which are symmetric: each
// acts on the threads it *can* toggle (resolve → unresolved, unresolve →
// resolved), numbered in the same order `ghx comments` lists them. With no
// --thread it lists those targets; with --thread N it toggles the Nth.
func newThreadToggleCmd(resolve bool) *cobra.Command {
	var (
		repo   string
		thread int
	)
	verb := "resolve"
	if !resolve {
		verb = "unresolve"
	}
	state := stateWord(resolve) // the thread state this verb acts on

	cmd := &cobra.Command{
		Use:   verb + " [PR]",
		Short: cases(resolve, "Mark a review thread resolved", "Reopen a resolved review thread"),
		Long: fmt.Sprintf("%s a review thread by its listing number (the same N that\n"+
			"`ghx comments` shows by default). With no --thread, lists the %s threads\n"+
			"you can %s.", cases(resolve, "Resolve", "Unresolve"), state, verb),
		Example: fmt.Sprintf("  ghx %s                 # list %s threads, numbered\n"+
			"  ghx %s --thread 2      # %s thread #2\n"+
			"  ghx %s 1667 --thread 1 # ...on a specific PR", verb, state, verb, verb, verb),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

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

			targets := actionableThreads(pr.Threads, resolve)

			// No --thread: show the numbered targets and how to act on one.
			if thread == 0 {
				listThreadTargets(os.Stdout, targets, prNum, verb, state)
				return nil
			}
			if thread < 1 || thread > len(targets) {
				return fail(fmt.Errorf("--thread %d out of range: %d %s thread(s) on #%d",
					thread, len(targets), state, prNum))
			}

			t := targets[thread-1]
			// Report the thread's actual resulting state (from the mutation), not
			// just the intent — so a no-op or surprising API result reads honestly.
			nowResolved, err := toggleThread(c, t.ID, resolve)
			if err != nil {
				return fail(err)
			}
			fmt.Printf("✓ %s thread %d  %s\n", pastTense(nowResolved), thread, threadLoc(t))
			return nil
		},
	}
	f := cmd.Flags()
	f.IntVar(&thread, "thread", 0, "thread number (from the listing) to "+verb)
	f.StringVarP(&repo, "repo", "R", "", "target repo as owner/repo (default: current repo)")
	return cmd
}

// actionableThreads returns the threads a toggle can act on, in listing order:
// unresolved ones for resolve, resolved ones for unresolve.
func actionableThreads(threads []model.Thread, resolve bool) []model.Thread {
	var out []model.Thread
	for _, t := range threads {
		if t.IsResolved != resolve { // resolve wants unresolved; unresolve wants resolved
			out = append(out, t)
		}
	}
	return out
}

func toggleThread(c *ghclient.Client, id string, resolve bool) (bool, error) {
	if resolve {
		return provider.ResolveThread(c, id)
	}
	return provider.UnresolveThread(c, id)
}

func listThreadTargets(w io.Writer, targets []model.Thread, pr int, verb, state string) {
	if len(targets) == 0 {
		fmt.Fprintf(w, "no %s threads on #%d\n", state, pr)
		return
	}
	fmt.Fprintf(w, "%s threads on #%d:\n", state, pr)
	for i, t := range targets {
		fmt.Fprintf(w, "  %d  %s\n", i+1, threadLoc(t))
		if s := starterSnippet(t); s != "" {
			fmt.Fprintf(w, "     %s\n", s)
		}
	}
	fmt.Fprintf(w, "\n%s one with: ghx %s --thread <N>\n", capitalize(verb), verb)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// threadLoc is the "path:line" locator for a thread.
func threadLoc(t model.Thread) string {
	if t.Path == "" {
		return "(file-level)"
	}
	if t.Line > 0 {
		return fmt.Sprintf("%s:%d", t.Path, t.Line)
	}
	return t.Path
}

// starterSnippet is a one-line "author: body…" preview of the thread's root
// comment, for the listing.
func starterSnippet(t model.Thread) string {
	c := t.Starter()
	if c == nil {
		return ""
	}
	body := c.Body
	if i := strings.IndexByte(body, '\n'); i >= 0 {
		body = body[:i]
	}
	body = strings.TrimSpace(body)
	// Truncate by rune, not byte, so a multibyte glyph (emoji, CJK) at the
	// boundary isn't sliced into a replacement char.
	const max = 60
	if r := []rune(body); len(r) > max {
		body = strings.TrimSpace(string(r[:max])) + "…"
	}
	if body == "" {
		return c.Author
	}
	return c.Author + ": " + body
}

func stateWord(resolve bool) string { return cases(resolve, "unresolved", "resolved") }
func pastTense(resolve bool) string { return cases(resolve, "resolved", "reopened") }
func cases(resolve bool, a, b string) string {
	if resolve {
		return a
	}
	return b
}
