# ghx — roadmap

A living prioritization pass. The *why* lives in [`vision.md`](./vision.md);
this is the *what next*.

It is grounded in a survey of what the GitHub web UI does versus what `gh` actually
does (introspected against `gh` 2.93). The point: not everything annoying in the web
UI is a `gh` *capability* gap. Sorting honestly is what keeps ghx aimed at real
differentiation instead of re-wrapping commands `gh` already ships.

## The bar

Not 100% coverage of the web UI — the asymptote costs more than it returns. Target:
**open the web UI at most about once a week.** Win the daily friction; leave the rare
admin to the browser.

## Three buckets

Every web-UI capability falls into one of these. Only **A** is real differentiation.

- **A — `gh` genuinely can't or won't.** The true UI > `gh` capability gap. ghx's turf.
- **B — `gh` can, but awkwardly / not PR-aware.** Ergonomics. ghx adds *convenience*,
  not capability — and we say so. Quick wins, honestly labeled.
- **C — `gh` does fine.** Leave it. Wrapping it is noise.

## The survey

| Web UI capability | `gh` today | Bucket | ghx stance |
| --- | --- | --- | --- |
| Inline review **threads w/ resolution + outdated state** | `pr view --comments` flattens; no resolve state | **A** | ✅ done (`ghx comments`) |
| **Synthesized "what's blocking merge"** verdict | pieces only (`pr view --json reviewDecision,mergeStateStatus,statusCheckRollup` + `pr checks --required`); no single verdict | **A** (synthesis) | ✅ `ghx gate` (honest, merge-button-anchored) |
| **Resolve / unresolve** a thread | ❌ none (GraphQL only) | **A** (write) | ✅ done (operational write) |
| **Compose inline / multi-comment review** (file:line) | ❌ `pr review` is PR-level only | **A** (write, content) | gap → agent-driven (content write) |
| **Diff with review threads in context** | `pr diff` shows diff; no threads/annotations overlay | **A** (heavy) | unbuilt frontier |
| Inline check annotations on diff lines; suggested changes; "viewed" tracking | ❌ | **A** (niche) | low value in a CLI; skip for now |
| **Rerun failed / flaky checks** | ✅ `gh run rerun --failed` (needs run-id) | **B** | `ghx rerun` = PR-aware wrapper |
| **Read a failing check's log** | ✅ `gh run view --log-failed` (needs run-id) | **B** | `ghx logs` = PR-aware wrapper |
| Required-vs-optional checks | ✅ `gh pr checks --required` | **B** | `ghx gate` consumes this |
| Checks rollup / what failed | ✅ `gh pr checks` | **B/C** | ✅ `ghx checks` (nicer shape) |
| Merge; edit title/body/labels/reviewers; ready/draft; update-branch; PR-level comment & approve; basic diff; artifacts; cancel; watch | ✅ `pr merge` / `pr edit` / `pr ready` / `pr update-branch` / `pr comment` / `pr review` / `pr diff` / `run download` / `run cancel` / `run watch` | **C** | leave to `gh` |

## What the survey corrected

- **`ghx rerun` / `ghx logs` are bucket B, not Tier-1 capability gaps.** `gh` already
  does the rerun and the log-read; the web-UI scroll-jack and buried dropdown are *UI*
  pains `gh` itself already escaped. ghx's value is narrow: collapse *check-name-on-this-
  PR* → action in one step, without hand-mapping check → run-id. Real, but ergonomics.
- **The genuine unbuilt *capability* frontier is the writes + the diff overlay** —
  resolve/unresolve, inline review composition, diff-with-threads. Heavier, partly
  agent-territory. This is the honest "40% the UI does and `gh` can't."
- **ghx already captured the *cheap* bucket-A gaps** — threads-with-state (read) and the
  merge verdict. What's left is ergonomics polish or genuinely heavy/write work.
- **The honest merge gate is a synthesis job, not a data-access one.** Everything the
  gate needs is reachable today; `gh` just never unions it into a verdict. This was the
  highest-value real work — now shipped as `ghx gate`.

## Priority (re-derived from the survey)

1. **Honest merge gate.** ✅ Shipped — `ghx gate`, anchored on GitHub's merge-button state.
   This was the single best thing on the board.
2. **Writes story.** Real bucket-A capability gap. Split by blast radius:
   - *Operational* writes — resolve/unresolve. ✅ Shipped.
   - *Content* writes — inline / multi-comment review composition. Heavier; largely
     agent-driven. Still open.
3. **`ghx rerun` / `ghx logs` — ergonomics quick wins.** Bucket-B, labeled as
   convenience over `gh`, not gap-filling. Cheap, pleasant, not differentiation.
4. **Diff-with-threads overlay.** Real but heavy bucket-A; defer until the cheaper gaps
   are closed and there's a clear design.
5. **Leave to `gh`.** Everything in bucket C. Not ghx's job.

## What's left

- **Content writes** — inline / multi-comment review composition (bucket A, agent-driven,
  heavier).
- **`ghx rerun` + `ghx logs`** — bucket-B ergonomics, explicitly *not* sold as filling a
  `gh` gap.
- **Diff-with-threads overlay** — the heavy bucket-A frontier.
