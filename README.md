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
ghx comments --thread 2      # drill into thread #2 from the listing, full text
ghx comments --conversation  # expand the PR-level conversation
ghx comments --full          # expand everything: full bodies + conversation
ghx comments --lines 4       # cap each body at 4 wrapped lines (0 = unlimited)
ghx comments --width 100     # wrap at 100 cols (0 / default = full terminal width)
ghx comments --json          # machine-readable (full bodies)
```

The default view leads with the **decision** and unresolved count, lists
**reviews** (✓/✗ glyphs), then **numbered inline threads** grouped by file —
basename and line in front, directory elided behind. Bodies wrap to the full
terminal width (measured at run time, like `gh pr checks`; `--width N` to
override) and are capped at 2 lines (`--lines N`). Threads are numbered so you
drill in by index (`--thread N`) rather than copying a node id. Only
**unresolved** threads show by default (`--all` adds resolved, badged); the
**conversation** collapses to a one-line count (`--conversation` to expand) —
that's where bot noise lives. Color is on for a terminal, off when piped.

### `ghx checks`

```
ghx checks            # current branch's PR
ghx checks 1667 --json
```

Reuses `gh`'s own status-check rollup (no reimplementation), reshaped into
bucket counts and failing-check detail with workflow links.

## Install

```
go install github.com/tone-labs/ghx@latest   # lands in $GOBIN / $HOME/go/bin
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
