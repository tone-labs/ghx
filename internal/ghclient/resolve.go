package ghclient

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	gh "github.com/cli/go-gh/v2"
)

// ResolvePR returns the PR number to operate on: an explicit argument when
// given (accepts "123" or "#123"), otherwise the open PR for the current
// branch, resolved by shelling out to `gh pr view` (same mechanism the bash
// ghprc/ghpcs helpers used). repoOverride, when set, is passed through as -R.
func ResolvePR(arg, repoOverride string) (int, error) {
	if arg != "" {
		n, err := strconv.Atoi(strings.TrimPrefix(arg, "#"))
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("invalid PR number %q", arg)
		}
		return n, nil
	}

	args := []string{"pr", "view", "--json", "number"}
	if repoOverride != "" {
		args = append(args, "-R", repoOverride)
	}
	stdout, stderr, err := gh.Exec(args...)
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return 0, fmt.Errorf("no PR found for the current branch (pass a PR number): %s", msg)
	}

	var out struct {
		Number int `json:"number"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return 0, fmt.Errorf("parse `gh pr view` output: %w", err)
	}
	if out.Number == 0 {
		return 0, fmt.Errorf("no PR found for the current branch (pass a PR number)")
	}
	return out.Number, nil
}
