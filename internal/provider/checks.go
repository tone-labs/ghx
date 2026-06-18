package provider

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	gh "github.com/cli/go-gh/v2"

	"github.com/tone-labs/ghx/internal/model"
)

// FetchChecks reuses gh's own status-check rollup rather than reimplementing
// it: it shells out to `gh pr checks --json` and reshapes the result into
// bucket counts plus failing-check detail (the job the bash ghpcs did).
func FetchChecks(repoOverride string, pr int) (*model.Checks, error) {
	args := []string{"pr", "checks", strconv.Itoa(pr), "--json", "name,bucket,state,workflow,link"}
	if repoOverride != "" {
		args = append(args, "-R", repoOverride)
	}
	stdout, stderr, err := gh.Exec(args...)

	// `gh pr checks` exits non-zero when checks are failing (1) or pending (8)
	// even though the command itself succeeded and printed JSON. Treat a
	// non-empty stdout as success; only a truly empty stdout is a real error.
	if stdout.Len() == 0 {
		if err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return nil, fmt.Errorf("gh pr checks for #%d: %s", pr, msg)
		}
		// No error and no output means the PR has no checks at all.
		return &model.Checks{Counts: map[string]int{}}, nil
	}

	var raw []struct {
		Name     string `json:"name"`
		Bucket   string `json:"bucket"`
		State    string `json:"state"`
		Workflow string `json:"workflow"`
		Link     string `json:"link"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		return nil, fmt.Errorf("parse gh pr checks output: %w", err)
	}

	checks := &model.Checks{Counts: map[string]int{}, Total: len(raw)}
	for _, r := range raw {
		checks.Counts[r.Bucket]++
		if r.Bucket == "fail" || r.Bucket == "cancel" {
			checks.Failing = append(checks.Failing, model.Check{
				Name:     r.Name,
				Bucket:   r.Bucket,
				State:    r.State,
				Workflow: r.Workflow,
				Link:     r.Link,
			})
		}
	}
	sort.Slice(checks.Failing, func(i, j int) bool {
		return checks.Failing[i].Name < checks.Failing[j].Name
	})
	return checks, nil
}
