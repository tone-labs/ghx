# ghx — north star

> **ghx closes the PR loop inside the terminal — so neither you nor your agents
> ever have to open the GitHub web UI.** The agent drives it for the mechanical
> work; you read it when you want to *see*; and it always bottoms out in the
> terminal, never at a browser tab.

Pitch line: **"Never open the PR page again."** (Aspirational — see the v1 bar
below for the honest target.)

## Who this is for

- **For:** people who live in the terminal and want PR review — reading threads,
  checking CI, rerunning a flaky job, knowing if it can merge — to happen *there*,
  whether they drive it by hand or hand it to an in-terminal agent.
- **Not really for:** people happy in the GitHub web UI or a heavy IDE PR
  experience. ghx isn't trying to win you over; the web UI is a good product. ghx
  is for the people it slows down.

You don't need to use agents to get value here — but if you do, the same tool is
the stable contract they drive. Both audiences, one loop.

## The pain (why this exists)

The web GitHub UI is good, but for someone living in the terminal it's a pile of
paper cuts:

- Rerunning a flaky check is a dropdown buried several clicks deep.
- The Actions log page scroll-jacks you to the bottom before you can hit "rerun."
- Seeing inline threads *and* the conversation *and* the review decision together
  means three different places.

Each one is small. They add up, and they pull you out of flow. ghx exists to
delete them.

## The bet

Humans are not leaving the terminal. Despite a steady stream of GUI wrappers and
heavy IDE-based agentic experiences, terminal-native agentic work (Claude Code et
al.) has only grown. The durable shape of PR work is a **hybrid**: the agent does
the mechanical reading and writing (survey checks, draft and submit reviews,
respond to comments), and the human stays in the loop for judgment — *without*
either of them being dragged to a browser tab to do it.

## The moat (why this is durable)

`gh` gets maybe 60% of the way to "you never need the web UI," and a focused
third-party tool can take the rest — because **focus out-curates breadth.** `gh` is
GitHub's official, general-purpose surface: it optimizes for completeness and
neutrality across every workflow. It is not going to ship the opinionated,
synthesized, friction-killing *terminal PR-review workflow* — the curated defaults,
the one-glance verdict, the rerun-without-the-dance — because that's a strong
product opinion a neutral official CLI won't take. (There's a business angle too —
the web UI is where GitHub's engagement and Copilot surface live — but you don't
need to impute motive: breadth simply doesn't optimize the same thing focus does.)

So the gap isn't a feature `gh` is about to close; it's the difference between a
broad official tool and an opinionated one. That difference is where ghx lives.
The missing 40% is not random — it's exactly the *workflow* surface a general CLI
under-curates. Which means **"find gh's missing 40%" and "find the UI-killer
features" are the same search.**

## Two legs, one loop

The loop closes inside the terminal on both sides:

- **Agent leg — the primitive.** A synthesized, stable, structured contract the
  agent gates on: `--json`, exit codes, the `gate` verdict. The agent re-deriving
  raw GraphQL ad hoc *drifts* — sometimes it misses bots, sometimes it gets inline
  threads but not the conversation view. A tool gives the same complete answer
  every time. ghx is the determinism layer the agent drives.
- **Human leg — the escape hatch.** When you just want to *see* the PR, a complete,
  deterministic, readable view — that bottoms out in the terminal, not at "...now go
  look in the browser." This leg is **not** subordinate to the primitive; it's
  co-equal, because the entire mission is *the loop never leaves the terminal*. An
  escape hatch that dumps you in the browser defeats the point.

**Determinism is the pillar that answers "why not just let the agent run gh/graphql
ad hoc?"** Because freehand prompting drifts and the human view it produces is
different each time. The tool is the stable contract for both consumers.

## The design method: a paper-cut inventory

Don't chase "feature parity with the web UI" — chase the paper cuts. Enumerate
**every reason you currently open the GitHub PR / Actions page**, and each one is a
candidate ghx verb or view. That list *is* the map of gh's missing 40%, and it's
grounded in real felt pain rather than a market guess.

## The 40% gap, prioritized (first pass)

The v1 bar is **not 100%** — the asymptote costs more than it returns. The target:
**you open the GitHub web UI at most about once a week.** Get the daily paper cuts
and the bar is met; leave the rare admin stuff to the browser.

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

### What this implies for the backlog

- Splits **TON-52** in two: *operational* writes (rerun, resolve — high paper-cut
  value, near-term) vs *content* writes (reviews/replies — agent-driven, lower
  native priority).
- Surfaces two **new** Tier-1 candidates not yet ticketed: `ghx rerun`, `ghx logs`.
- Reframes **TON-49** from "credibility nice-to-have" to Tier-1 daily-path work.
- The README's current "views gh leaves out" framing undersells this; reframe it
  around the north star at go-public (with **TON-46**).

## Proposing a feature (the on-mission test)

Contributions and ideas are welcome — and there's a simple test for whether
something belongs here:

> Does this let a terminal-native developer (or their agent) do something they
> currently open the GitHub web UI for — something a broad official CLI is unlikely
> to curate itself?

If yes, it's on-mission — open an issue and make the case in those terms. If it's
just "a nicer way to see something gh already shows well," it's probably not.
