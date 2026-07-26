---
id: 01KYCYCA4CY9FJE3TKVHDTH4VM
type: handoff
phase: token-estimation
lane: high-risk
run_id: 01KYCVPE9326H17R0KV7DHC2Z4
check_id: 01KYCVQEX866578AD3EZAHY6FK
created: 2026-07-25
updated: 2026-07-25
session-date: 2026-07-25
branch: feat/credential-concurrency-token-estimation
status: merged
continuity-mode: full-harness
active-phase: token-estimation
last-updated: 2026-07-25 22:30
---

# Session Handoff — feat/credential-concurrency-token-estimation

## Current State

**Branch**: `feat/credential-concurrency-token-estimation` tracking `origin/…` (same name), fully pushed, working tree clean
**Status**: merged — PR #2 merged into `master` at 2026-07-25 15:26 UTC, merge commit `566d5e7`
**Continuity Mode**: full-harness (roadmap + phase + run + check all present; ROADMAP content is stale, see Proof / Drift Notes)
**Active Phase**: `token-estimation` (gated); sibling `auth-credential-concurrency` also gated in the same commit
**Last Commit**: `f6294be` — `feat(auth,executor): port credential concurrency and token estimation from CLIProxyAPI v7.2.96`

**Working Tree**:
- 0 staged files
- 0 modified files
- 0 untracked files
- `master` locally is 2 commits behind `origin/master` — the merge has not been pulled down; the merged feature branch still exists locally and on the remote

## What We're Building

A selective backport of CLIProxyAPI `v7.2.95`/`v7.2.96` behavior into llmhub, split
into two phases that landed together. **`auth-credential-concurrency`** adds
Home-dispatched credential accounting with exactly-once lifecycle release, bounded
drain, and in-flight observation — new config primitives (`internal/config/credential_*.go`),
a new `sdk/cliproxy/executionregistry` package, Home dispatch/release/observation
contracts under `sdk/cliproxy/auth/`, and lifecycle binding threaded through nine
executors and `sdk/api/handlers`. **`token-estimation`** adds Claude input-token
estimation that patches only the first SSE `message_start` event when upstream
omits `usage.input_tokens`, plus xAI input counting and a tool-setting allocation
optimization in the OpenAI-Responses translator; it bumps
`github.com/tiktoken-go/tokenizer` to `v0.8.1`.

Both phases were already implemented and uncommitted when this session began. This
session's work was **not** implementation — it was correcting a lying run log,
gating both phases, repairing the harness artifact chain so verdicts persist, and
shipping the result.

## Continuity Anchors

**Latest Cook Run**: `.kit/runs/work/20260725-1210-token-estimation.md` (ULID `01KYCVPE9326H17R0KV7DHC2Z4`)
**Sibling Cook Run**: `.kit/runs/work/20260723-2210-auth-credential-concurrency.md` (ULID `01KYCVPE8V5CP8TWVN3FX5JEEK`)
**Latest Check Verdict**: approve-with-requests (ULID `01KYCVQEX866578AD3EZAHY6FK`, `.kit/reports/check/20260725-1215-token-estimation.md`)
**Sibling Check Verdict**: approve-with-requests (ULID `01KYCVQEX1N4W28X5SFB751THG`, `.kit/reports/check/20260725-1200-auth-credential-concurrency.md`)

**Proof / Drift Notes**:
- `zharness resume --json` → `readiness: clean`, `pointer_drift: []`, `unlinked_proofs: []`
- `zharness audit --json` → **3 contract_violations remain, all zharness-side gaps, none is an artifact error** (see Blockers)
- **Scope drift, accepted by the user, not resolved**: both phases were committed as one unit. Each phase's CONTEXT forbids the other's surfaces (`token-estimation` explicitly forbids "auth selection and credential concurrency"; `auth-credential-concurrency` does not allow `internal/translator/`). Both are individually green, so behavior is not wrong — but neither phase's boundary is independently auditable from `f6294be`. Recorded as a Major on both check reports and in the commit's `Known follow-ups`.
- **RUN `20260725-1210-token-estimation.md` is explicitly labeled reconstructed** — see its `## Provenance` section. The implementation predates it; every `verification:` line is a command actually run on 2026-07-25 against existing code, but no implementation step is claimed as performed by that run. Do not read it as a live `work` transcript.
- `.kit/planning/ROADMAP.md` is **stale**: it names `model-routing`, `websocket-pluginhost`, and `docs-frontend`, none of which exist on disk. It cannot currently be used to pick the next phase.
- `staticcheck` is not installed, so that gate stage has **no evidence rather than a pass** on both reports.

## Progress This Session

### Completed ✓
- Corrected `.kit/runs/work/20260723-2210-auth-credential-concurrency.md` — W2 claimed `status: running` / `changed files: none` while the code existed and passed; rewrote to DONE with the real file list and added W3.
- Wrote `.kit/runs/work/20260725-1210-token-estimation.md` for the already-implemented phase, labeled reconstructed.
- Gated both phases at `lane: high-risk` — all four required proof classes (unit, integration, manual-check, command-output) evidenced; matrix PASS on both.
- Root-caused **11 harness contract violations to one cause: the harness DB was entirely empty** (`zharness query phases` → `[]`, `query state` → all null). `.kit/` markdown had never been imported, which also explains why an earlier watzup rendered `readiness: no-harness` while `resume --json` said `clean`. Fixed by registering entities: `story` ×2 → `run create` ×2 → `check record` ×2. Violations 11 → 3.
- Added `---` YAML frontmatter fences to `.kit/planning/SPEC.md` (confirmed missing via `od -c` — the file started directly with `id:`) and both RUN artifacts.
- Stripped trailing-whitespace markdown line breaks from `.kit/planning/phases/websocket-message-too-big/{CONTEXT,PLAN}.md` that were failing `git diff --check`.
- Rewrote `.kit/workflow-state.yml` — `latest_check_report` was stale at `none`.
- Committed as `f6294be` (362 files, +9001/−36871: 49 A, 284 D, 28 M, 1 R060), pushed the branch, opened and merged PR #2.

Final gate on `f6294be`: `go test ./...` exit 0 (58 packages) · `go vet ./...` exit 0 · `go build ./...` exit 0 · `git diff --check` clean.

### In Progress ⏳
- Nothing. Working tree clean, PR merged, no work mid-flight.

### Not Started
- The two non-blocking code follow-ups below (`closeErr`, typeless content blocks).
- Refreshing `.kit/planning/ROADMAP.md` to match the phases actually on disk.
- Advancing the stale `Status: ready` field on the 7 PLAN files (see correction below).

> **Correction — the other 5 phases are already shipped, not pending.**
> An earlier draft of this handoff listed `auth-count-reliability`,
> `model-display-compatibility`, `tool-protocol-parity`,
> `translator-content-fidelity`, and `websocket-message-too-big` as unexecuted
> because their PLAN files read `Status: ready`. That field is stale. Their code
> landed in commit `e4bf0f7` (333 files, +44513/−1042) as the `US-016` story —
> verified by file presence in that commit:
> `internal/runtime/executor/codex_websockets_executor.go` and
> `sdk/api/handlers/openai/openai_responses_websocket.go` (websocket close-1009),
> `sdk/cliproxy/auth/conductor.go` + `conductor_count_tokens_test.go` +
> `cooldown_backoff_test.go` (auth-count-reliability),
> `internal/translator/common/file_data.go` (translator-content-fidelity),
> `internal/translator/openai/openai/responses/openai_openai-responses_tools.go`
> (tool-protocol-parity), and commits `8883d51`/`6abda44` (model display names).
> **Do not re-implement them.** The only real work left on those phases is
> flipping the stale `Status:` field. The handoff entity's `open_items` still
> carries the incorrect claim — this markdown is authoritative.
>
> Those 5 belong to the **older `US-016` SPEC** (`v7.2.49`→`v7.2.93`), not to the
> current one — their PLANs are all stamped `Updated At: 2026-07-22`, while this
> SPEC's two phases are stamped `2026-07-23`. Phase directories on disk are a mix
> of two initiatives; do not read `ls .kit/planning/phases/` as this SPEC's scope.

### SPEC coverage — 2 of 5 ROADMAP phases done, NOT complete

`.kit/planning/ROADMAP.md` is the authority for SPEC `01KY1T7GGY3DBEW2NAAV620VCK`
("all post-v7.2.93 updates", `v7.2.93`→`v7.2.96`). It declares 5 phases. Verified
status of each:

| ROADMAP phase | Status | Evidence |
|---|---|---|
| `auth-credential-concurrency` | ✅ done | PR #2, gated `01KYCVQEX1N4W28X5SFB751THG` |
| `token-estimation` | ✅ done | PR #2, gated `01KYCVQEX866578AD3EZAHY6FK` |
| `model-routing` — Codex Alpha Search + new models + error handling | ❌ **not started** | no phase directory; no PLAN; **no Codex Alpha Search anywhere in code** (searched `internal/registry/`, `internal/runtime/executor/`, `sdk/api/handlers/openai/` for `alpha.search`/`alpha_search`/`codex-alpha` — only unrelated `alpha` test fixtures in `claude_executor_test.go`) |
| `websocket-pluginhost` — WebSocket 1009 + race fixes + pluginhost | ⚠️ **partial** | the 1009 half shipped via `US-016` (`codex_websockets_executor.go`, `openai_responses_websocket.go` in `e4bf0f7`); the **pluginhost half is absent by decision** — `sdk/pluginabi`, `sdk/pluginapi`, `sdk/pluginhost`, `sdk/pluginstore` do not exist, deferred as Slice 5 in `plans/cliproxyapi-upstream-parity-2026-07-02.md` pending its own threat model |
| `docs-frontend` — AIUsage showcase + management panel UI + changelog + Docker | ❌ **not started** | no phase directory; no `CHANGELOG.md` in repo root; every recent `web/` commit is llmhub's own UI work (branding, quota cards, sonner) with no upstream AIUsage showcase port |
| `quota-usage-reset` — quota/usage reset controls in the panel | ❌ **not started** | **added to the SPEC on 2026-07-25** at the user's request; see the SPEC Amendment. llmhub-custom, the single approved YAGNI carve-out. W1 = wire existing `POST /v0/management/reset-quota` (frontend only); W2 = new usage-statistics reset endpoint + UI. Exempt from the frontend-last ordering; W2 must not block W1. |

**Earlier statements that this SPEC was fully covered were wrong** — they were
inferred from the 7 phase directories on disk rather than from the ROADMAP.
The ROADMAP is not stale: it is this SPEC's real plan, and 3 of its 5 phases were
never planned into phase directories at all.

Correction to an earlier note in this handoff: `.kit/planning/ROADMAP.md` should
**not** be rewritten to match `ls .kit/planning/phases/`. Doing so would delete
the record of the 3 unstarted phases and make the SPEC look complete.

**SPEC Validation Expectations — verified 2026-07-25:**
- `go test ./...` → pass (58 packages, exit 0)
- `make build-web` → pass (exit 0; tsc + vite, 2484 modules, 1,914.91 kB single-file bundle)
- `make build` → pass (exit 0; embed + `go build`, binary `llmhub` at `v0.0.20-6-gf6294be-dirty`; working tree unchanged — `management.html` and `llmhub` are untracked)
- "Parity report updated with new evidence" → **not done**; `plans/cliproxyapi-upstream-parity-2026-07-02.md` still ends at its `Implementation Status - 2026-07-02` section and records nothing from `US-016` or PR #2
- "Frontend UI matches new model display/name changes" → **unverified**; no browser or API smoke was run against the panel, which the parity plan explicitly requires for web-visible changes ("Static Go tests are not enough for those surfaces")

## Key Decisions

1. **Both phases committed as one commit, not split** — the user explicitly chose this ("gộp luôn đi cho đỡ tốn tokens") as an informed acceptance of the scope-drift Major on both gate reports. The cost is that neither phase's surface boundary is independently auditable from the merge commit. Do not re-litigate this; it was decided with the finding in hand.
2. **`plan_id: none` left as-is rather than given a fake ULID** — `sqlite3 .kit/harness.db ".tables"` shows `checks, handoffs, intakes, meta, interventions, runs, stories, traces`: there is **no `plans` table** and no `plan create` command. An honest `none` was judged better than a valid-looking pointer to a nonexistent row. This is why 2 of the 3 remaining violations cannot be closed locally.
3. **The DB is the source of truth for ULIDs, not the files** — `zharness run create` mints its own ULIDs, which differed from the ones hand-assigned to the RUN files (`01KYCVPE8V5CP8TWVN3FX5JEEK` vs. the hand-written `01KYCVMSS85TA14JCZZKFJ36W3`). All 4 affected files were synced to the DB-issued values. **This is a quiet trap**: `audit` stays green while file and DB diverge. Always take the ULID the CLI returns.
4. **`scripts/schema/*.sql` deletions do not violate the "database schema" forbidden surface** — verified via `git show HEAD:scripts/schema/001-init.sql` → header reads "Harness v0 schema". Those are zharness's own SQLite schema, not llmhub's DB.
5. **Two suspected defects were investigated and deliberately not recorded as findings** — a suspected conn leak at `codex_websockets_executor.go:193` (both callers route through `invalidateUpstreamConn`, which closes at 1705-1709) and a suspected missing test (the real name is `TestStopHomeLifetimeDetachesDrainsAndFlushes`). Both are logged under "Verified clean" on the auth report. Do not re-raise them.

## Blockers & Issues

No blockers on shipping — the work is merged. Three unresolved items, none blocking:

### 3 harness contract violations — tool gaps, not artifact errors
- **Issue**: `zharness audit --json` reports (a) `SPEC->PLAN: not_yet_implemented — PLAN artifacts don't carry a spec_id field yet`, zharness self-reporting its own gap; and (b,c) `plan_id value "none" is not a valid ULID` on both RUN artifacts, because the schema has no `plans` table and no command registers a PLAN entity.
- **Needed**: an upstream zharness change — `spec_id` on PLAN artifacts, and a `plans` table plus `plan create`.
- **Next**: report upstream. Do **not** patch by minting ULIDs locally; that trades a visible gap for an invisible lie.

### `Registry.Close()` can never report failure
- **Issue**: `sdk/cliproxy/executionregistry/registry.go:50` declares `closeErr error`. It is read at lines 406 and 437 but **never assigned anywhere in the package**, so `Close()` always returns `nil` and callers cannot distinguish clean shutdown from a failed resource close (individual failures are only logged, in `closeResource`).
- **Needed**: either assign `closeErr` from the close path, or drop the field and change `Close()` to return no error.
- **Next**: pick one — the misleading contract is the whole problem, so either fix closes it.

### Typeless Claude content blocks bypass media exclusion
- **Issue**: `internal/runtime/executor/helps/claude_input_tokens.go:199-200` — the `case "":` fallback serializes an entire content object as compacted JSON. The explicit media branch at line 197 (`image`, `input_audio`, `audio`, `video`, `redacted_thinking`) is only reached when `type` is present, so a block lacking `type` but carrying base64 payload would be counted in full, contradicting the locked decision to exclude multimedia.
- **Needed**: skip objects whose serialized form exceeds a size threshold, or allow-list keys in the fallback.
- **Next**: defensive hardening only — the Claude API always supplies `type`, so this is not a live defect. Low priority.

## Technical Context

**Approach**: per-provider executors bind an execution lifecycle; `executionregistry.Scope` guarantees exactly-once release. The invariant to preserve: **release strictly follows resource close.** `Scope.EndWithRelease` (`registry.go:271`) wraps its whole body in `s.ended.Do`, calls `waitForBoundResourceClose()` at line 284, and only then `markReleasedLocked` at line 287 — release can neither precede close nor happen twice. Lock ordering is always `registry.mu` before `scope.mu`, and the release sink is invoked with no registry mutex held. **Any change here must keep that ordering.**

Token estimation patches the SSE stream in place: it strips a trailing `\r` for parsing only (lines 297-299) and splices the rewritten payload back into its exact byte span (330-336), so CRLF/LF framing and non-target events stay byte-identical. It never overwrites a non-zero upstream `input_tokens` (313-316), and a zero estimate or a count error leaves the chunk untouched.

**Key Files**:
- `sdk/cliproxy/executionregistry/registry.go` — exactly-once release core (447 lines); also holds the dead `closeErr`
- `sdk/cliproxy/executor/lifecycle.go` — Bind-failure cleanup via `errors.Join(errBind, closeResource())`
- `sdk/cliproxy/auth/home_{dispatch,concurrency,selection,session,in_flight_publisher}.go` — Home accounting contracts
- `internal/home/concurrency_release.go` — wire contract for release
- `internal/runtime/executor/helps/claude_input_tokens.go` — token estimation (369 lines); `state.codec` is an **intentional test seam**, not dead state (`claude_input_tokens_test.go:336` assigns `failingClaudeInputCodec{}`)
- `internal/runtime/executor/codex_websockets_executor.go` — `bindExecutionLifecycle` returns `errBind` without closing `conn`, but both callers (452, 690) route through `invalidateUpstreamConn`

**Dependencies**:
- `github.com/tiktoken-go/tokenizer v0.8.1` — O200k codec for semantic token counting; init guarded by `claudeInputTokenizerOnce`

**Configuration**:
- `GOCACHE=/private/tmp/llmhub-go-cache` was used for all gate commands this session
- `LLMHUB_POSTGRES_TEST_DSN` unset → `internal/store` has 2 tests that skip. Not blocking (the phase does not touch `internal/store`) but a `high-risk` lane whose locked decisions concern Postgres auth persistence would be better served by running that suite once with a DSN set.
- `staticcheck` not installed

## Next Steps

1. **→ START HERE: sync `master` and retire the merged branch** — `git checkout master && git pull` (2 commits behind), then delete `feat/credential-concurrency-token-estimation` locally and on the remote. Expected: `master` at `566d5e7`, no stale merged branch. This is the only thing standing between now and starting clean work.
2. **Advance both gated phases' `Status:` field from `ready` to done** — `.kit/planning/phases/{auth-credential-concurrency,token-estimation}/*-PLAN.md` still read `Status: ready` despite being gated and merged, so a future selection pass would re-pick them. The 5 `US-016` PLANs have the same stale field.
3. ~~Refresh ROADMAP to match the phase directories~~ — **do not do this.** An earlier draft of this handoff recommended it. It is wrong: ROADMAP is this SPEC's real plan, and rewriting it to match `ls .kit/planning/phases/` would erase the record of the phases never started. `.kit/planning/ROADMAP.md` was updated on 2026-07-25 to carry per-phase status instead.
4. **Run `/to-plan` for `quota-usage-reset`** — the newest phase and the only one that ships independently of everything else. W1 is small and self-contained: add `web/src/services/api/quota.ts` calling `POST /v0/management/reset-quota` with a stable `auth_index`, then surface the action on `QuotaCard` (which today exposes only `onRefresh` at line 149). Plan W2 (new usage-reset backend) in the same PLAN but as a separate wave so W1 can ship first.
5. **Plan the 3 remaining ROADMAP phases — the SPEC is NOT fully covered.** `.kit/planning/ROADMAP.md` declares 6 phases for SPEC `01KY1T7GGY3DBEW2NAAV620VCK`; only 2 are done. See "SPEC coverage" below. `model-routing` is next by the dependency rule (executor/translator tier), then `docs-frontend`. `websocket-pluginhost` needs a scope decision before planning, because its pluginhost half is a separately-gated high-risk initiative.
6. **Close the two code follow-ups** — `closeErr` first (a misleading API contract), then the typeless-content-block hardening (defensive only).

## Notes

- Stale PLAN surfaces worth fixing when either phase's PLAN is next touched: `internal/homeplugins/` (auth W2 `touches`) **does not exist in llmhub**; `internal/api/` (auth W3) required no change because lifecycle binding lands in `sdk/api/handlers` and the executors; and token-estimation PLAN W2 step 4 names `internal/runtime/executor/xai_executor_test.go` while the real file is `xai_token_count_test.go`. All three are documented in the RUN logs, but the PLANs themselves would mislead a future `to-plan` refresh.
- ULID alphabet is Crockford base32, 26 chars, `0123456789ABCDEFGHJKMNPQRSTVWXYZ` — **no I, L, O, or U**. A hand-written ID containing `U` was rejected by `audit` this session. Generate them, don't type them.
- `zharness resume --facts` rejects certain phrases in `facts.risks[].action` — `"git status"` was rejected with `resume: facts.risks[0].action contains forbidden phrase "git status"`. Reword rather than fight it.
- Prior handoff context that still holds: PR #1 (`feat/model-display-name-propagation`) was closed without merge and its remote branch deleted, though the closed PR record may remain cached on GitHub. The remote branch `origin/feat/model-display-name-propagation` is still visible in `git log --decorate`.

---

*Generated by handoff on 2026-07-25 22:30*
