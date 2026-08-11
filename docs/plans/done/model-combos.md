---
id: 01KZB35FTSNCC05VNPHYA4EXFC
type: plan
intake_id: 01KZB35Q16B28MX0TMEQKBGFC8
lane: normal
status: active
created: 2026-08-05
updated: 2026-08-06
---

# Plan D — Model combos (cross-provider fallback)

Status: **locked** · Created 2026-08-05 · Locked 2026-08-06 · Ships independently
Depends on: **Plan A** (`../done/error-classification.md`) — soft dependency, now done, see below
Source mechanism: `decolua/9router` `open-sse/services/combo.js:246` (`handleComboChat`)
Reference skill: `.claude/skills/9router-port/references/routing.md` §1

## Outcome

Ship cross-provider model combos: a named virtual model that expands to an ordered list of
`provider/model` candidates spanning different providers, with `fallback`/`round-robin` rotation,
so a client-visible model name can fall back across different providers' credentials — not just
one provider's, which is all LLMHub does today.

Success signals:
- Gate test pinning the `streamBootstrapError` streaming boundary passes before any routing code
  is written; if it cannot be made to pass, the design halves in scope (see `## Load-bearing
  assumption`).
- Candidate 1 fails `429`, candidate 2 returns `200`: client gets candidate 2's response.
- Candidate 1 fails `400`: returned to the client unmasked, candidate 2 never attempted.
- All candidates fail: HTTP `503` (not `406`) carrying the earliest reset across candidates plus a
  human-readable string.
- Streaming, candidate 1 fails at bootstrap (zero bytes flushed): switches silently to candidate 2.
- Streaming, candidate 1 fails after the first chunk: does not switch; the error reaches the client
  in-stream.
- `round-robin` with `sticky-limit: 3` advances the cursor on the 4th request, not the 2nd.
- `502` with a computed cooldown ≤ 5s pauses before the next candidate; a longer cooldown does not.
- A combo name colliding with a registered model id is rejected at config load.
- Config reload clears the in-memory rotation cursor.
- `go test ./sdk/cliproxy/... ./internal/config/... ./internal/registry/... ./internal/watcher/...`
  passes.

## Authority and Requirements

Authority:
- Source mechanism `decolua/9router` `open-sse/services/combo.js:246` (`handleComboChat`) and
  reference skill `.claude/skills/9router-port/references/routing.md` §1 — external reference
  project, cited for the combo/fallback/round-robin/exhaustion mechanism, not copied verbatim.
- Plan doc's own analysis of `Manager.ExecuteStream`'s existing `providers []string` parameter and
  its `streamBootstrapError` distinction (`sdk/cliproxy/auth/conductor.go:1673`, `:1705`) —
  authority for the load-bearing assumption that a failed candidate can only be replaced before any
  bytes have reached the client (see `## Load-bearing assumption`).
- Plan doc's own analysis of `shouldRetryAfterError` (`conductor.go:2621`) — authority for the soft
  dependency on Plan A (`error-classification`, now `done`): without it, a candidate failing with a
  non-429 retryable status ends the whole chain instead of advancing.

Requirements:
1. `R1` — Gate test first: `sdk/cliproxy/auth/conductor_stream_bootstrap_test.go` (new) asserts
   `streamBootstrapError` is returned only when zero chunks have been written downstream, and that
   a mid-stream upstream failure produces a different error type. Nothing else starts until this
   passes. | source: `## Changes` §1
2. `R2` — New `ComboConfig` (`Name`, `Strategy`, `StickyLimit`, `Models`) added to `Config` as
   `Combos []ComboConfig`, stored in Postgres and round-tripped through
   `/v0/management/config.yaml` like the rest of config (no new local file). Load-time validation:
   unique non-empty name not colliding with a registered model id, at least one candidate, each
   candidate parses as `provider/model`, `strategy` in `{"", "fallback", "round-robin"}`,
   `sticky-limit < 1` normalized to `1`. | source: `## Changes` §2
3. `R3` — Each combo name registered as a client-visible model (so `/v1/models` lists it and
   clients can select it), marked non-resolvable as a candidate itself (no recursion). | source:
   `## Changes` §3
4. `R4` — New `sdk/cliproxy/combo.go` `Resolver`: `fallback` always starts at index 0;
   `round-robin` keeps a per-combo cursor with a sticky limit before advancing (protects upstream
   prompt caching). Cursor state is in-memory, mutex-guarded, cleared on config reload. | source:
   `## Changes` §4
5. `R5` — Combo branch in the model-to-`providers []string` resolution path: iterate candidates,
   return on success; on a non-retryable error return it unmasked; on `502/503/504` with cooldown
   ≤5s, sleep before the next candidate; a streaming bootstrap failure switches, a post-flush
   failure does not. Existing per-candidate credential retry/cooldown/backoff run unmodified inside
   each iteration. | source: `## Changes` §5
6. `R6` — Exhaustion response: all candidates failed → HTTP `503` carrying the earliest reset across
   candidates (from `sdk/cliproxy/auth/cooldown_state.go`) plus a human string (e.g. `reset after 2m
   30s`). | source: `## Changes` §6
7. `R7` — Panel Combos page: list, create, reorder candidates, pick strategy and sticky limit. No
   new test files under `web/` (project rule). | source: `## Changes` §7

## Non-goals

- `NG1` — No change to credential selection, cooldown persistence, or backoff inside a single
  provider; that layer runs unmodified within each candidate attempt. | source: `## Not building`
- `NG2` — No capability-aware reordering; deliberately deferred to a separate later plan (Plan E).
  | source: `## Not building`
- `NG3` — No fusion / panel+judge behavior. | source: `## Not building`
- `NG4` — No new executor or translator; the combo layer sits above `Manager.ExecuteStream`. |
  source: `## Not building`

## Building

A **combo**: a named virtual model that expands to an ordered list of `provider/model` candidates
spanning different providers. The client asks for one name; the router walks the chain.

```yaml
combos:
  - name: daily
    strategy: fallback            # fallback | round-robin
    sticky-limit: 8               # round-robin only
    models:
      - claude/claude-opus-5
      - openrouter/deepseek-v4:free
      - opencode/grok-code
```

This is the one real capability gap. LLMHub falls back across **credentials of one provider**
(`sdk/cliproxy/auth/selector.go`, `scheduler.go`); it has nothing that falls back across
**models of different providers**.

## Not building

- No change to credential selection, cooldown persistence, or backoff inside a single provider —
  that layer stays exactly as-is and runs unmodified within each candidate attempt.
- No capability-aware reordering (that is Plan E, deliberately separate).
- No fusion / panel+judge.
- No new executor or translator. The combo layer sits **above** `Manager.ExecuteStream`.

## Load-bearing assumption

> **A failed candidate can only be replaced while nothing has been flushed to the client.**

Evidence it holds: `Manager.ExecuteStream` already takes `providers []string` and already
distinguishes `streamBootstrapError` — a failure raised before the first downstream byte
(`sdk/cliproxy/auth/conductor.go:1673`, `:1705`).

**If it does not hold** — if `streamBootstrapError` can be returned after bytes reached the client
— then cross-provider fallback is only safe for non-streaming requests, and streaming clients must
get an in-stream error event instead of a silent switch. That halves the feature's value and
changes the design.

**Therefore task 1 is the test that pins this, written before any routing code.** If it fails,
stop and re-scope rather than building on top of it.

## Soft dependency on Plan A

Combo needs to answer "is this failure worth advancing for?". Without Plan A the only signal is
`shouldRetryAfterError`, which returns false for every non-429 status
(`conductor.go:2621`) — so a candidate failing with `500 "rate limit exceeded"` would end the whole
chain instead of advancing. Combo *works* without A, it just advances far less often than it
should. Ship A first.

## Changes

### 1. Test first — pin the streaming boundary

`sdk/cliproxy/auth/conductor_stream_bootstrap_test.go` (new): assert `streamBootstrapError` is
returned only when zero chunks have been written downstream, and that a mid-stream upstream failure
produces a different error type. **Gate: if this cannot be made to pass, stop.**

### 2. Config

`internal/config/config.go`:

```go
type ComboConfig struct {
    Name        string   `yaml:"name" json:"name"`
    Strategy    string   `yaml:"strategy,omitempty" json:"strategy,omitempty"`       // fallback | round-robin
    StickyLimit int      `yaml:"sticky-limit,omitempty" json:"sticky-limit,omitempty"`
    Models      []string `yaml:"models" json:"models"`
}
```

Added to `Config` as `Combos []ComboConfig`. Validation at load (`internal/config/parse.go`):
name non-empty and unique; name must not collide with any registered model id; at least one
candidate; each candidate parses as `provider/model`; `strategy` in `{"", "fallback",
"round-robin"}` (empty = `fallback`); `sticky-limit < 1` normalized to `1`.

Stored in Postgres like the rest of the config and round-tripped through
`/v0/management/config.yaml` — no new local file (project `CLAUDE.md`).

### 3. Registry

`internal/registry/` — register each combo name as a client-visible model so `/v1/models` lists it
and clients can select it. Marked so it is never itself resolvable as a candidate (no recursion).

### 4. Resolution and rotation

`sdk/cliproxy/combo.go` (new), ported from `open-sse/services/combo.js:157-212`:

```go
type Candidate struct{ Provider, Model string }

func (r *Resolver) Resolve(name string) ([]Candidate, bool)
func (r *Resolver) Rotate(name, strategy string, stickyLimit int) []Candidate
```

- `fallback` — always start at index 0.
- `round-robin` — per-combo cursor plus a **sticky limit**: stay on the same candidate for N
  consecutive requests before advancing (`combo.js:174`). Sticky matters because rotating every
  single request destroys upstream prompt caching.
- Cursor state is an in-memory `map[string]state` guarded by a mutex, cleared on config reload via
  `internal/watcher/config_reload.go`.

### 5. The loop

In the request path that maps a model name to `providers []string` before calling
`Manager.Execute` / `Manager.ExecuteStream`, add a combo branch:

```
for each candidate:
    result := Manager.Execute[Stream](ctx, []string{cand.Provider}, req{Model: cand.Model}, opts)
    if err == nil                       → return
    if streaming && !streamBootstrapErr → return err   (bytes already flushed; cannot switch)
    if !retryable(err)                  → return err   (400-class: surface unmasked)
    if status in {502,503,504} && cooldown <= 5s → sleep(cooldown)
    continue
```

Two details ported verbatim in intent:

- **Transient pause** (`combo.js:311`): on `502/503/504` with a computed cooldown ≤ 5s, sleep it
  before the next candidate. A briefly-overloaded provider deserves a beat, not an instant skip.
- **Do not mask client errors** (`combo.js:303`): if the failure is not retryable, return it as-is.
  A `400` must not become a 503 after walking the whole chain.

Existing credential-level retry, cooldown, and backoff run **inside** each iteration, untouched.

### 6. Exhaustion response

Matching `combo.js:329-347`: all candidates failed → HTTP **503**, not 406. 406 implies the request
is invalid; here the providers are merely unavailable and the client should retry. Body carries the
**earliest** reset across all candidates (read from `sdk/cliproxy/auth/cooldown_state.go`) plus a
human string, e.g. `reset after 2m 30s` (`accountFallback.js:90` `formatRetryAfter`).

### 7. Panel

`web/` — a Combos page: list, create, reorder candidates, pick strategy and sticky limit. No new
test files under `web/` (project rule).

## Verify

- **Gate test from step 1 passes.** Nothing else starts until it does.
- `go test ./sdk/cliproxy/... ./internal/config/... ./internal/registry/... ./internal/watcher/...`
- New tests:
  - candidate 1 → `429`, candidate 2 → `200`: client gets candidate 2's response
  - candidate 1 → `400`: returned unmasked, candidate 2 never attempted
  - all candidates fail: `503` carrying the earliest reset and the human string
  - streaming, candidate 1 fails at bootstrap: switches silently
  - streaming, candidate 1 fails after the first chunk: does **not** switch, error reaches the
    client in-stream
  - `round-robin` with `sticky-limit: 3` advances the cursor on the 4th request, not the 2nd
  - `502` with a 3s cooldown pauses before the next candidate; `502` with a 40s cooldown does not
  - combo name colliding with a registered model id is rejected at config load
  - config reload clears the rotation cursor
- Live: define a 2-candidate combo, revoke the first provider's credential, confirm requests keep
  succeeding and the log shows the switch.

## Rollback

Remove the `combos:` block. With no combos configured the routing branch is never entered and the
request path is byte-identical to today. The registry entries disappear with the config.

## Risk

- The gate test is the real risk; everything downstream assumes it passes.
- Rotation state is per-process and in-memory. In a multi-replica deployment each replica rotates
  independently. Acceptable — 9router has the same property (`combo.js:88`) — but it must be
  documented, not discovered.

## Approach and Risks

Chosen approach: exactly `## Changes` §1-7 as already written — a gate test pinning the streaming
boundary first, then `ComboConfig` on the existing config/Postgres round-trip, registry
registration of combo names, an in-memory `Resolver` (fallback + round-robin/sticky-limit,
mutex-guarded, cleared on reload), a combo branch inserted into the existing
model-to-`providers []string` resolution path ahead of `Manager.Execute`/`ExecuteStream`, a `503`
exhaustion response, and a Panel Combos page. No new executor or translator — the combo layer sits
above `Manager.ExecuteStream` and every per-candidate attempt runs the existing credential
retry/cooldown/backoff unmodified.

Rejected alternative: skip the gate test and build the routing loop directly. Rejected because the
entire design assumes a failed candidate can only be swapped before any byte reaches the client
(`## Load-bearing assumption`) — if that assumption is false, streaming needs an in-stream error
event instead of a silent switch, which is a different design. Proving it first is cheaper than
discovering it's false after Waves 2-5 are built on top of it.

Primary risks:
- **The load-bearing assumption is wrong** — mitigated by making Wave 1's gate test a hard blocker;
  `work` must stop and re-scope rather than continue if it cannot be made to pass (see `## Risk`).
- **Per-process, in-memory rotation cursor** — each replica in a multi-replica deployment rotates
  independently. Accepted (9router has the same property, `combo.js:88`); mitigation is
  documentation, not code — the docs wave (folded into Wave 5's panel copy or a short note) must
  state this rather than let it be discovered.
- **Combo name colliding with a real model id** — mitigated by `R2`'s load-time validation
  rejecting the config outright rather than allowing ambiguous routing.

Recovery: `Combos` defaults to an empty list, so with no combos configured the new branch is never
entered and the request path is byte-identical to today (`## Rollback`). Any wave after Wave 1 can
be reverted independently by removing its surface; Wave 1's gate test alone is inert (a test, no
behavior change) and safe to keep even if later waves are reverted.

## Phases and Verification
<!-- Phase and task definitions are immutable after to-plan. Do not add task status fields. Append-only Progress is the sole task execution-status source. Only each phase lifecycle status changes to mirror DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. Each planned phase records phase_slug, story_id, status, goal, depends_on, waves, tasks, and checks. -->
- planning_status: planned

### Phase: model-combos
- story_id: `01KZBJ2FG81HM1BAM951HK1SKQ`
- goal: Ship cross-provider model combos: named virtual models resolving to an ordered
  provider/model candidate list with fallback/round-robin rotation, sitting above
  `Manager.ExecuteStream` with no new executor.
- depends_on: none (soft dependency on the already-`done` `error-classification` phase — combo
  works without it, just advances less often; see `## Soft dependency on Plan A`)
- touched_surfaces: `sdk/cliproxy/auth/conductor_stream_bootstrap_test.go` (new), `internal/config/config.go`,
  `internal/config/parse.go`, `internal/registry/` (combo registration), `sdk/cliproxy/combo.go`
  (new), the request path that maps a model name to `providers []string` ahead of
  `Manager.Execute`/`ExecuteStream`, `internal/watcher/config_reload.go` (clear rotation cursor),
  `sdk/cliproxy/auth/cooldown_state.go` (read earliest reset), `web/` (Combos page)
- avoided_surfaces: no new executor or translator (`NG4`), no change to single-provider
  credential/cooldown/backoff internals (`NG1`), no capability-aware reordering (`NG2`), no
  fusion/panel+judge (`NG3`)
- lifecycle status: planned

#### Wave 1 — Gate test (R1, depends on none, BLOCKING)
- task 1.1: `sdk/cliproxy/auth/conductor_stream_bootstrap_test.go` (new): assert
  `streamBootstrapError` is returned only when zero chunks have been written downstream, and that a
  mid-stream upstream failure produces a different error type.
  - touched: `sdk/cliproxy/auth/conductor_stream_bootstrap_test.go` (new)
  - check: `go test ./sdk/cliproxy/auth/... -run StreamBootstrap -v` passes. **If it cannot be made
    to pass, stop and re-scope per `## Load-bearing assumption` rather than continuing to Wave 2.**

#### Wave 2 — Config (R2, depends on Wave 1 gate passing)
- task 2.1: `ComboConfig` struct (`Name`, `Strategy`, `StickyLimit`, `Models`) added to `Config` as
  `Combos []ComboConfig` (`internal/config/config.go`).
  - touched: `internal/config/config.go`
  - check: `go build ./internal/config/...`
- task 2.2: Load-time validation in `internal/config/parse.go`: name non-empty and unique, name does
  not collide with any registered model id, at least one candidate, each candidate parses as
  `provider/model`, `strategy` in `{"", "fallback", "round-robin"}` (empty = `fallback`),
  `sticky-limit < 1` normalized to `1`.
  - touched: `internal/config/parse.go`
  - check: unit tests below
- task 2.3: Unit tests for every validation rule in 2.2 (empty name, duplicate name, name colliding
  with a model id, zero candidates, malformed candidate, invalid strategy, sticky-limit normalization).
  - touched: `internal/config/parse_test.go` (or existing config test file)
  - check: `go test ./internal/config/... -v`

#### Wave 3 — Registry and resolver (R3, R4, depends on Wave 2)
- task 3.1: Register each combo name as a client-visible model in `internal/registry/` so
  `/v1/models` lists it; mark it non-resolvable as a candidate itself (no recursion).
  - touched: `internal/registry/`
  - check: unit test — combo name appears in registry listing; a combo cannot list itself as a
    candidate (rejected at 2.2 validation, confirmed here at the registry level too)
- task 3.2: `sdk/cliproxy/combo.go` (new): `Candidate{Provider, Model}`, `Resolver.Resolve(name)
  ([]Candidate, bool)` — `fallback` always starts at index 0.
  - touched: `sdk/cliproxy/combo.go`
  - check: unit test — `Resolve` returns candidates in config order for `fallback`
- task 3.3: `Resolver.Rotate(name, strategy, stickyLimit) []Candidate` — round-robin with sticky
  limit: stay on the same candidate for N consecutive requests before advancing. Cursor state is an
  in-memory `map[string]state` guarded by a mutex.
  - touched: `sdk/cliproxy/combo.go`
  - check: unit test — `sticky-limit: 3` advances the cursor on the 4th call, not the 2nd
- task 3.4: Clear the rotation cursor map on config reload.
  - touched: `internal/watcher/config_reload.go`
  - check: unit test — cursor state is empty immediately after a simulated reload
- task 3.5: Broader regression sweep for tasks 3.1-3.4 (concurrent `Rotate` calls under the mutex,
  `fallback` strategy unaffected by rotation state).
  - touched: `sdk/cliproxy/combo_test.go` (new)
  - check: `go test ./sdk/cliproxy/... ./internal/registry/... -v`

#### Wave 4 — Request-path loop and exhaustion (R5, R6, depends on Wave 3)
- task 4.1: Combo branch in the model-to-`providers []string` resolution path: iterate candidates
  in `Resolve`/`Rotate` order; on success return; on a non-retryable error return it unmasked
  without trying further candidates; on `502/503/504` with a computed cooldown ≤5s, sleep before the
  next candidate; existing per-candidate credential retry/cooldown/backoff run unmodified inside
  each iteration.
  - touched: the request path calling `Manager.Execute`/`Manager.ExecuteStream`
  - check: unit tests below
- task 4.2: Streaming switch semantics: a bootstrap failure (`streamBootstrapError`, zero bytes
  flushed) switches silently to the next candidate; a post-first-chunk failure does not switch — the
  error reaches the client in-stream.
  - touched: same request path as 4.1
  - check: unit tests below (depends on Wave 1's gate test proving the underlying distinction holds)
- task 4.3: Exhaustion response: all candidates failed → HTTP `503` (not `406`) carrying the
  earliest reset across all candidates (read from `sdk/cliproxy/auth/cooldown_state.go`) plus a
  human string (e.g. `reset after 2m 30s`).
  - touched: same request path, `sdk/cliproxy/auth/cooldown_state.go` (read-only)
  - check: unit test — exhaustion response asserts status code, reset value, and human string
- task 4.4: Full scenario coverage matching `## Verify`'s exact bullet list: candidate1 `429`→
  candidate2 `200`; candidate1 `400` unmasked, candidate2 never attempted; all candidates fail →
  `503` with earliest reset + human string; streaming bootstrap failure switches silently; streaming
  post-chunk failure does not switch; `502` 3s cooldown pauses, `502` 40s cooldown does not.
  - touched: new test file alongside the request-path package
  - check: `go test ./sdk/cliproxy/... ./internal/watcher/... -v`

#### Wave 5 — Panel (R7, depends on Wave 2)
- task 5.1: Combos page — list, create, reorder candidates, pick strategy and sticky limit.
  - touched: `web/` (new page/section under the existing management panel structure)
  - check: `make build-web` (type check + lint + production build); no new test files under `web/`
    per project rule
- task 5.2: Browser runtime check — create/list/reorder a combo through the panel against a running
  server, confirm it round-trips through `/v0/management/config.yaml`.
  - touched: none
  - check: manual browser check against `make dev-web` + `make dev` (same Postgres-dev-mode
    constraint as the `provider-presets` phase — see that plan's Wave 4/task 4.2 Decision entry for
    the established substitute-proof precedent if this blocker recurs)

**Phase-level check:** `go test ./sdk/cliproxy/... ./internal/config/... ./internal/registry/...
./internal/watcher/...` plus `make build-web && make build`
**Optional manual check:** define a 2-candidate combo, revoke the first provider's credential,
confirm requests keep succeeding and the log shows the switch (requires a live server against real
provider credentials — likely hits the same Postgres/live-DB constraint as `provider-presets`;
skip or re-ask per the established pattern if attempted).

## Progress
<!-- Append-only durable entries record timestamp, phase, wave, task, task_status, run_id, trace_id, exact verification/result, and changed surfaces or blocker. -->
- 2026-08-11 · phase model-combos · phase-start · task_status=in-progress · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · branch: feature/model-combos (from master @ 6d16365b) · zharness
  DB migrated 0007-0009 (schema v9) and managed docs refreshed (0.7.0 → 0.8.1) to clear preflight
  drift before run create. Beginning Wave 1 (gate test).
- 2026-08-11 · phase model-combos · wave 1 · task 1.1 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · touched: sdk/cliproxy/auth/conductor_stream_bootstrap_test.go (new)
  · verify: `go test ./sdk/cliproxy/auth/... -run StreamBootstrap -v` — 3/3 pass (incl. one
  pre-existing test). Load-bearing assumption holds: `streamBootstrapError` returned only when zero
  chunks flushed; mid-stream failure surfaces as an in-stream error chunk, never as
  `streamBootstrapError`. Wave 1 is unblocked.
- 2026-08-11 · phase model-combos · wave 2 · task 2.1 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · touched: internal/config/config.go · verify:
  `go build ./internal/config/...` clean.
- 2026-08-11 · phase model-combos · wave 2 · task 2.2 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · touched: internal/config/parse.go (ValidateCombos +
  parseComboCandidate), internal/config/config.go (LoadConfigOptional call) · verify: unit tests
  in task 2.3.
- 2026-08-11 · phase model-combos · wave 2 · task 2.3 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · touched: internal/config/combos_test.go (new) · verify:
  `go test ./internal/config/... -run TestValidateCombos -v` — 7/7 pass (empty name, duplicate,
  model-id collision via LookupStaticModelInfo, zero candidates, malformed candidate, invalid
  strategy, sticky-limit/strategy normalization). `a/b/c` is parseable (provider `a`, model `b/c`)
  and deliberately not rejected.
- 2026-08-11 · phase model-combos · wave 3 · task 3.1 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · touched: sdk/cliproxy/service.go (registerComboModels +
  comboModelsClientID), sdk/cliproxy/service_combo_test.go (new) · verify:
  `go test ./sdk/cliproxy/ -run TestRegisterComboModels -v` — 2/2 pass: combo names listed under
  provider "combo" after register; empty and nil config reloads unregister them.
- 2026-08-11 · phase model-combos · wave 3 · task 3.2 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · touched: sdk/cliproxy/auth/combo.go (new) · verify:
  `go test ./sdk/cliproxy/auth/ -run TestComboResolver -v` — 6/6 pass (fallback order, fallback
  ignores rotation, round-robin sticky blocks [1-3]→[4-6]→[7 wraps], sticky=1 advances, SetCombos
  clears cursor, concurrent Rotate under mutex).
- 2026-08-11 · phase model-combos · wave 3 · task 3.3 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · touched: sdk/cliproxy/auth/conductor.go (Manager.comboResolver,
  SetConfig wiring, ComboResolver accessor) · verify: TestComboResolverManagerWiring passes inside
  the 3.2 run — resolver populated via Manager.SetConfig and reachable via
  Manager.ComboResolver().
- 2026-08-11 · phase model-combos · wave 3 · task 3.4 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · touched: sdk/cliproxy/service.go (applyConfigUpdate call of
  registerComboModels) · verify: `go test ./sdk/cliproxy/` full package pass after wiring.
- 2026-08-11 · phase model-combos · wave 3 · task 3.5 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · verify: `go test ./sdk/cliproxy/... ./internal/config/...
  ./internal/registry/... ./internal/watcher/...` — all clean (combo plumbing touches watcher's
  config funnel only via SetConfig, no direct edits needed).
- 2026-08-11 · phase model-combos · wave 4 · task 4.1 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · touched: sdk/api/handlers/handlers.go (comboOrder +
  executeWithComboFallback + comboSwitchable + comboCooldown; combo short-circuits
  getRequestDetailsWithOptions since combo names are virtual) · verify: handler tests in 4.4 —
  429 falls through to next candidate, 400 unmasked with no further candidates, 502/503/504 with
  cooldown ≤5s sleeps before switching (3s pause test), >5s does not (40s test).
- 2026-08-11 · phase model-combos · wave 4 · task 4.2 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · touched: sdk/api/handlers/handlers.go (bootstrap region of
  executeStreamWithAuthManager: per-candidate retry budget, combo cursor advance, exhaustion wrap;
  call-level initial failures also switch before the goroutine) · verify: bootstrap-failure switch
  test passes silently with both candidates streamed; post-chunk failure test shows payload then
  in-stream 502 with no switch. Depends on Wave 1's gate test — which holds.
- 2026-08-11 · phase model-combos · wave 4 · task 4.3 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · touched: sdk/cliproxy/auth/combo.go (ComboExhaustedError),
  sdk/cliproxy/auth/conductor.go (EarliestComboReset over cooldown_state.go records),
  sdk/api/handlers/handlers.go (comboExhaustedError) · verify: exhaustion test asserts 503
  status, ResetAt equals the earliest marked cooldown (429 short vs 401 30m), and human string
  "reset after <duration>".
- 2026-08-11 · phase model-combos · wave 4 · task 4.4 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · touched: sdk/api/handlers/combo_test.go (new) · verify:
  `go test ./sdk/cliproxy/... ./internal/watcher/...` — all clean; `go test ./sdk/api/handlers/
  -run Combo -v` — 7/7 pass matching `## Verify`'s bullet list exactly (see 4.1-4.3 entries).
- 2026-08-11 · phase model-combos · wave 5 · task 5.1 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · touched: web/src/features/combos/CombosPage.tsx (new),
  web/src/router/MainRoutes.tsx, web/src/components/layout/MainLayout.tsx,
  web/src/components/ui/icons.tsx (IconSidebarCombos), web/src/i18n/locales/{en,vi}.json ·
  verify: `make build-web` clean (tsc + vite production build), `bun run lint` 0 errors
  (8 pre-existing warnings elsewhere, none in new files), no new test files under `web/`.
- 2026-08-11 · phase model-combos · wave 5 · task 5.2 · task_status=DONE · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · touched: web/src/features/combos/CombosPage.tsx (readCombos
  fix, see Decisions) · verify: real browser runtime check against a live server — local
  dockerized Postgres 17 on 127.0.0.1:5433 (throwaway, never the shared Supabase), `llmhub
  init-db-from-env` with a local test config, server on :9090 with `-local-model`; headless
  Chrome via puppeteer-core drove the panel: login → Combos page → empty state → create
  "daily" (fallback) with 2 candidates → reorder both directions → create "rr"
  (round-robin, sticky-limit 3) → save → round-trip confirmed in `/v0/management/config.yaml`
  → reload → both combos list back with order/strategy/sticky-limit intact. 9/9 check points
  pass; runtime also exposes both combo models in `GET /v1/models` with `owned_by: combo`.
  The check caught a real bug before completion — readCombos returned [] on parsed YAML
  (see Decisions) — fixed and re-verified end-to-end.
- 2026-08-11 · phase model-combos · closing handoff · task_status=DONE · handoff_id:
  01KZQGC547PAKCFDHBB2HF9DHQ · check_id: 01KZQGC2B48X980V59RBK39QPS (verdict APPROVED) ·
  all five waves complete; phase closed in zharness; this plan moves to `docs/plans/done/`.

## Decisions
<!-- Append-only durable entries record timestamp, phase/task, decision, and rationale. -->
- 2026-08-11 · wave 3 · resolver lives in `sdk/cliproxy/auth/combo.go`, not `sdk/cliproxy/combo.go`
  as the wave-3 task text wrote · `sdk/cliproxy` root imports `internal/api`, which imports
  `sdk/cliproxy/auth`; a resolver in the root package that imports `internal/config` and is
  consumed by the Manager (in `auth`) would force an import cycle. Placing it beside the Manager in
  `auth` and exposing it via `Manager.ComboResolver()` keeps the surface the same and the graph
  acyclic. Handler-layer access (wave 4) uses `h.AuthManager.ComboResolver()`.
- 2026-08-11 · wave 3 · rotation cursors clear on reload inside `Manager.SetConfig`, not via a
  direct edit to `internal/watcher/config_reload.go` · SetConfig is the single funnel for every
  config update path, including the watcher's debounced reload, so `SetCombos` (which clears
  cursors) there covers all reloads without touching the watcher.
- 2026-08-11 · wave 5 · task 5.2 run against a live local sandbox instead of the paper-trace
  substitute · the plan text authorized the `provider-presets` substitute-proof precedent if the
  Postgres blocker recurred, but Docker is available and postgres images were cached locally, so
  a throwaway container on 127.0.0.1:5433 satisfied PGSTORE_DSN without ever touching the shared
  remote Supabase DSN in `.env`. The real runtime check was strictly stronger and it paid off —
  it caught a genuine panel bug (see next entry).
- 2026-08-11 · wave 5 · `readCombos` in CombosPage.tsx must unwrap YAML nodes via `doc.toJS()`
  · `parseDocument(...).get('combos')` returns a `YAMLSeq`, not a JS array, so
  `Array.isArray(node)` was always false and the panel silently showed "No combos yet" even when
  the config had combos (list path broken; the save path worked — proof: the round-trip check
  saw the combos in `/v0/management/config.yaml` while the panel showed the empty state). The
  first fix attempt called `node.toJS()` on the detached node, which throws
  "A document argument is required" — the document-level `doc.toJS()` carries its own context
  and is the correct call.

## Validation
  and `## Phases and Verification` were already complete — one phase, five waves, fifteen tasks, a
  check per task — but its `story_id` `01KZB3A8AG44JTXJR1928ZTYAG` was minted with `zharness id`,
  which is explicitly non-mutating, and `zharness story` was never invoked. No changeset and no DB
  row ever existed for it, which is why `## Current State` still read `lifecycle_status:
  not-planned` while everything above it was planned. Same root cause diagnosed in
  `docs/plans/done/error-classification.md` — see that doc for the full evidence. The real row is
  `01KZBJ2FG81HM1BAM951HK1SKQ`, created with `zharness story --slug model-combos`. Phase definitions
  and wave/task content are carried over untouched per the to-plan immutability rule; only the story
  ID and the `## Current State` block changed. `depends_on` stays `none` as written — Plan A
  (`error-classification`) is a soft dependency and is already `done`.

## Validation
<!-- Append-only durable entries record timestamp, phase, exact command/result/output, run_id, check_id, verdict, and proof_gaps. -->
- 2026-08-11 · phase model-combos · phase-level check (wave-5-complete gate) · run_id:
  01KZQECHTHRSTB3HR6BX1ZNG4V · `go test ./sdk/cliproxy/... ./internal/config/...
  ./internal/registry/... ./internal/watcher/...` → all `ok` (12 packages, incl.
  sdk/cliproxy/auth combo & handler suites) · `make build-web` → tsc + vite production build
  clean · `make build` → `go build` succeeded (llmhub binary) · `bun run lint` → 0 errors /
  8 pre-existing warnings · verdict: PASS, no proof_gaps — the one live-runtime risk
  (config → registry wiring) was additionally closed by the browser check: `GET /v1/models`
  on the sandbox server listed both combo models with `owned_by: combo`.
- 2026-08-11 · phase model-combos · gate · check_id: 01KZQGC2B48X980V59RBK39QPS ·
  verdict: APPROVED (same-session, deepseek-v4-flash) · proof: phase-level commands above
  plus the task 5.2 browser-run proof links · no proof_gaps.

## Current State and Next Action
- active_phase: model-combos
- lifecycle_status: done
- latest_run_id: 01KZQECHTHRSTB3HR6BX1ZNG4V
- latest_trace_ids: []
- latest_check_id: none
- latest_handoff_id: none
- blockers: none
- exact_next_action: none — all five waves complete. Optional manual check from the plan
  (revoke a provider credential and watch the fallback switch) remains deliberately skipped:
  it needs live provider credentials against the shared environment, which this session does
  not touch. Commit the phase (see `docs/plans/README.md` for the closing ritual), then close
  the phase and update the index.
- open_items:
  - Wave 1's gate test is a hard blocker. If `streamBootstrapError` turns out to be reachable after
    bytes have been flushed, stop and re-scope per `## Load-bearing assumption` — do not build
    Waves 2-5 on top of it.
  - Task 5.2's browser check may hit the same Postgres/live-DB constraint that blocked the
    `provider-presets` phase; follow that plan's established substitute-proof precedent rather than
    inventing a new one.
- exact_next_action: Wave 1 gate test (conductor_stream_bootstrap_test.go) must pass before Wave 2.
