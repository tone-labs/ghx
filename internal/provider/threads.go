package provider

import (
	"fmt"

	"github.com/tone-labs/ghx/internal/ghclient"
)

// The two mutations alias their payload field to `result` so a single response
// shape decodes both — the only difference is which mutation GitHub runs.
const resolveThreadMutation = `
mutation($id:ID!) {
  result: resolveReviewThread(input:{threadId:$id}) { thread { isResolved } }
}`

const unresolveThreadMutation = `
mutation($id:ID!) {
  result: unresolveReviewThread(input:{threadId:$id}) { thread { isResolved } }
}`

// ResolveThread marks a review thread resolved; UnresolveThread reopens it.
// Both return the thread's resulting isResolved state so the caller can confirm.
func ResolveThread(c *ghclient.Client, threadID string) (bool, error) {
	return mutateThread(c, resolveThreadMutation, threadID)
}

func UnresolveThread(c *ghclient.Client, threadID string) (bool, error) {
	return mutateThread(c, unresolveThreadMutation, threadID)
}

func mutateThread(c *ghclient.Client, mutation, threadID string) (bool, error) {
	vars := map[string]any{"id": threadID}
	var resp struct {
		Result struct {
			Thread struct {
				IsResolved bool `json:"isResolved"`
			} `json:"thread"`
		} `json:"result"`
	}
	if err := c.GraphQL().Do(mutation, vars, &resp); err != nil {
		return false, fmt.Errorf("toggle thread resolution in %s: %w", c.Slug(), err)
	}
	return resp.Result.Thread.IsResolved, nil
}
