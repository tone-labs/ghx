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
ghx gate     [PR] [flags]   mergeability verdict: decision + threads + checks (exit 8 if blocked)
ghx version                 print the version
```

With no `PR` argument, `ghx` operates on the open PR for the current branch
(resolved via `gh`). Pass `-R, --repo owner/repo` to target another repository.
Commands accept short aliases (`ghx c`, `ghx ck`, `ghx g`), and `-h`/`--help`
works on every command (`ghx comments -h` lists flags and examples).

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
ghx comments --color never   # disable color (--color always forces it when piped)
ghx comments --json          # machine-readable (full bodies)
```

The default view leads with the **decision** and unresolved count, lists
**reviews** (✓/✗ glyphs), then **numbered inline threads** grouped by file —
basename and line in front, directory elided behind. Bodies wrap to the full
terminal width (measured at run time, like `gh pr checks`; `--width N` to
override) and are capped at 2 lines (`--lines N`). Threads are numbered so you
drill in by index (`--thread N`) rather than copying a node id. Only
**unresolved** threads show by default (`--all` adds resolved, badged) while
**outdated** threads show until you `--hide-outdated` — the asymmetry is
deliberate: resolved means *done*, outdated means *the code moved but the note
may still matter*. The **conversation** collapses to a one-line count
(`--conversation` to expand) — that's where bot noise lives. Color is on for a
terminal and off when piped; `--color auto|always|never` overrides that, and
`NO_COLOR` is honored.

### `ghx checks`

```
ghx checks                # current branch's PR
ghx checks 1667 --json
ghx checks --exit-code    # exit 8 if any check is failing (for CI/scripts)
```

Reuses `gh`'s own status-check rollup (no reimplementation), reshaped into
**colored bucket counts** (failures first — green pass / red fail / yellow
pending) and failing-check detail with workflow links. Color follows the same
rules as `comments` (`--color auto|always|never`, `NO_COLOR` honored).

### `ghx gate`

```
ghx gate                  # is the current branch's PR ready to merge?
ghx gate 1667 --json      # structured verdict
ghx gate && gh pr merge   # gate before merging
```

Anchors on GitHub's own merge-button state (`mergeStateStatus`) — which already
accounts for **branch protection**, **required reviews**, and **required** checks
— then explains it with the finer signals (review decision, unresolved threads,
CI checks) as a `MERGEABLE` / `BLOCKED` verdict with blockers listed. Because the
anchor is the merge button itself, the verdict agrees with it: a red *non-required*
check (GitHub's `UNSTABLE`) doesn't block, a merge conflict or out-of-date branch
does. When GitHub hasn't computed a state yet it falls back to a best-effort union
of the finer signals. Exits `8` when blocked (no flag needed — the verdict is the
command's purpose), so it gates a merge or a CI step.

## Scripting

`--json` is a boolean toggle that emits the full normalized structure (unlike
`gh`'s `--json <fields>` form) — a stable contract you pipe to `jq`:

```
ghx comments --json | jq -r '.threads[] | select(.isResolved | not) | .path'
ghx checks  --json | jq -r '.failing[] | "\(.name)\t\(.link)"'
```

Exit codes: `0` success, `1` runtime error (no PR found, `gh` failure), `2`
usage/flag error. With `--exit-code`, `ghx checks` additionally returns `8` (the
`gh pr checks` convention — distinct from `1`/`2`) when any check is failing, so
CI gates and the `/scout` flow can branch on status without parsing JSON. `ghx
gate` returns `8` when the PR is blocked — no flag needed, since the verdict is
the whole point of the command.

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

The `gate` subcommand unions the review decision, unresolved threads, and CI
checks into one "what's holding up this PR" verdict (`internal/gate`, a pure
evaluation over the same normalized model the other views fill).
