---
title: Kiro quota tracking in 9router
description: Evidence for 9router Kiro provider quota, runtime usage, and routing behavior
status: active
created: 2026-06-08
tags: [github, 9router, kiro, quota]
---

## Findings

9router HEAD (`decolua/9router`, default branch `master`) now has a real Kiro usage fetcher. The stale conclusion that 9router only exposes Kiro model catalog is no longer correct.

### Provider quota fetch

- `getUsageForProvider` dispatches `provider === "kiro"` to `getKiroUsage`.
  `.kit/cache/github/decolua/9router/open-sse/services/usage.js:60-80`
- `parseKiroQuotaData` parses `usageBreakdownList`, `nextDateReset`, `subscriptionInfo.subscriptionTitle`, and nested `freeTrialInfo` into a generic `quotas` map keyed by resource type.
  `.kit/cache/github/decolua/9router/open-sse/services/usage.js:716-752`
- `getKiroUsage` tries three endpoint shapes:
  `codewhisperer.us-east-1.amazonaws.com/getUsageLimits` GET,
  `codewhisperer.us-east-1.amazonaws.com` POST with `AmazonCodeWhispererService.GetUsageLimits`,
  and `q.us-east-1.amazonaws.com/getUsageLimits` GET.
  `.kit/cache/github/decolua/9router/open-sse/services/usage.js:755-819`
- The usage API route refreshes OAuth credentials if needed, calls `getUsageForProvider`, force-refreshes once for auth-expired messages, and returns usage JSON. It does not persist quota snapshots back into the connection record.
  `.kit/cache/github/decolua/9router/src/app/api/usage/[connectionId]/route.js:122-182`

### Dashboard quota UI

- ProviderLimits fetches quota per connection from `/api/usage/{connectionId}`, parses provider-specific quota data, and caches it in browser localStorage.
  `.kit/cache/github/decolua/9router/src/app/(dashboard)/dashboard/usage/components/ProviderLimits/index.js:307-369`
- The quota table renders arbitrary quota rows with `used`, `total`, remaining percent, and `resetAt`.
  `.kit/cache/github/decolua/9router/src/app/(dashboard)/dashboard/usage/components/ProviderLimits/QuotaTable.js:95-189`

### Runtime token usage

- Kiro streaming executor reads `contextUsageEvent`, `meteringEvent`, and `metricsEvent`.
  `.kit/cache/github/decolua/9router/open-sse/executors/kiro.js:298-326`
- If metrics are absent, it estimates input/output tokens from context usage percentage and content length, then emits `usage` on the final OpenAI-compatible chunk.
  `.kit/cache/github/decolua/9router/open-sse/executors/kiro.js:328-369`
- Streaming and non-streaming handlers save token usage by `provider`, `model`, `connectionId`, and API key.
  `.kit/cache/github/decolua/9router/open-sse/handlers/chatCore/streamingHandler.js:76-101`
  `.kit/cache/github/decolua/9router/open-sse/handlers/chatCore/nonStreamingHandler.js:164-166`
  `.kit/cache/github/decolua/9router/open-sse/handlers/chatCore/requestDetail.js:75-101`

### Routing and fallback

- Account selection filters by excluded connection IDs and active `modelLock_*` fields. It does not inspect the dashboard quota snapshot before selecting an account.
  `.kit/cache/github/decolua/9router/src/sse/services/auth.js:55-98`
- The selected account is chosen by provider strategy (`round-robin` or default `fill-first`) after those filters.
  `.kit/cache/github/decolua/9router/src/sse/services/auth.js:100-187`
- When a request fails, `handleSingleModelChat` calls `markAccountUnavailable`, excludes that connection, and retries another account.
  `.kit/cache/github/decolua/9router/src/sse/handlers/chat.js:161-245`
- `markAccountUnavailable` stores a per-model lock (`modelLock_${model}`), error metadata, and backoff level. It does not permanently disable accounts for quota-like provider errors.
  `.kit/cache/github/decolua/9router/src/sse/services/auth.js:193-241`

## Implications For llmhub

The current llmhub WIP is directionally useful but incomplete:

- Keep the Kiro provider quota fetch, but normalize a `quotas` collection/map, not only scalar `current/limit`.
- Add the 9router endpoint fallback sequence instead of only `q.{region}.amazonaws.com/getUsageLimits`.
- Treat quota refresh failure as UI/runtime metadata, not as account error state unless the underlying auth is truly invalid.
- Port Kiro stream `metricsEvent/contextUsageEvent` usage extraction so llmhub records per-account token/request stats after successful calls.
- Be careful with routing: 9router itself does not pre-skip accounts from fetched dashboard quota. If llmhub keeps pre-skip, document it as a deliberate stricter llmhub behavior, not a direct 9router port.
