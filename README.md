# ghx

`gh` extras — the pull-request review views the GitHub CLI leaves out.

`gh pr checks` is great for CI state at a glance, but `gh` has no first-class
way to see **inline review threads with their resolution state**, the **review
decision gate**, and **PR-level conversation** together. You end up digging
through `gh api` JSON. `ghx` fills that gap with a readable terminal view and a
stable `--json` contract for tooling.

> **Why ghx?** The terminal-native, "never open the PR page again" pitch — who
> it's for and the bet behind it — lives in [the vision](docs/vision.md).

_Status: pre-1.0 and actively developed. The `--json` structure and exit codes
are a deliberate stable contract; the human-readable views may still evolve._

## What it looks like

`ghx comments` — inline threads with resolution state, reviews, and the review
decision in one view, threads numbered so you drill in by index:

```text
#42  Add xpath support
https://github.com/o/r/pull/42
CHANGES REQUESTED · 1 unresolved · 2 reviews · open · draft

REVIEWS
  ✓ ci-bot (bot)  approved  ·  1 day ago
  ✗ bob  changes requested  ·  1 day ago
      Needs work

THREADS · 2

  [1] a.ts:72  src
      bob  ·  1 day ago
      Why does this return element-not-found?
    ↳ alice  ·  13 hours ago  ·  author
      Good catch, fixing.

  [2] b.ts:10  src  (resolved, outdated)
      lint-bot (bot)  ·  1 day ago
      nit: prefer const here for the long explanation that should be truncated in the compact view

CONVERSATION · 1 comment   — --conversation to show

drill in:  ghx comments 42 --thread <n>
```

`ghx gate` — one merge-readiness verdict with blockers called out (exits `8` when
blocked, so it gates a merge or a CI step):

```text
#42  Add xpath support
https://github.com/o/r/pull/42
✗ BLOCKED  ·  2 blockers
merge state: blocked

  ✗ review   changes requested
  ○ threads  2 threads unresolved
  ✗ checks   1 check failing

  ○ = advisory: related to the merge, but not blocking it
```

`ghx checks` — the CI rollup as colored bucket counts (failures first) with
workflow links:

```text
checks  PR #42  ·  6 checks

  ✗ 1 fail
  ○ 2 pending
  ✓ 3 pass

FAILING
  ✗ lint  CI
    https://x/runs/1
```

> Output is colored in a terminal (green pass / red fail / yellow pending) and
> plain when piped; the samples above are the plain form.

## Commands

```
ghx comments  [PR] [flags]   inline threads (+ resolution state), reviews + decision, conversation
ghx checks    [PR] [flags]   CI status-check rollup: bucket counts + failing detail
ghx gate      [PR] [flags]   mergeability verdict: decision + threads + checks (exit 8 if blocked)
ghx resolve   [PR] [flags]   mark a review thread resolved (by listing number)
ghx unresolve [PR] [flags]   reopen a resolved review thread
ghx version                  print the version
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

### `ghx resolve` / `ghx unresolve`

```
ghx resolve                  # list unresolved threads, numbered
ghx resolve --thread 2       # resolve thread #2
ghx unresolve                # list resolved threads, numbered
ghx unresolve --thread 1     # reopen thread #1
```

Toggle a review thread's resolution state by its **listing number** — the same
`N` that `ghx comments` shows by default — so you never copy a node id. Each verb acts on the
threads it can: `resolve` numbers the *unresolved* threads, `unresolve` the
*resolved* ones. With no `--thread`, it lists those targets (a one-line preview
each) so you can pick; with `--thread N` it toggles the Nth. These are the first
*write* verbs in ghx — explicit, single-thread, no bulk flag.

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
CI gates and automation can branch on status without parsing JSON. `ghx
gate` returns `8` when the PR is blocked — no flag needed, since the verdict is
the whole point of the command.

## Install

```sh
brew install tone-labs/tap/ghx                # Homebrew (macOS / Linux)
# or
go install github.com/tone-labs/ghx@latest    # needs the Go toolchain; lands in $GOBIN / $HOME/go/bin
# or
go build -o ghx . && mv ghx ~/.local/bin/
```

Prefer a binary? Prebuilt archives for macOS and Linux (amd64/arm64), with
checksums, are attached to every
[release](https://github.com/tone-labs/ghx/releases/latest).

The [Homebrew tap](https://github.com/tone-labs/homebrew-tap) builds from source
(no macOS notarization prompt) and pulls in `gh` automatically; `brew install
--HEAD tone-labs/tap/ghx` tracks `main`. The `go` paths need the Go toolchain.

Requires the [`gh`](https://cli.github.com) CLI installed and authenticated
(`gh auth status`) — `ghx` inherits gh's auth, host, and config via
[`go-gh`](https://github.com/cli/go-gh). It is **not** a `gh` extension: it's a
standalone binary that reuses gh's auth, so there's nothing to `gh extension
install`.

## Design

Data flows through a normalized model (`internal/model`) that data sources fill
via a provider seam (`internal/provider`) — today a single GraphQL query owns
the read path; swapping or augmenting sources is contained to that package.
Rendering (`internal/render`) offers a human view and a JSON view; the JSON
schema is the stable contract for downstream tooling.

The `gate` subcommand unions the review decision, unresolved threads, and CI
checks into one "what's holding up this PR" verdict (`internal/gate`, a pure
evaluation over the same normalized model the other views fill).
