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
		repo     string
		jsonOut  bool
		exitCode bool
		color    colorFlag
	)
	cmd := &cobra.Command{
		Use:     "checks [PR]",
		Aliases: []string{"ck"},
		Short:   "CI status-check rollup (buckets + failing detail)",
		Long: "CI status-check rollup: bucket counts then failing-check detail.\n" +
			"Defaults to the current branch's PR.",
		Example: "  ghx checks                 # current branch's PR\n" +
			"  ghx checks 1667            # a specific PR\n" +
			"  ghx checks --json | jq .   # machine-readable rollup\n" +
			"  ghx checks --exit-code     # exit 8 if any check is failing (for CI/scripts)",
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
			} else {
				render.ChecksView(os.Stdout, prNum, ck, color.mode)
			}

			// --exit-code: after emitting output, signal red CI via the exit
			// status (Failing = the fail+cancel buckets) so scripts/CI gates can
			// branch on it without parsing. Code 8 matches `gh pr checks` and
			// stays distinct from 1 (runtime error) / 2 (usage). Pending and
			// skipping don't trip it.
			if exitCode && len(ck.Failing) > 0 {
				return statusExit(8)
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVar(&jsonOut, "json", false, "machine-readable JSON output")
	f.BoolVar(&exitCode, "exit-code", false, "exit 8 if any check is failing (output is still printed)")
	f.Var(&color, "color", "when to use color: auto, always, never")
	f.StringVarP(&repo, "repo", "R", "", "target repo as owner/repo (default: current repo)")
	return cmd
}
