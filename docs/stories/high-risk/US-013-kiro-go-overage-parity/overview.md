# Overview

## Current Behavior

llmhub can import and route Kiro auths, fetch Kiro provider quota, persist
`kiro_quota`, and skip exhausted accounts. The quota behavior still diverges
from Kiro-Go when an auth lacks `profile_arn` or when upstream overage is
enabled.

## Target Behavior

llmhub resolves missing Kiro profile ARNs before runtime requests, respects
upstream `overageStatus=ENABLED` when deciding whether an exhausted account is
routable, and exposes a management-only overage toggle that writes through
`setUserPreference` and refreshes the normalized quota snapshot.

## Affected Users

- Operators managing Kiro OAuth accounts.
- API clients routed through Kiro auth pools.

## Affected Product Docs

- `docs/decisions/0009-kiro-provider-quota-routing.md`
- `docs/stories/US-002-kiro-provider/validation.md`

## Non-Goals

- No import schema change.
- No Postgres migration.
- No regular client API for changing overage.
- No global allow-over-quota flag.
