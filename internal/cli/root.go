// Package cli implements ghx's subcommands. Built on cobra/pflag so the
// positional PR argument and flags parse in any order, with gh-style long/short
// forms (`--repo`/`-R`) — no hand-rolled arg splitting.
package cli

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Version is the build version. Defaults to "dev"; an explicit -ldflags override
// at release time wins, otherwise version() fills it from the module version Go
// embeds into `go install module@vX.Y.Z` builds.
var Version = "dev"

// version resolves the string shown by `--version` / `ghx version`. Precedence:
// an explicit -ldflags override of Version; then the module version Go embeds in
// binaries built via `go install ...@vX.Y.Z` (so a tagged install self-reports
// its tag); then "dev" for a plain `go build` from a checkout.
func version() string {
	if Version != "dev" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return Version
}

// cmdError marks a runtime failure (as opposed to a flag/usage error) returned
// from a command's RunE, so Execute can print it in ghx's "ghx: <msg>" form and
// exit 1 while leaving cobra's usage-on-parse-error path (exit 2) intact.
type cmdError struct{ err error }

func (e *cmdError) Error() string { return e.err.Error() }
func (e *cmdError) Unwrap() error { return e.err }

// fail wraps err as a runtime failure for return from a command's RunE.
func fail(err error) error { return &cmdError{err} }

// statusError requests a specific exit code with no output — a status signal,
// not a failure (e.g. `checks --exit-code` when CI is red). Returned from RunE.
type statusError struct{ code int }

func (e *statusError) Error() string { return "" }

// statusExit makes Execute exit with code and print nothing.
func statusExit(code int) error { return &statusError{code: code} }

// Execute builds the command tree, runs it, and returns a process exit code.
func Execute() int {
	err := newRootCmd().Execute()
	code, show := resolveExit(err)
	if show {
		fmt.Fprintf(os.Stderr, "ghx: %v\n", err)
	}
	return code
}

// resolveExit maps a command error to its process exit code and whether to
// print it: 0 success; 1 runtime failure (*cmdError via fail()); 2 usage/flag/
// unknown-command error (cobra's own, already-printed-usage errors). A
// *statusError carries an explicit code and prints nothing.
func resolveExit(err error) (code int, show bool) {
	switch {
	case err == nil:
		return 0, false
	case errors.As(err, new(*statusError)):
		var se *statusError
		errors.As(err, &se)
		return se.code, false
	case errors.As(err, new(*cmdError)):
		return 1, true
	default:
		return 2, true
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "ghx",
		Short: "gh extras: the PR-review views gh leaves out",
		Long: "ghx — gh extras: the PR-review views gh leaves out\n\n" +
			"Inline review threads (with resolution state), the review-decision\n" +
			"gate, PR-level conversation, and the CI status-check rollup. With no\n" +
			"PR argument, ghx operates on the open PR for the current branch.",
		Version:       version(),
		SilenceErrors: true, // Execute() owns error printing ("ghx: <msg>")
		// SilenceUsage stays false so flag/arg errors still show usage; each
		// RunE flips it true after parsing so runtime failures don't dump usage.
	}
	root.SetVersionTemplate("ghx {{.Version}}\n")
	// Keep the surface minimal: drop cobra's auto `completion` command for now.
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(newCommentsCmd(), newChecksCmd(), newGateCmd(), newResolveCmd(), newUnresolveCmd(), newVersionCmd())
	return root
}

// newVersionCmd preserves the `ghx version` form (alongside cobra's --version),
// matching gh and the pre-cobra CLI.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the ghx version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "ghx %s\n", version())
		},
	}
}

// prArg returns the optional positional PR argument, or "" when omitted.
func prArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}
