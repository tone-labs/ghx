package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/tone-labs/ghx/internal/gate"
	"github.com/tone-labs/ghx/internal/ghclient"
	"github.com/tone-labs/ghx/internal/provider"
	"github.com/tone-labs/ghx/internal/render"
)

func newGateCmd() *cobra.Command {
	var (
		repo    string
		jsonOut bool
		color   colorFlag
	)
	cmd := &cobra.Command{
		Use:     "gate [PR]",
		Aliases: []string{"g"},
		Short:   "One-shot mergeability verdict (decision + threads + checks)",
		Long: "Union the review decision, unresolved review threads, and CI checks into\n" +
			"a single mergeability verdict. Exits 8 when the PR is blocked (distinct\n" +
			"from 1 = error / 2 = usage), so it works as a CI gate or pre-merge check.",
		Example: "  ghx gate                  # is the current branch's PR ready to merge?\n" +
			"  ghx gate 1667             # a specific PR\n" +
			"  ghx gate --json | jq .    # structured verdict\n" +
			"  ghx gate && gh pr merge   # gate before merging",
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
			ck, err := provider.FetchChecks(repo, prNum)
			if err != nil {
				return fail(err)
			}

			result := gate.Evaluate(pr, ck)
			if jsonOut {
				if err := render.JSON(os.Stdout, result); err != nil {
					return fail(err)
				}
			} else {
				render.GateView(os.Stdout, result, color.mode)
			}

			// The verdict IS the exit status: 8 when blocked, mirroring
			// `checks --exit-code` so a gate composes in scripts / CI.
			if !result.Mergeable {
				return statusExit(8)
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVar(&jsonOut, "json", false, "machine-readable JSON output")
	f.Var(&color, "color", "when to use color: auto, always, never")
	f.StringVarP(&repo, "repo", "R", "", "target repo as owner/repo (default: current repo)")
	return cmd
}
