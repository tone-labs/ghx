// Command ghx provides the PR-review views the gh CLI leaves out: inline
// review threads with resolution state, the review-decision gate, PR-level
// conversation, and the CI status-check rollup.
package main

import (
	"os"

	"github.com/tone-labs/ghx/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
