# ghx — north star

> **ghx closes the PR loop inside the terminal, so neither you nor your agents
> ever have to open the GitHub web UI.** The agent drives it for the mechanical
> work; you read it when you want to *see*; and it always bottoms out in the
> terminal, never at a browser tab.

Pitch line: **"Never open the PR page again."**

## Who this is for

- **For:** people who live in the terminal and want PR review (reading threads,
  checking CI, rerunning a flaky job, knowing if it can merge) to happen *there*,
  whether they drive it by hand or hand it to an in-terminal agent.
- **Not really for:** people happy in the GitHub web UI or a heavy IDE PR
  experience. ghx isn't trying to win you over; the web UI is a good product. ghx
  is for the people it slows down.

You don't need to use agents to get value here, but if you do, the same tool is
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
respond to comments), and the human stays in the loop for judgment, *without*
either of them being dragged to a browser tab to do it.

## The moat (why this is durable)

`gh` gets maybe 60% of the way to "you never need the web UI," and a focused
third-party tool can take the rest, because **focus out-curates breadth.** `gh` is
GitHub's official, general-purpose surface: it optimizes for completeness and
neutrality across every workflow. It is not going to ship the opinionated,
synthesized, friction-killing *terminal PR-review workflow* (the curated defaults,
the one-glance verdict, the rerun-without-the-dance) because that's a strong
product opinion a neutral official CLI won't take. (There's a business angle too:
the web UI is where GitHub's engagement and Copilot surface live. But you don't
need to impute motive; breadth simply doesn't optimize the same thing focus does.)

So the gap isn't a feature `gh` is about to close; it's the difference between a
broad official tool and an opinionated one. That difference is where ghx lives.

## Two legs, one loop

The loop closes inside the terminal on both sides:

- **Agent leg — the primitive.** A synthesized, stable, structured contract the
  agent gates on: `--json`, exit codes, the `gate` verdict. The agent re-deriving
  raw GraphQL ad hoc *drifts*: sometimes it misses bots, sometimes it gets inline
  threads but not the conversation view. A tool gives the same complete answer
  every time. ghx is the determinism layer the agent drives.
- **Human leg — the escape hatch.** When you just want to *see* the PR: a complete,
  deterministic, readable view that bottoms out in the terminal, not at "...now go
  look in the browser." This leg is **not** subordinate to the primitive; it's
  co-equal, because the entire mission is *the loop never leaves the terminal*. An
  escape hatch that dumps you in the browser defeats the point.

That determinism is the answer to "why not just let the agent run gh/graphql ad
hoc?" Freehand prompting drifts; the tool is the same stable contract every time,
for both consumers.

## The design method

Don't chase "feature parity with the web UI"; chase the paper cuts. Enumerate
**every reason you currently open the GitHub PR / Actions page**, and each one is a
candidate ghx verb or view. That list *is* the map of gh's missing 40%, grounded in
real felt pain rather than a market guess.

The bar is **not 100%**: the asymptote costs more than it returns. Success is
opening the GitHub web UI **at most about once a week**: win the daily paper cuts,
leave the rare admin to the browser. The working inventory and its prioritization
live in [`roadmap.md`](./roadmap.md).

## Proposing a feature (the on-mission test)

Contributions and ideas are welcome, and there's a simple test for whether
something belongs here:

> Does this let a terminal-native developer (or their agent) do something they
> currently open the GitHub web UI for, something a broad official CLI is unlikely
> to curate itself?

If yes, it's on-mission: open an issue and make the case in those terms. If it's
just "a nicer way to see something gh already shows well," it's probably not.
