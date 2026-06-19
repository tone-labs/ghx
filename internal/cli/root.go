// Package cli implements ghx's subcommands. Built on cobra/pflag so the
// positional PR argument and flags parse in any order, with gh-style long/short
// forms (`--repo`/`-R`) — no hand-rolled arg splitting.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is the build version, overridable via -ldflags at release time.
var Version = "dev"

// cmdError marks a runtime failure (as opposed to a flag/usage error) returned
// from a command's RunE, so Execute can print it in ghx's "ghx: <msg>" form and
// exit 1 while leaving cobra's usage-on-parse-error path (exit 2) intact.
type cmdError struct{ err error }

func (e *cmdError) Error() string { return e.err.Error() }
func (e *cmdError) Unwrap() error { return e.err }

// fail wraps err as a runtime failure for return from a command's RunE.
func fail(err error) error { return &cmdError{err} }

// Execute builds the command tree, runs it, and returns a process exit code:
// 0 success, 1 runtime failure, 2 usage/flag error.
func Execute() int {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		var ce *cmdError
		if errors.As(err, &ce) {
			fmt.Fprintf(os.Stderr, "ghx: %v\n", ce.err)
			return 1
		}
		// Flag/usage/unknown-command error: cobra already printed usage; we own
		// the error line (root.SilenceErrors stops cobra double-printing).
		fmt.Fprintf(os.Stderr, "ghx: %v\n", err)
		return 2
	}
	return 0
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "ghx",
		Short: "gh extras: the PR-review views gh leaves out",
		Long: "ghx — gh extras: the PR-review views gh leaves out\n\n" +
			"Inline review threads (with resolution state), the review-decision\n" +
			"gate, PR-level conversation, and the CI status-check rollup. With no\n" +
			"PR argument, ghx operates on the open PR for the current branch.",
		Version:       Version,
		SilenceErrors: true, // Execute() owns error printing ("ghx: <msg>")
		// SilenceUsage stays false so flag/arg errors still show usage; each
		// RunE flips it true after parsing so runtime failures don't dump usage.
	}
	root.SetVersionTemplate("ghx {{.Version}}\n")
	root.AddCommand(newCommentsCmd(), newChecksCmd())
	return root
}

// prArg returns the optional positional PR argument, or "" when omitted.
func prArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}
