// Package cli implements ghx's subcommand dispatch and per-command flag
// parsing. Kept dependency-light: stdlib flag with one FlagSet per command.
package cli

import (
	"fmt"
	"io"
	"os"
)

// Version is the build version, overridable via -ldflags at release time.
var Version = "dev"

// Run dispatches to a subcommand and returns a process exit code.
func Run(args []string) int {
	if len(args) == 0 {
		usage(os.Stderr)
		return 2
	}
	switch args[0] {
	case "comments", "c":
		return runComments(args[1:])
	case "checks", "ck":
		return runChecks(args[1:])
	case "-h", "--help", "help":
		usage(os.Stdout)
		return 0
	case "-v", "--version", "version":
		fmt.Printf("ghx %s\n", Version)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "ghx: unknown command %q\n\n", args[0])
		usage(os.Stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `ghx — gh extras: the PR-review views gh leaves out

Usage:
  ghx comments [PR] [flags]   inline review threads, reviews, and conversation
  ghx checks   [PR] [flags]   CI status-check rollup (buckets + failing detail)

Common flags:
  -R, --repo owner/repo   target repo (default: current directory's repo)
      --json              machine-readable output
  -h, --help              help for a command

With no PR argument, ghx operates on the open PR for the current branch.
Run "ghx comments -h" or "ghx checks -h" for command-specific flags.
`)
}

// fail prints an error to stderr and returns exit code 1.
func fail(err error) int {
	fmt.Fprintf(os.Stderr, "ghx: %v\n", err)
	return 1
}

// splitPR extracts a single positional PR argument from args (whether it
// appears before or after flags) and returns it plus the remaining args for
// flag parsing. stdlib flag stops at the first non-flag token, so we pull the
// positional out ourselves to allow `ghx comments 123 --json`.
func splitPR(args []string) (pr string, rest []string) {
	rest = make([]string, 0, len(args))
	for _, a := range args {
		if pr == "" && len(a) > 0 && a[0] != '-' && isPRish(a) {
			pr = a
			continue
		}
		rest = append(rest, a)
	}
	return pr, rest
}

func isPRish(s string) bool {
	s = trimHash(s)
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func trimHash(s string) string {
	if len(s) > 0 && s[0] == '#' {
		return s[1:]
	}
	return s
}
