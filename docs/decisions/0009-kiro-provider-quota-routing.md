# 0009 Kiro Provider Quota Routing

Date: 2026-06-08

## Status

Accepted

## Context

Kiro auth records were already normalized at import time and routed through a
dedicated executor, but llmhub only showed runtime cooldown state.

Current 9router HEAD has a real Kiro quota fetcher. Its dashboard usage path
tries CodeWhisperer GET, CodeWhisperer POST with
`AmazonCodeWhispererService.GetUsageLimits`, then Q GET. It parses
`usageBreakdownList`, `nextDateReset`, subscription metadata, trial metadata,
and overage fields into arbitrary quota rows.

9router does not route from that dashboard quota snapshot. Its runtime routing
is still model-lock/error driven: failed accounts are marked unavailable for the
model and the request falls back to another connection.

## Decision

Fetch Kiro account quota through the Kiro executor boundary using the 9router
endpoint fallback order. Normalize the response into scalar compatibility
fields plus a `quotas` row collection, then persist the snapshot under auth
runtime metadata as `kiro_quota`.

Expose the normalized quota from management auth-file responses and provide a
dedicated management refresh action. Do not expose raw upstream JSON to the UI.

When a persisted Kiro quota snapshot shows `current >= limit` and the reset
time has not passed, skip that auth during routing unless the normalized
upstream `overageStatus` is `ENABLED`. Treat skipped quota exhaustion as
transient runtime unavailability, not as an operator disable.

This pre-skip behavior is a deliberate llmhub policy, not a direct 9router
port. It avoids burning a request when llmhub already has a fresh exhausted
provider quota snapshot. The overage exception follows Kiro-Go: an account that
has upstream overage enabled remains routable because the operator has accepted
provider-side overage behavior.

For Kiro-Go parity, llmhub also resolves missing `profile_arn` at runtime via
`ListAvailableProfiles` before falling back to one token refresh, and exposes a
management-only `setUserPreference` action for upstream overage toggling. The
toggle writes through to Kiro, refreshes provider quota, and stores the
normalized snapshot under existing auth metadata.

Capture Kiro runtime token/request stats from stream events and persist them
under auth metadata as `kiro_usage_stats`. `metricsEvent` is authoritative when
present; `contextUsageEvent` plus `meteringEvent` can produce an estimated
token count when metrics are absent.

## Alternatives Considered

1. Keep Kiro quota runtime-only. Rejected because `getUsageLimits` gives a real
   account quota source.
2. Store raw Kiro or 9router quota JSON. Rejected because llmhub runtime and UI
   need a stable provider-neutral shape.
3. Disable exhausted Kiro auth records. Rejected because quota exhaustion is
   recoverable at provider reset time.
4. Match 9router routing exactly and wait for provider errors before fallback.
   Rejected because llmhub can make a stronger routing decision when it has a
   persisted exhausted provider quota snapshot.
5. Keep skipping exhausted accounts even when upstream overage is enabled.
   Rejected because it conflicts with Kiro-Go account routing semantics.

## Consequences

Positive:

- Operators can see true Kiro account quota alongside llmhub cooldown state.
- Routing avoids accounts that are already exhausted before burning a request.
- Operators can compare provider quota, runtime cooldown, and observed runtime
  token/request stats per Kiro auth.

Tradeoffs:

- The Kiro quota endpoint is unofficial and may drift, so parsing remains
  defensive and mocked tests cover the contract used by llmhub.
