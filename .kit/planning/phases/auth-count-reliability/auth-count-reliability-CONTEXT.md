# Context: Auth and CountTokens reliability

Phase: auth-count-reliability
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: high
Expected Proof: unit, integration

## Goal

Correct cooldown escalation, bounded jitter, HTTP 400 `invalid_grant`, and CountTokens availability behavior without disrupting Postgres persistence, Amp, or Kiro.

## Scope Boundary

### Allowed Surfaces

- `sdk/cliproxy/auth/conductor.go`
- `sdk/cliproxy/auth/*cooldown*_test.go`
- `sdk/cliproxy/auth/conductor_overrides_test.go`
- `sdk/cliproxy/auth/conductor_count_tokens_test.go`
- `sdk/cliproxy/auth/conductor_scheduler_refresh_test.go`
- `internal/api/modules/amp/routes_test.go`

### Forbidden Surfaces

- Shared request-time 401 refresh.
- OAuth session management and credential persistence.
- Kiro executor refresh behavior.
- Database schema, storage implementation, plugin, logging, translator, registry, UI, installer, and release changes.

## Spec Hooks

- Requirements 3-6.
- Existing Postgres cooldown/auth state remains authoritative.
- Amp CountTokens must inherit shared behavior.

## Locked Decisions

- Escalate quota backoff only when no active recovery window remains.
- Clamp base wait to `[0, max-retry-interval]`, then add jitter bounded by `min(clamped/4, 2s, max-retry-interval-clamped)`; a non-positive maximum yields zero.
- Recognize structured/textual `invalid_grant`, including HTTP 400; do not broaden all 400 errors.
- Generic unsupported CountTokens endpoint failures are availability-neutral; explicit `model_not_found` is not.

## Assumptions

- Existing auth error types expose enough status/code/message data for precise classification.
- New tests can use deterministic seams for jitter rather than asserting random exact values.

## Canonical Refs

- `.kit/planning/SPEC.md`
- `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`
- `sdk/cliproxy/auth/conductor.go:2294-2305`
- `sdk/cliproxy/auth/conductor.go:2397-2417`
- `sdk/cliproxy/auth/conductor.go:2797-2806`
- `sdk/cliproxy/auth/conductor.go:2994-3041`
- `internal/api/modules/amp/routes.go:312-324`

## Rejected Options

- Replace the conductor with upstream: rejected because local persistence and Kiro usage hooks would be lost.
- Implement CountTokens neutrality only in HTTP handlers: rejected because Amp and other callers share the conductor.
- Add shared 401 refresh now: rejected because Kiro already owns executor-local refresh/replay.

## Deferred Ideas

- Shared synchronized request-time 401 refresh with explicit executor ownership/bypass.
- Postgres-safe OAuth cancellation.
- Kiro CountTokens implementation beyond the current zero result.

## Escalate If

- Correct CountTokens classification requires changing the public error schema.
- Cooldown changes cannot preserve the current persistence callbacks or Kiro usage update path.
