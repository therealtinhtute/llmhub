# Documentation Map

This directory holds the workflow contract used by agents working in this repo,
plus the durable records that work produces.

## Main Files

- `WORKFLOW.md`: the workflow contract. Read this first — `AGENTS.md` points
  every agent here before it does anything else.
- `playbooks/`: one playbook per workflow stage. `zharness preflight <stage>`
  returns the path of the one to read; do not read the others.

## Folders

- `plans/`: executable plans. `plans/active/` is in flight, `plans/done/` is
  finished work kept for reference.
- `stories/`: work packets derived from plans.
- `decisions/`: durable decisions and tradeoffs (ADRs). Append-only.
- `product/`: product truth derived from a spec.
- `upstream/`: notes on tracking the upstream project.
- `templates/`: reusable `decision.md` and `story.md` formats.

## Harness

Lifecycle state lives in the `zharness` database outside this repo, not in these
files. Repository docs, code, tests, and observable behavior are authoritative;
the database is a lifecycle ledger and recovery index.
