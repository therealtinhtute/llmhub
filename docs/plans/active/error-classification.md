# Plan A — Unified error classification (text-before-status)

Status: **awaiting approval** · Created 2026-08-05 · Ships independently
Source mechanism: `decolua/9router` `open-sse/config/errorConfig.js:59` (`ERROR_RULES`)
Reference skill: `.claude/skills/9router-port/references/routing.md` §2

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
