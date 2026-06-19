# ghx — roadmap

A living, internal prioritization pass. The *why* behind it lives in
[`vision.md`](./vision.md); this is the *what next*. It applies the vision's
design method — the paper-cut inventory — and sorts the surface practically.
Ticket IDs (`TON-…`) are internal Linear references.

## The bar

Not 100% coverage of the GitHub web UI — the asymptote costs more than it returns.
Target: **open the web UI at most about once a week.** Win the daily paper cuts;
leave the rare admin to the browser.

## The 40% gap, prioritized

### Tier 1 — the daily paper cuts (highest leverage)

| Paper cut ("I open the UI to…") | ghx today | Work |
| --- | --- | --- |
| Read inline threads + conversation + review decision together | ✅ `ghx comments` | done |
| See CI status / what failed | ✅ `ghx checks` | done |
| Know if it can merge / what's blocking | ⚠️ `ghx gate` (verdict is a heuristic) | **TON-49** — make it honest vs real merge rules |
| **Rerun a flaky / failed check** without the dropdown dance | ❌ | **new:** `ghx rerun` (operationalize `gh run rerun --failed`) |
| **Read a failing check's log** without the scroll-jack | ❌ | **new:** `ghx logs <check>` / `ghx checks --logs` |

### Tier 2 — frequent, but partly gh-covered or heavier

| Paper cut | ghx today | Work |
| --- | --- | --- |
| See the diff / files changed | ❌ (`gh pr diff` exists, ~ok) | evaluate: does ghx add value (diff + threads inline)? else defer |
| Resolve / unresolve a thread (operational write) | ❌ | **TON-52** (operational-write half) |
| See requested reviewers / decision rollup detail | partial (`gate` has decision) | small surfacing add |

### Tier 3 — rare; leave to gh/browser (does not count against the bar)

- Merge the PR — `gh pr merge` from the terminal is *not* "opening the UI." Fine.
- Edit PR metadata, labels, linked issues — rare, gh covers it.
- Submit review content / replies — the **agent** drives these (or **TON-52**'s
  content-write half). Not native-ghx-urgent.

## Implications for the backlog

- Split **TON-52** in two: *operational* writes (rerun, resolve — high paper-cut
  value, near-term) vs *content* writes (reviews/replies — agent-driven, lower
  native priority).
- File two **new** Tier-1 candidates not yet ticketed: `ghx rerun`, `ghx logs`.
- **TON-49** is reframed from "credibility nice-to-have" to Tier-1 daily-path work.
- The README's current "views gh leaves out" framing undersells the vision; reframe
  it around the north star at go-public (**TON-46**).
