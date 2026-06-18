# ghx

`gh` extras — the pull-request review views the GitHub CLI leaves out.

`gh pr checks` is great for CI state at a glance, but `gh` has no first-class
way to see **inline review threads with their resolution state**, the **review
decision gate**, and **PR-level conversation** together. You end up digging
through `gh api` JSON. `ghx` fills that gap with a readable terminal view and a
stable `--json` contract for tooling.

## Commands

```
ghx comments [PR] [flags]   inline threads (+ resolution state), reviews + decision, conversation
ghx checks   [PR] [flags]   CI status-check rollup: bucket counts + failing detail
```

With no `PR` argument, `ghx` operates on the open PR for the current branch
(resolved via `gh`). Pass `-R owner/repo` to target another repository.

### `ghx comments`

```
ghx comments                 # current branch's PR, unresolved threads
ghx comments 1667 --all      # include resolved threads
ghx comments --bots          # only bot-authored items (Copilot, linters, …)
ghx comments --humans        # only human-authored items
ghx comments --author alice  # only a specific login (overrides --bots/--humans)
ghx comments --thread PRRT_… # drill into one thread, full text
ghx comments --full          # full bodies, no truncation
ghx comments --json          # machine-readable (full bodies; ignores --truncate)
```

Default output groups inline threads by file with an `[open]` / `[resolved]` /
`[outdated]` badge and surfaces each thread id so you can drill in. The compact
view truncates bodies (default 200 chars, `--truncate N`, `0` = no limit) and
tells you how much was elided. By default only **unresolved** threads show —
the "what still needs attention" view; `--all` adds resolved ones.

### `ghx checks`

```
ghx checks            # current branch's PR
ghx checks 1667 --json
```

Reuses `gh`'s own status-check rollup (no reimplementation), reshaped into
bucket counts and failing-check detail with workflow links.

## Install

```
go install github.com/cbuchan/ghx@latest   # lands in $GOBIN / $HOME/go/bin
# or
go build -o ghx . && mv ghx ~/.local/bin/
```

Requires the [`gh`](https://cli.github.com) CLI installed and authenticated
(`gh auth status`) — `ghx` inherits gh's auth, host, and config via
[`go-gh`](https://github.com/cli/go-gh), so it works as a standalone binary
(not only as a gh extension).

## Design

Data flows through a normalized model (`internal/model`) that data sources fill
via a provider seam (`internal/provider`) — today a single GraphQL query owns
the read path; swapping or augmenting sources is contained to that package.
Rendering (`internal/render`) offers a human view and a JSON view; the JSON
schema is the stable contract for downstream tooling.

A `gate` subcommand — unioning review decision, unresolved threads, and
required checks into one "what's holding up this PR" payload — is a planned
addition; the model already carries every field it needs.
