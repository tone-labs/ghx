package cli

import (
	"flag"
	"os"

	"github.com/tone-labs/ghx/internal/ghclient"
	"github.com/tone-labs/ghx/internal/provider"
	"github.com/tone-labs/ghx/internal/render"
)

func runChecks(args []string) int {
	fs := flag.NewFlagSet("checks", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		repo    string
		jsonOut = fs.Bool("json", false, "machine-readable JSON output")
	)
	fs.StringVar(&repo, "repo", "", "target repo as owner/repo (default: current repo)")
	fs.StringVar(&repo, "R", "", "shorthand for --repo")
	fs.Usage = func() {
		fs.Output().Write([]byte("Usage: ghx checks [PR] [flags]\n\n" +
			"CI status-check rollup: bucket counts then failing-check detail.\n" +
			"Defaults to the current branch's PR.\n\nFlags:\n"))
		fs.PrintDefaults()
	}

	prArg, rest := splitPR(args, valueFlagNames(fs))
	if err := fs.Parse(rest); err != nil {
		return 2
	}

	// checks resolves the PR via gh too, but needs the repo slug for -R
	// passthrough; New() validates/normalizes the repo override early.
	if repo != "" {
		if _, err := ghclient.New(repo); err != nil {
			return fail(err)
		}
	}
	prNum, err := ghclient.ResolvePR(prArg, repo)
	if err != nil {
		return fail(err)
	}
	ck, err := provider.FetchChecks(repo, prNum)
	if err != nil {
		return fail(err)
	}

	if *jsonOut {
		if err := render.JSON(os.Stdout, ck); err != nil {
			return fail(err)
		}
		return 0
	}
	render.ChecksView(os.Stdout, prNum, ck)
	return 0
}
