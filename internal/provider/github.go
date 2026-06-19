// Package provider is the data-source seam: it owns how PR review state is
// fetched and maps it into internal/model. Swapping data sources (or leaning
// on a third-party tool) is contained to this package.
package provider

import (
	"fmt"
	"time"

	"github.com/tone-labs/ghx/internal/ghclient"
	"github.com/tone-labs/ghx/internal/model"
)

// prQuery fetches review decision, reviews, PR-level conversation comments, and
// one page of inline review threads (with resolution + outdated state).
// reviewThreads is paginated via $cursor; the top-level connections use a
// single page of 100, which covers effectively every real PR.
const prQuery = `
query($owner:String!, $repo:String!, $pr:Int!, $cursor:String) {
  repository(owner:$owner, name:$repo) {
    pullRequest(number:$pr) {
      number
      title
      url
      state
      isDraft
      reviewDecision
      mergeable
      mergeStateStatus
      author { login }
      reviews(first:100) {
        nodes { author { login __typename } state body submittedAt }
      }
      comments(first:100) {
        nodes { author { login __typename } body createdAt url }
      }
      reviewThreads(first:50, after:$cursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          comments(first:100) {
            nodes { author { login __typename } body createdAt url }
          }
        }
      }
    }
  }
}`

type ghActor struct {
	Login    string `json:"login"`
	Typename string `json:"__typename"`
}

func (a ghActor) login() string {
	if a.Login == "" {
		return "ghost"
	}
	return a.Login
}

type ghComment struct {
	Author    ghActor `json:"author"`
	Body      string  `json:"body"`
	CreatedAt string  `json:"createdAt"`
	URL       string  `json:"url"`
}

func (c ghComment) toModel() model.Comment {
	return model.Comment{
		Author:    c.Author.login(),
		IsBot:     model.IsBotActor(c.Author.Login, c.Author.Typename),
		Body:      c.Body,
		CreatedAt: c.CreatedAt,
		URL:       c.URL,
	}
}

type gqlResp struct {
	Repository struct {
		PullRequest struct {
			Number           int     `json:"number"`
			Title            string  `json:"title"`
			URL              string  `json:"url"`
			State            string  `json:"state"`
			IsDraft          bool    `json:"isDraft"`
			ReviewDecision   string  `json:"reviewDecision"`
			Mergeable        string  `json:"mergeable"`
			MergeStateStatus string  `json:"mergeStateStatus"`
			Author           ghActor `json:"author"`
			Reviews          struct {
				Nodes []struct {
					Author      ghActor `json:"author"`
					State       string  `json:"state"`
					Body        string  `json:"body"`
					SubmittedAt string  `json:"submittedAt"`
				} `json:"nodes"`
			} `json:"reviews"`
			Comments struct {
				Nodes []ghComment `json:"nodes"`
			} `json:"comments"`
			ReviewThreads struct {
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []struct {
					ID         string `json:"id"`
					IsResolved bool   `json:"isResolved"`
					IsOutdated bool   `json:"isOutdated"`
					Path       string `json:"path"`
					Line       int    `json:"line"`
					Comments   struct {
						Nodes []ghComment `json:"nodes"`
					} `json:"comments"`
				} `json:"nodes"`
			} `json:"reviewThreads"`
		} `json:"pullRequest"`
	} `json:"repository"`
}

// FetchPR retrieves and normalizes the full review state of a pull request.
func FetchPR(c *ghclient.Client, pr int) (*model.PR, error) {
	out := &model.PR{}
	var cursor *string
	first := true

	for {
		vars := map[string]any{
			"owner":  c.Owner,
			"repo":   c.Repo,
			"pr":     pr,
			"cursor": cursor,
		}
		var resp gqlResp
		if err := c.GraphQL().Do(prQuery, vars, &resp); err != nil {
			return nil, fmt.Errorf("fetch PR #%d in %s: %w", pr, c.Slug(), err)
		}
		p := resp.Repository.PullRequest

		// Top-level fields and one-page connections only need the first page.
		if first {
			out.Number = p.Number
			out.Title = p.Title
			out.URL = p.URL
			out.State = p.State
			out.IsDraft = p.IsDraft
			out.Author = p.Author.login()
			out.ReviewDecision = p.ReviewDecision
			out.Mergeable = p.Mergeable
			out.MergeStateStatus = p.MergeStateStatus

			for _, r := range p.Reviews.Nodes {
				out.Reviews = append(out.Reviews, model.Review{
					Author:      r.Author.login(),
					IsBot:       model.IsBotActor(r.Author.Login, r.Author.Typename),
					State:       r.State,
					Body:        r.Body,
					SubmittedAt: r.SubmittedAt,
				})
			}
			for _, cm := range p.Comments.Nodes {
				out.Conversation = append(out.Conversation, cm.toModel())
			}
			first = false
		}

		for _, t := range p.ReviewThreads.Nodes {
			thread := model.Thread{
				ID:         t.ID,
				Path:       t.Path,
				Line:       t.Line,
				IsResolved: t.IsResolved,
				IsOutdated: t.IsOutdated,
			}
			for _, cm := range t.Comments.Nodes {
				thread.Comments = append(thread.Comments, cm.toModel())
			}
			out.Threads = append(out.Threads, thread)
		}

		if !p.ReviewThreads.PageInfo.HasNextPage {
			break
		}
		end := p.ReviewThreads.PageInfo.EndCursor
		cursor = &end
	}

	// GitHub computes mergeable / mergeStateStatus lazily — the first query on an
	// open PR often returns UNKNOWN, then a follow-up returns the real value. Only
	// when it matters (open PR, still unknown), re-query a few times so the gate
	// gets GitHub's authoritative merge-button state instead of falling back to a
	// heuristic. Best-effort: a failure or persistent UNKNOWN just leaves the gate
	// to degrade gracefully.
	if out.State == "OPEN" && mergeStateUnknown(out.MergeStateStatus) {
		refreshMergeState(c, pr, out)
	}

	return out, nil
}

const (
	mergeStateRetries = 3
	mergeStateBackoff = 700 * time.Millisecond
)

// mergeQuery refetches only the lazily-computed merge-button signals, so a
// retry doesn't re-paginate the whole PR.
const mergeQuery = `
query($owner:String!, $repo:String!, $pr:Int!) {
  repository(owner:$owner, name:$repo) {
    pullRequest(number:$pr) { mergeable mergeStateStatus }
  }
}`

func mergeStateUnknown(s string) bool { return s == "" || s == "UNKNOWN" }

func refreshMergeState(c *ghclient.Client, pr int, out *model.PR) {
	vars := map[string]any{"owner": c.Owner, "repo": c.Repo, "pr": pr}
	for attempt := range mergeStateRetries {
		// Re-query first: the initial FetchPR query already raced ahead of
		// GitHub's lazy computation, so this follow-up often returns the real
		// value immediately. Only back off *between* attempts, never before the
		// first — otherwise the common case eats a needless 700ms.
		if attempt > 0 {
			time.Sleep(mergeStateBackoff)
		}
		var resp gqlResp
		if err := c.GraphQL().Do(mergeQuery, vars, &resp); err != nil {
			return
		}
		p := resp.Repository.PullRequest
		if !mergeStateUnknown(p.MergeStateStatus) {
			out.Mergeable = p.Mergeable
			out.MergeStateStatus = p.MergeStateStatus
			return
		}
	}
}
