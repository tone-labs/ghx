// Package ghclient wraps go-gh so the rest of the tool never touches gh's
// auth, host resolution, or subprocess details directly. Auth and host are
// inherited from the installed gh CLI's configuration (env, keyring, config
// file) — this works in a standalone binary, not only in a gh extension, as
// long as gh itself is installed and authenticated.
package ghclient

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

// Client carries a GraphQL client plus the resolved repo coordinates.
type Client struct {
	gql   *api.GraphQLClient
	Owner string
	Repo  string
	Host  string
}

// New resolves the target repo (explicit "owner/repo" override, else the repo
// of the current working directory) and builds an authenticated GraphQL client.
func New(repoOverride string) (*Client, error) {
	var repo repository.Repository
	var err error
	if repoOverride != "" {
		repo, err = repository.Parse(repoOverride)
		if err != nil {
			return nil, fmt.Errorf("parse repo %q: %w", repoOverride, err)
		}
	} else {
		repo, err = repository.Current()
		if err != nil {
			return nil, fmt.Errorf("could not resolve the current repository (run inside a git repo or pass -R owner/repo): %w", err)
		}
	}

	host := repo.Host
	if host == "" {
		host = "github.com"
	}

	gql, err := api.NewGraphQLClient(api.ClientOptions{Host: host})
	if err != nil {
		return nil, fmt.Errorf("build GraphQL client (is gh installed and authenticated? try `gh auth status`): %w", err)
	}

	return &Client{gql: gql, Owner: repo.Owner, Repo: repo.Name, Host: host}, nil
}

// GraphQL exposes the underlying client for providers.
func (c *Client) GraphQL() *api.GraphQLClient { return c.gql }

// Slug returns "owner/repo" for messages and gh -R passthrough.
func (c *Client) Slug() string { return c.Owner + "/" + c.Repo }
