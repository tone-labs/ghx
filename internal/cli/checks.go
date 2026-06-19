package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/tone-labs/ghx/internal/ghclient"
	"github.com/tone-labs/ghx/internal/provider"
	"github.com/tone-labs/ghx/internal/render"
)

func newChecksCmd() *cobra.Command {
	var (
		repo    string
		jsonOut bool
	)
	cmd := &cobra.Command{
		Use:     "checks [PR]",
		Aliases: []string{"ck"},
		Short:   "CI status-check rollup (buckets + failing detail)",
		Long: "CI status-check rollup: bucket counts then failing-check detail.\n" +
			"Defaults to the current branch's PR.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true // parsing done; runtime errors shouldn't dump usage

			// checks resolves the PR via gh too, but needs the repo slug for -R
			// passthrough; New() validates/normalizes the repo override early.
			if repo != "" {
				if _, err := ghclient.New(repo); err != nil {
					return fail(err)
				}
			}
			prNum, err := ghclient.ResolvePR(prArg(args), repo)
			if err != nil {
				return fail(err)
			}
			ck, err := provider.FetchChecks(repo, prNum)
			if err != nil {
				return fail(err)
			}

			if jsonOut {
				if err := render.JSON(os.Stdout, ck); err != nil {
					return fail(err)
				}
				return nil
			}
			render.ChecksView(os.Stdout, prNum, ck)
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVar(&jsonOut, "json", false, "machine-readable JSON output")
	f.StringVarP(&repo, "repo", "R", "", "target repo as owner/repo (default: current repo)")
	return cmd
}
