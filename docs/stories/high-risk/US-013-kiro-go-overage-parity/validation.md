# Validation

## Proof Strategy

Use mocked upstream HTTP servers and local type/build checks. Live Kiro
credentials are not required for acceptance.

## Test Plan

| Layer | Cases |
| --- | --- |
| Unit | Profile resolver uses existing ARN, resolves via profile list, retries transient list failures, falls back to refresh, and errors when no profile exists |
| Unit | Exhausted quota is not blocked when overage is enabled; disabled, unknown, or empty overage still blocks |
| Integration | Management overage endpoint posts `setUserPreference`, refreshes quota, and persists `kiro_quota` |
| Integration | Runtime 401 retry reapplies refreshed `profile_arn` to non-stream request body |
| Integration | Selector skips exhausted persisted Kiro quota only when overage is disabled or unavailable |
| E2E | Not required for mocked provider slice |
| Platform | Go package bundle, full Go sweep, web type-check/build, `git diff --check` |
| Performance | Not required |
| Logs/Audit | Existing upstream request/response hooks remain in use |

## Fixtures

- Mock `ListAvailableProfiles` responses with one profile, transient 5xx, and empty list.
- Mock Kiro refresh responses with and without `profileArn`.
- Mock `setUserPreference` and `getUsageLimits` responses.
- Persisted `kiro_quota` snapshots using scalar fields, quota arrays, and legacy quota maps.

## Commands

```text
go test ./internal/runtime/executor ./internal/api/handlers/management ./sdk/cliproxy/auth ./sdk/cliproxy
go test ./...
cd web && bun run type-check && bun run build
git diff --check
```

## Acceptance Evidence

Pending final verification.
