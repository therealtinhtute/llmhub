---
id: 01KZAHSQFG0JTC7GZHZJX0FECJ
type: plan
intake_id: 01KZAHST9K8JKBFEEDHNMESY7V
lane: normal
status: active
created: 2026-08-05
updated: 2026-08-06
---

# Plan A — Unified error classification (text-before-status)

Status: **approved** · Created 2026-08-05 · Ships independently
Source mechanism: `decolua/9router` `open-sse/config/errorConfig.js:59` (`ERROR_RULES`)
Reference skill: `.claude/skills/9router-port/references/routing.md` §2

## Outcome
- result: `shouldRetryAfterError` (`sdk/cliproxy/auth/conductor.go:2600`) stops silently dropping non-429 quota/rate-limit failures — a shared text-before-status classifier drives retry/cooldown/return decisions, and the four existing hand-rolled keyword sites consolidate onto it with no behavior change.
- success_signals:
  - A `500`/`503`/`200`-with-error-body response carrying quota/rate-limit language (e.g. `"rate limit exceeded"`, `"model is overloaded"`) now triggers retry + cooldown + credential switch instead of surfacing as a hard, unretried failure.
  - `isRequestInvalidError` (`conductor.go:3516`) still runs before the new classifier, so genuine client errors (e.g. `400 invalid_request_error`) are never converted into retries.
  - Antigravity's three-way 429 decision (`instantRetrySameAuth` / `shortCooldownSwitchAuth` / `fullQuotaExhausted`, `internal/runtime/executor/antigravity_executor.go:333`) keeps its structure; only its keyword-matching loop (`:391`) calls the shared classifier.
  - The 4 existing keyword-matching sites (`antigravity_executor.go:391,2358,2376`, `codex_executor.go:1214`) are consolidated onto the one shared table with no behavior change.
  - `go test ./sdk/cliproxy/... ./internal/runtime/executor/...` passes, including a new `classify_test.go`, and `cooldown_backoff_test.go` / `cooldown_persistence_test.go` / `conductor_availability_test.go` pass unchanged.

## Authority and Requirements
- authority:
  - User session approval, 2026-08-06: "ok vậy implement phase A — Unified error classification đi", after reviewing this plan's content directly.
  - Source mechanism: `decolua/9router` `open-sse/config/errorConfig.js:59` (`ERROR_RULES`), ported via reference skill `.claude/skills/9router-port/references/routing.md` §2.
  - Current retry gate: `sdk/cliproxy/auth/conductor.go:2600` `shouldRetryAfterError`, `conductor.go:3516` `isRequestInvalidError`.
  - Existing per-auth backoff/cooldown tracking: `sdk/cliproxy/auth/cooldown_state.go`, `cooldown_backoff_test.go`.
  - Existing provider-specific keyword classifiers being consolidated: `internal/runtime/executor/antigravity_executor.go:391,2358,2376`, `internal/runtime/executor/codex_executor.go:1214`.
- requirements:
  - R1 [accepted]: Add `sdk/cliproxy/auth/classify.go` with a `Disposition` enum (`DispositionNone`/`DispositionRetryBackoff`/`DispositionCooldown`/`DispositionReturn`), a `Rule` struct (`Text string`, `Status int`, `Cooldown time.Duration`, `Backoff bool`, `Disp Disposition`), and `func Classify(status int, body string) (Disposition, time.Duration, bool)` matching text rules before status rules per the table in "Approach" below. | source: `errorConfig.js:59`, reconciled with existing executor keyword tables
  - R2 [accepted]: Unmatched input returns `DispositionNone`, not a default cooldown — this deliberately diverges from 9router's `TRANSIENT_COOLDOWN_MS` default because LLMHub already has its own unmatched path and a silent default would change behavior for every provider at once. | source: plan author decision, this doc
  - R3 [accepted]: Insert the classifier into `shouldRetryAfterError` between the existing `closestCooldownWait` check and the `status != 429` bail; `isRequestInvalidError` must still run first so client errors are never converted into retries. | source: `conductor.go:2600`
  - R4 [accepted]: `DispositionRetryBackoff` reuses the existing per-auth backoff counter (`cooldown_state.go`) rather than introducing a second counter; falls back to `attempt` when no auth is in scope at the call site. | source: `cooldown_backoff_test.go` established pattern
  - R5 [accepted]: Any provider-reported reset window honored by the classifier is capped at 30m (port of `MAX_RATE_LIMIT_COOLDOWN_MS`, `errorConfig.js:42`), since Codex reports `resets_at` 5-6h out and honoring that verbatim parks a credential for a day. | source: `errorConfig.js:42`
  - R6 [accepted]: Replace the 4 existing keyword-matching call sites with calls to `Classify`, preserving each executor's surrounding decision logic — mechanical, no behavior change. | source: `antigravity_executor.go:391,2358,2376`, `codex_executor.go:1214`

## Non-goals
- NG1: No change to cooldown persistence, backoff math, or credential selection internals — `cooldown_state.go`, `selector.go`, `scheduler.go` stay as-is.
- NG2: No removal of Antigravity's three-way 429 decision logic beyond the keyword-loop swap.
- NG3: No user-facing YAML config for the classification table in v1 — Go table only.

## Building

One classification table that maps an upstream failure to a routing decision
(`retry` / `cooldown` / `return-as-is`), matched **text rules first, then status rules** — and
wire it into `Manager.shouldRetryAfterError` so non-429 quota errors stop being silently
unretried.

## Not building

- No change to cooldown persistence, backoff math, or credential selection — those exist and work
  (`sdk/cliproxy/auth/cooldown_state.go`, `selector.go`, `scheduler.go`).
- No removal of provider-specific decision logic that does more than classify (e.g. Antigravity's
  three-way 429 decision at `internal/runtime/executor/antigravity_executor.go:333` produces
  `instantRetrySameAuth` / `shortCooldownSwitchAuth` / `fullQuotaExhausted` — that stays).
- No user-facing config for the table in v1. It is a Go table, not YAML.

## The bug this fixes

`sdk/cliproxy/auth/conductor.go:2600` `shouldRetryAfterError`:

```go
status := statusCodeFromError(err)
if status == http.StatusOK        { return 0, false }
if isRequestInvalidError(err)     { return 0, false }
wait, found := m.closestCooldownWait(providers, model, attempt)
if found { ... return wait, true }
if status != http.StatusTooManyRequests { return 0, false }   // ← everything else: no retry
```

LLMHub today has a **negative** classifier (`isRequestInvalidError`, `conductor.go:3516` — does
match on text for 400/404/500) but no **positive** one. So an upstream returning
`500` with body `"rate limit exceeded"`, or `503 "model is overloaded"`, or a `200` carrying an
error envelope, falls through to `return 0, false`: no retry, no cooldown recorded, no credential
marked. The client eats the error while a healthy sibling credential sits idle.

Provider-level keyword tables already exist, one per executor, each invented separately:

| Site | Keywords |
|---|---|
| `internal/runtime/executor/antigravity_executor.go:391` | `antigravityQuotaExhaustedKeywords` loop over lowercased body |
| `internal/runtime/executor/antigravity_executor.go:2358` | `"no capacity available"` |
| `internal/runtime/executor/antigravity_executor.go:2376` | `"resource has been exhausted"` |
| `internal/runtime/executor/codex_executor.go:1214` | `"selected model is at capacity"`, `"model is at capacity. please try a different model"` |

9router solves this with one ordered table checked text-first, because status alone misclassifies
often enough to matter (`errorConfig.js:59`).

## Approach

Add a classifier package that returns a decision, and call it from the one place that currently
decides retryability. Provider executors keep their own richer decisions; the table is the
**fallback** when the executor did not already produce one.

### Changes

1. **`sdk/cliproxy/auth/classify.go`** (new, ~120 lines)

   ```go
   type Disposition int
   const (
       DispositionNone Disposition = iota // no match — caller keeps current behavior
       DispositionRetryBackoff            // quota/rate/capacity → exponential backoff cooldown
       DispositionCooldown                // fixed cooldown, switch credential
       DispositionReturn                  // client error — surface as-is, never retry
   )

   type Rule struct {
       Text     string        // lowercase substring; empty = status rule
       Status   int           // matched only when Text is empty
       Cooldown time.Duration // fixed cooldown; zero with Backoff=true
       Backoff  bool          // use exponential backoff instead of a fixed cooldown
       Disp     Disposition
   }

   func Classify(status int, body string) (Disposition, time.Duration, bool)
   ```

   Table contents, ported from `errorConfig.js:59` and reconciled with the executor keywords above:

   ```
   text "no credentials"             → DispositionCooldown, 2m
   text "request not allowed"        → DispositionCooldown, 5s
   text "improperly formed request"  → DispositionCooldown, 2m
   text "rate limit"                 → DispositionRetryBackoff
   text "too many requests"          → DispositionRetryBackoff
   text "quota exceeded"             → DispositionRetryBackoff
   text "resource has been exhausted"→ DispositionRetryBackoff
   text "no capacity available"      → DispositionRetryBackoff
   text "at capacity"                → DispositionRetryBackoff
   text "overloaded"                 → DispositionRetryBackoff
   status 401 | 402 | 403 | 404      → DispositionCooldown, 2m
   status 429                        → DispositionRetryBackoff
   (unmatched)                       → DispositionNone
   ```

   Unmatched returns `DispositionNone`, **not** a 30s default (9router's `TRANSIENT_COOLDOWN_MS`).
   Reason: LLMHub already has its own unmatched path and silently adding a 30s cooldown to every
   unknown error would change existing behavior for every provider at once.

2. **`sdk/cliproxy/auth/conductor.go:2600`** — insert the classifier between the
   `closestCooldownWait` check and the `status != 429` bail:

   ```go
   if disp, wait, ok := Classify(status, strings.ToLower(err.Error())); ok {
       switch disp {
       case DispositionReturn:      return 0, false
       case DispositionRetryBackoff, DispositionCooldown:
           if !m.retryAllowed(attempt, providers) { return 0, false }
           if wait > maxWait { return 0, false }
           return wait, true
       }
   }
   if status != http.StatusTooManyRequests { return 0, false }
   ```

   Order matters: `isRequestInvalidError` still runs first, so genuine client errors are never
   converted into retries.

3. **Backoff level source.** `DispositionRetryBackoff` needs a level. Reuse the existing per-auth
   backoff already tracked for cooldowns (`sdk/cliproxy/auth/cooldown_state.go`,
   `cooldown_backoff_test.go`) rather than introducing a second counter. If no auth is in scope at
   the call site, fall back to `attempt` as the level.

4. **Cap.** Port `MAX_RATE_LIMIT_COOLDOWN_MS = 30m` (`errorConfig.js:42`) as a ceiling on any
   *provider-reported* reset honored by the classifier. Rationale in-source: Codex reports
   `resets_at` 5–6 hours out, and honoring it verbatim parks a credential for the day.

5. **Consolidation (mechanical, no behavior change).** Replace the four executor keyword sites in
   the table above with calls to `Classify`, keeping each executor's surrounding decision logic.
   Antigravity's three-way 429 decision keeps its own structure; only the keyword loop at
   `antigravity_executor.go:391` is swapped for a `Classify` call.

## Verify

- `go test ./sdk/cliproxy/... ./internal/runtime/executor/...`
- New `classify_test.go`:
  - `500` + body `"rate limit exceeded"` → `DispositionRetryBackoff` (the bug case; fails on
    current `master`)
  - `200` + body `"quota exceeded"` → `DispositionRetryBackoff`
  - `400` + body `"invalid_request_error"` → not retried (guards against the classifier
    overriding `isRequestInvalidError`)
  - text rule beats a conflicting status rule: `403` + `"rate limit"` → backoff, not 2m cooldown
  - unmatched `418` → `DispositionNone`, existing behavior preserved
  - provider-reported reset of `6h` clamps to `30m`
- Regression: existing `cooldown_backoff_test.go`, `cooldown_persistence_test.go`,
  `conductor_availability_test.go` pass unchanged.
- Live: point an `openai-compatibility` entry at a provider that returns a non-429 quota error,
  confirm the log shows a cooldown and a credential switch instead of a surfaced error.

## Rollback

`Classify` returns `DispositionNone` for everything if the table is emptied — the inserted block
becomes a no-op and `shouldRetryAfterError` behaves exactly as it does today. Step 5 is revertable
independently of steps 1–4.

## Risk

Widening retry means an error that previously surfaced immediately now costs a cooldown plus a
retry. Mitigated by: `isRequestInvalidError` still running first, `retryAllowed`/`maxWait` still
bounding the loop, and `DispositionNone` as the default for anything unmatched.

## Approach and Risks
- approach:
  - Add a standalone classifier package (`sdk/cliproxy/auth/classify.go`) that returns a
    `Disposition` from `(status, body)`, matched text-rules-first then status-rules, per the table
    in `## Approach` above. Call it from the one place that currently decides retryability
    (`shouldRetryAfterError`, `conductor.go:2600`), inserted between the existing
    `closestCooldownWait` check and the `status != 429` bail. `isRequestInvalidError` keeps running
    first so genuine client errors are never converted into retries.
  - Reuse the existing per-auth backoff counter (`cooldown_state.go`) for `DispositionRetryBackoff`
    rather than a second counter; fall back to `attempt` when no auth is in scope at the call site.
  - Cap any provider-reported reset window the classifier honors at 30m
    (`MAX_RATE_LIMIT_COOLDOWN_MS`, ported from `errorConfig.js:42`) — Codex reports `resets_at`
    5–6h out and honoring that verbatim parks a credential for the day.
  - Mechanically consolidate the 4 existing hand-rolled keyword-matching sites
    (`antigravity_executor.go:391,2358,2376`, `codex_executor.go:1214`) onto `Classify`, preserving
    each executor's surrounding decision logic — no behavior change.
  - Rejected alternative: defaulting unmatched errors to a fixed cooldown, as 9router's
    `TRANSIENT_COOLDOWN_MS` does. Rejected because LLMHub already has its own unmatched path and a
    silent default would change behavior for every provider at once; unmatched instead returns
    `DispositionNone` (R2).
- risks:
  - Widening retry means an error that previously surfaced immediately now costs a cooldown plus a
    retry. | mitigation: `isRequestInvalidError` still runs first, `retryAllowed`/`maxWait` still
    bound the loop, `DispositionNone` is the default for anything unmatched.
- recovery:
  - `Classify` returning `DispositionNone` for everything (empty table) makes the inserted block a
    no-op — `shouldRetryAfterError` behaves exactly as it does today. The executor-consolidation
    wave (Wave 3) is revertible independently of Waves 1–2.

## Phases and Verification

### Phase: error-classification
- story_id: `01KZAHXD0Y9R5B5EYA61D15048`
- goal: Fix `shouldRetryAfterError` to retry non-429 quota/rate-limit errors via a shared
  text-before-status classifier.
- depends_on: none
- touched_surfaces: `sdk/cliproxy/auth/classify.go` (new), `sdk/cliproxy/auth/classify_test.go`
  (new), `sdk/cliproxy/auth/conductor.go`, `internal/runtime/executor/antigravity_executor.go`,
  `internal/runtime/executor/codex_executor.go`
- avoided_surfaces: `sdk/cliproxy/auth/cooldown_state.go`, `sdk/cliproxy/auth/selector.go`,
  `sdk/cliproxy/auth/scheduler.go` (NG1 — logic unchanged), no config YAML (NG3)
- lifecycle status: checked

#### Wave 1 — Build the classifier
- task 1.1: Implement `sdk/cliproxy/auth/classify.go` — `Disposition` enum, `Rule` struct, and
  `func Classify(status int, body string) (Disposition, time.Duration, bool)` with the exact table
  from `## Approach` (text rules first, then status rules; unmatched → `DispositionNone` per R2;
  30m reset cap per R5).
  - touched: `sdk/cliproxy/auth/classify.go`
  - check: `go build ./sdk/cliproxy/auth/...`
- task 1.2: Implement `sdk/cliproxy/auth/classify_test.go` covering the 6 cases from `## Verify`:
  `500`+"rate limit exceeded"→RetryBackoff (bug case); `200`+"quota exceeded"→RetryBackoff;
  `400`+"invalid_request_error"→not retried; text rule beats conflicting status rule
  (`403`+"rate limit"→backoff, not 2m cooldown); unmatched `418`→`DispositionNone`; 6h
  provider-reported reset clamps to 30m.
  - touched: `sdk/cliproxy/auth/classify_test.go`
  - check: `go test ./sdk/cliproxy/auth/... -run TestClassify -v` — all 6 cases pass

#### Wave 2 — Wire into the retry gate (depends on Wave 1)
- task 2.1: Insert the classifier call into `shouldRetryAfterError` (`conductor.go:2600`) per the
  exact snippet in `## Approach` item 2, between `closestCooldownWait` and the `status != 429`
  bail; confirm `isRequestInvalidError` still runs first; wire `DispositionRetryBackoff` to reuse
  the existing per-auth backoff counter (R4), falling back to `attempt` when no auth is in scope.
  - touched: `sdk/cliproxy/auth/conductor.go`
  - check: `go test ./sdk/cliproxy/...` — full package pass, including unchanged
    `cooldown_backoff_test.go`, `cooldown_persistence_test.go`, `conductor_availability_test.go`,
    plus `classify_test.go`

#### Wave 3 — Consolidate executor keyword sites (depends on Wave 1)
- task 3.1: Replace the 4 keyword-matching call sites with calls to `Classify`, preserving each
  executor's surrounding decision logic (Antigravity's three-way 429 decision keeps its structure —
  only its keyword loop at `:391` is swapped).
  - touched: `internal/runtime/executor/antigravity_executor.go`,
    `internal/runtime/executor/codex_executor.go`
  - check: `go test ./internal/runtime/executor/...` — behavior unchanged

**Phase-level check:** `go test ./sdk/cliproxy/... ./internal/runtime/executor/...`
**Optional manual check:** point an `openai-compatibility` entry at a provider returning a
non-429 quota error; confirm the log shows a cooldown and credential switch instead of a surfaced
error.

## Progress
- 2026-08-06T00:00:00Z | phase: error-classification | wave: — | task: — | task_status: in-progress | run_id: 01KZAHYZG9ZH230122E2J587R5 | trace_id: none | changed_surfaces: none yet | verification: none yet — phase start
- 2026-08-06T00:10:00Z | phase: error-classification | wave: 1 | task: 1.1 | task_status: DONE | run_id: 01KZAHYZG9ZH230122E2J587R5 | trace_id: none | changed_surfaces: sdk/cliproxy/auth/classify.go | verification: `go build ./sdk/cliproxy/auth/...` — clean
- 2026-08-06T00:10:00Z | phase: error-classification | wave: 1 | task: 1.2 | task_status: DONE | run_id: 01KZAHYZG9ZH230122E2J587R5 | trace_id: none | changed_surfaces: sdk/cliproxy/auth/classify_test.go | verification: `go test ./sdk/cliproxy/auth/... -run TestClassify -v` — all cases pass
- 2026-08-06T00:10:00Z | phase: error-classification | wave: 1 | task: — | task_status: DONE | run_id: 01KZAHYZG9ZH230122E2J587R5 | trace_id: 01KZAJ593AK85EPFME1P153AFK | changed_surfaces: sdk/cliproxy/auth/classify.go, sdk/cliproxy/auth/classify_test.go | verification: wave complete — trace recorded
- 2026-08-06T00:25:00Z | phase: error-classification | wave: 2 | task: 2.1 | task_status: DONE | run_id: 01KZAHYZG9ZH230122E2J587R5 | trace_id: none | changed_surfaces: sdk/cliproxy/auth/conductor.go (shouldRetryAfterError insertion + new retryBackoffWait helper), sdk/cliproxy/auth/classify_test.go (added retryBackoffWait cases) | verification: `go test ./sdk/cliproxy/...` — all packages pass, including unchanged cooldown_backoff_test.go / cooldown_persistence_test.go / conductor_availability_test.go
- 2026-08-06T00:25:00Z | phase: error-classification | wave: 2 | task: — | task_status: DONE | run_id: 01KZAHYZG9ZH230122E2J587R5 | trace_id: 01KZAJ6Q7EZKRG4J6XEX1PZQ30 | changed_surfaces: sdk/cliproxy/auth/conductor.go, sdk/cliproxy/auth/classify_test.go | verification: wave complete — trace recorded
- 2026-08-06T00:40:00Z | phase: error-classification | wave: 3 | task: 3.1 | task_status: DONE (scope adjusted, see Decisions) | run_id: 01KZAHYZG9ZH230122E2J587R5 | trace_id: none | changed_surfaces: sdk/cliproxy/auth/classify.go (exported KeywordNoCapacityAvailable, KeywordResourceHasBeenExhausted), internal/runtime/executor/antigravity_executor.go (antigravityShouldRetryNoCapacity, antigravityShouldRetryTransientResourceExhausted429 now reference the shared constants) | verification: `go build`, `go vet`, `gofmt -l` clean; `go test ./sdk/cliproxy/... ./internal/runtime/executor/...` — all pass
- 2026-08-06T00:40:00Z | phase: error-classification | wave: 3 | task: — | task_status: DONE | run_id: 01KZAHYZG9ZH230122E2J587R5 | trace_id: 01KZAK76DJYSC5Q7SZZTSHJQWB | changed_surfaces: sdk/cliproxy/auth/classify.go, internal/runtime/executor/antigravity_executor.go | verification: wave complete — trace recorded

## Decisions
- 2026-08-06 | phase: error-classification | wave: 2 | task: 2.1 — `shouldRetryAfterError`'s signature (`err, attempt, providers []string, model, maxWait`) never carries an `*Auth`, at any of its 3 call sites. R4's "reuse the per-auth backoff counter, fall back to `attempt` if no auth is in scope" therefore always takes the fallback branch here; implemented `retryBackoffWait` keyed purely on `attempt` via the existing `nextQuotaCooldown` formula. No plan-scope change — R4 anticipated exactly this fallback.
- 2026-08-06 | phase: error-classification | wave: 2 | task: 2.1 — R5 asked to "port `MAX_RATE_LIMIT_COOLDOWN_MS = 30m` as a ceiling". The package already defines `quotaBackoffMax = 30 * time.Minute` (`conductor.go:85`) for the existing persisted-cooldown backoff formula. Reused that constant instead of adding a second 30m literal, to avoid two magic numbers that could drift apart; same value, same behavior, satisfies R5 as written.
- 2026-08-06 | phase: error-classification | wave: 3 | task: 3.1 — Task 3.1 as literally written ("replace the 4 keyword-matching call sites with calls to `Classify`") turned out unsafe: `Classify`'s `DispositionRetryBackoff` is a coarse bucket matching 7 different phrases (rate limit, too many requests, quota exceeded, resource has been exhausted, no capacity available, at capacity, overloaded), while each of the 4 sites is a narrow single/double-phrase gate feeding its own distinct decision. Branching any of them on `Disposition` instead of their own literal would make each fire on phrases it never matched before — a real behavior change, violating R6 and the plan's own "no behavior change" framing for this step. Surfaced to the user via `AskUserQuestion`; user chose the safer alternative: keep each site's own narrow `strings.Contains` check, but dedupe the *literal string* against `classify.go` wherever a genuine duplicate exists, rather than branching on `Disposition`. On inspection only 2 of the 4 sites are genuine duplicates of an R1-locked `classifyTextRules` entry: `antigravityShouldRetryNoCapacity` ("no capacity available") and `antigravityShouldRetryTransientResourceExhausted429` ("resource has been exhausted"). Those two now reference newly-exported `cliproxyauth.KeywordNoCapacityAvailable` / `cliproxyauth.KeywordResourceHasBeenExhausted`, sourced from the same table entries via the constants, so string and behavior stay identical to today. The other 2 sites — antigravity's `quota_exhausted`/`quota exhausted` keyword loop (`:391`) and Codex's `isCodexModelCapacityError` two capacity phrases (`:1200`) — have no matching entry in classify.go's R1 table at all; adding them there would expand what `conductor.go`'s `shouldRetryAfterError` retries on (new behavior, out of R1's exact enumerated scope), so they were left untouched. Net: task 3.1 done for the 2 sites where "shared source, same behavior" is actually achievable; the other 2 are correctly out of scope, not a shortfall.

## Validation
- 2026-08-06T01:05:00Z | phase: error-classification | run_id: 01KZAHYZG9ZH230122E2J587R5 | check_id: 01KZAKBKQHQDM17B0AM26J58YF | verdict: APPROVED | commands:
  - `go build ./...` — clean, no output
  - `go vet ./...` — clean, no output
  - `gofmt -l` on all touched .go files — empty, all formatted
  - `go test ./...` — all packages ok, no failures (repo-wide, not just touched packages)
  - manual: confirmed `isRequestInvalidError` still gates before the classifier block (`conductor.go:2611` before `:2618`) — R2 invariant preserved
  - manual: sibling-instance search across `internal/runtime/executor/*.go` for quota/rate-limit/capacity keyword checks found only the 4 plan-named sites — no undiscovered instances, consolidation scope complete
  - `zharness audit --json` — no contract_violations, no unlinked_proofs
  - proof_gaps: none

## Current State and Next Action
- active_phase: error-classification
- lifecycle_status: checked
- latest_run_id: 01KZAHYZG9ZH230122E2J587R5
- latest_check_id: 01KZAKBKQHQDM17B0AM26J58YF
- blockers: none
- open_items: 2 of Wave 3's 4 named sites (antigravity `:391` quota_exhausted loop, codex
  `isCodexModelCapacityError`) intentionally left unconsolidated — see Decisions. Not a defect;
  revisit only if those literal phrases are ever added to `classify.go`'s table under a separate,
  reviewed change.
- exact_next_action: check full APPROVED — route to `git` for commit, or `handoff` if wrapping up
  the session
