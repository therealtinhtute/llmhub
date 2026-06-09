# Design

## Domain Model

Kiro auth metadata continues to store OAuth tokens, `profile_arn`, and
normalized provider quota under `kiro_quota`. `overageStatus=ENABLED` means the
operator has accepted upstream overage behavior, so an otherwise exhausted
account stays routable.

## Application Flow

`KiroExecutor` implements request auth preparation for missing `profile_arn`.
It calls `POST /ListAvailableProfiles` with `{"maxResults":10}`, retries
transient failures, and falls back to one token refresh when profile listing
does not produce an ARN.

The overage management action resolves `profile_arn`, posts
`/setUserPreference`, refreshes quota, forces the returned normalized
`overage_status` to the requested value if upstream lags, and persists through
the existing auth manager update path.

Runtime execution keeps an unapplied base Kiro request body. After a 401 token
refresh, it reapplies the refreshed `profile_arn` before retrying upstream.

## Interface Contract

`POST /v0/management/auth-files/kiro/overage`

Request:

```json
{"name":"kiro-auth.json","id":"optional","auth_index":"optional","enabled":true}
```

Response:

```json
{"quota":{"provider_quota_available":true,"overage_status":"ENABLED"}}
```

Errors use existing management status mapping. Missing `enabled` returns 400.

## Data Model

No schema migration. Runtime metadata persists `profile_arn` and `kiro_quota`
inside existing auth payload storage.

## UI / Platform Impact

The Kiro quota card and auth-file quota section render a compact enable/disable
toggle beside the overage row when provider quota exposes overage status. The
toggle is disabled while loading, while the auth is disabled, or when provider
quota is unavailable.

## Observability

Existing upstream request/response logging remains the proof path. The slice
does not add new audit records.

## Alternatives Considered

1. Keep skipping all exhausted Kiro auths. Rejected because Kiro-Go treats
   upstream overage as an account-level routing allowance.
2. Add a global allow-overage flag. Rejected for this slice because the
   approved scope is upstream overage parity only.
3. Require import to provide `profile_arn`. Rejected because Kiro-Go can resolve
   it at runtime.
