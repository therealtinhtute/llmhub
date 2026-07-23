# 0010 Auth Cooldown and Error Classification

Date: 2026-07-23

## Status

Accepted

## Context

Repeated quota failures could advance backoff while an existing recovery window
was still active, and retry jitter could exceed the configured maximum wait.
OAuth `invalid_grant` failures at HTTP 400 could be treated as ordinary
request-invalid errors, preventing credential fallback and consistent
suspension. Generic unsupported `count_tokens` endpoint failures could also make
an otherwise valid model unavailable.

These behaviors share the conductor's persisted auth/model state, scheduler
snapshots, registry visibility, hooks, and Kiro cooling override, so
handler-local fixes would create inconsistent routing.

## Decision

Advance quota backoff only when no active `NextRecoverAt` window remains. During
an active window, retain its deadline and backoff level. Clamp the base retry
wait to the configured maximum before adding jitter, and cap jitter by one
quarter of the clamped wait, two seconds, and the remaining maximum budget. A
non-positive maximum yields zero.

Classify OAuth `invalid_grant` only for HTTP 400 or 401. Structured values must
exactly match `invalid_grant` in relevant error/code/type fields; textual
matching must use identifier boundaries so longer identifiers such as
`invalid_grant_type` and `invalid_granted` do not match. A classified failure
receives the existing 30-minute credential/model suspension behavior and remains
eligible for credential fallback. Unrelated HTTP 400 request errors retain
request-invalid behavior.

For CountTokens execution, treat generic HTTP 404 endpoint failures as
availability-neutral while still recording failure counters, persistence, and
result hooks. Explicit `model_not_found`, including nested structured payloads
and wrapped or joined errors, remains availability-changing and follows the
existing model suspension path. Keep this policy in the shared conductor so Amp
inherits it without route-specific production logic.

Preserve `disableCooling` behavior, including Kiro's existing override, and do
not add request-time refresh, storage/schema, or public error-contract changes.

## Alternatives Considered

1. Escalate backoff on every 429. Rejected because concurrent failures within
   one provider recovery window compound cooldown without new availability
   information.
2. Add jitter before clamping. Rejected because it can exceed the caller's
   maximum retry budget.
3. Match `invalid_grant` with an unrestricted substring. Rejected because it
   misclassifies longer unrelated identifiers.
4. Treat every CountTokens 404 as model unavailable. Rejected because many
   Anthropic-compatible providers support message generation but do not
   implement the optional count endpoint.
5. Handle CountTokens only in Amp. Rejected because direct provider routes and
   other callers use the same conductor and require one availability policy.

## Consequences

Positive:

- Concurrent quota failures no longer inflate one active cooldown window.
- Retry sleeps stay within their configured maximum.
- OAuth credential failures and ordinary request errors take distinct fallback
  paths.
- Missing CountTokens endpoints no longer hide otherwise working models.
- Explicit model absence still updates registry and scheduler availability.

Tradeoffs:

- Availability-neutral CountTokens failures still increment failure counters
  and invoke persistence/hooks, so operational metrics show the unsupported
  call even though routing remains unchanged.
- Error classification must recursively inspect wrapped structured errors and
  maintain narrow identifier rules as provider payloads evolve.
