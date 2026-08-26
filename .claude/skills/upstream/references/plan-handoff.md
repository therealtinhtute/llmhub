# Plan Handoff

Phase 4 converts approved scope into a locked initiative. This skill owns none of the
plan's later sections — `brainstorm` locks it, `to-plan` decomposes it, `work` builds it.

## Preconditions

1. Every ledger row has a disposition. A blank row means triage is not finished.
2. `scope_policy.include` and `.exclude` match what the user approved.
3. `zharness preflight brainstorm --mode lock --json` reports durable readiness. Reduced
   mode cannot lock — stop and report the recovery it returns.

## Invocation

Invoke `brainstorm` in `lock-from-files` mode, naming these as authoritative sources:

- `docs/upstream/{slug}-checkpoint.json` — checkpoint, baseline, scope policy
- `docs/upstream/{slug}-gap-{from}..{to}.json` — structural counts
- `docs/upstream/{slug}-ledger-{from}..{to}.md` — per-commit dispositions
- `CLAUDE.md` — the invariants that shaped every `reject`

Slug the initiative after the upstream and checkpoint: `{slug}-{tag}-parity`.

## Requirement shape

Every requirement is numbered, falsifiable, and cites its upstream source.

```
R7 [accepted]: Implement the full upstream Codex Live capability set — session handling,
sideband protocol, realtime media relay, TCP proxy, configuration propagation, lifecycle
shutdown, and deterministic tests — adapted to llmhub service startup and Postgres
configuration. | source: upstream release v7.2.99
```

- One requirement per capability slice, not per commit.
- "adapted behind existing interfaces" beats "port file X" — requirements describe behavior.
- Every `exclude` becomes a **Non-goal** with its invariant named. Exclusions that vanish
  from the plan reappear as scope creep during `work`.
- Requirements that preserve local behavior are as necessary as those that add upstream
  behavior. State them.

## Mandatory final-gate requirements

Two requirements must always be present, because upstream keeps moving during
implementation:

```
R{n} [accepted]: The final gate must re-resolve the latest stable upstream release, fetch
its commit into refs/upstream-checkpoints/{slug}/{tag}, and update the checkpoint and
ledger before closure. | source: checkpoint integrity

R{n+1} [accepted]: If the latest stable upstream release changes after planning,
implementation must not silently widen scope; the ledger must identify the delta and either
include it through an approved plan refinement or pin the completed scope with an explicit
follow-up. | source: targeted-scope policy
```

Without these, "parity with v7.2.112" quietly becomes a false claim the moment v7.2.113
ships.

## Verification contract to carry into the plan

Name the checks the plan must pass, so `check` has something falsifiable to run:

- `go test ./...` plus focused package tests for each touched subsystem
- `make build` (includes the embedded web asset path)
- Frontend: type check, lint, production build, browser runtime check — **no new test files
  under `web/`** per `CLAUDE.md`
- `git diff --check`
- JSON validation for any registry or model definition change

## Exit

Report the plan path, the locked include/exclude, and `to-plan` as the next action. Do not
write phases, tasks, or code — that authority belongs to the next stage.

## Handoff failure modes

- **Locking before triage finishes.** Blank dispositions become "figure it out during work",
  which is how a targeted port turns into a wholesale merge.
- **Citing the branch instead of the checkpoint ref.** `upstream/main` moves; the plan
  becomes unreproducible. Cite `refs/upstream-checkpoints/{slug}/{tag}`.
- **Dropping the exclusions.** If Non-goals do not list them, nothing prevents the excluded
  work from being done anyway.
- **One requirement per commit.** Produces dozens of unorderable requirements and phases
  that cannot be tested independently.

## Release note (llmhub)

- llmhub release series is `v0.0.x` (see `gh release list --repo therealtinhtute/llmhub`). The `v6.10.x` tags are stale local tags — never use them.
- Next version = `incpatch` of latest GitHub release: `gh api repos/therealtinhtute/llmhub/releases/latest --jq .tag_name` or `git ls-remote --tags origin | grep -E 'v0\.0\.[0-9]+' | sort -V | tail -1`. Never use `git tag --sort=-v:refname` alone — it picks the stale `v6` series.
- Tag must be annotated and pushed before release: `git tag -a v0.0.N -m "v0.0.N: ..." <commit> && git push origin v0.0.N`; workflow `.github/workflows/release.yml` triggers on `v*` and runs `goreleaser` with `ldflags -X 'main.Version={{.Version}}'`.
- Never create releases manually via `gh release create` — it bypasses `goreleaser` and produces a release with 0 assets. Always push the tag and let the workflow create the draft, or use `gh workflow run "Release" --ref master -f tag=v0.0.N` for an existing tag.
- Verify with `gh release view v0.0.N --repo therealtinhtute/llmhub --json tagName,assets` (expect 9 assets) and `git ls-remote --tags origin | grep v0.0.N`.
