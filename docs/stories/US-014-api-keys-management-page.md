# US-014 API Keys Management Page

## Status

implemented

## Lane

normal

## Product Contract

Management Center exposes a dedicated API Keys page for direct CRUD of gateway
API keys, while Config Panel visual mode no longer owns `api-keys` or
`auth-dir`.

## Relevant Product Docs

- `README.md`
- `docs/FEATURE_INTAKE.md`
- `docs/CONTEXT_RULES.md`

## Acceptance Criteria

- Sidebar includes an `API Keys` item in the `Gateway` group and opens
  `/api-keys`.
- `/api-keys` renders a real management page instead of redirecting to
  `/config`.
- The page uses direct CRUD through `apiKeysApi` for list, add, edit, delete,
  and copy flows.
- Successful API key mutations refresh config-backed consumers without a manual
  reload.
- Config Panel visual mode no longer shows `Authentication Configuration`.
- Visual mode no longer reads or writes `auth-dir` or `api-keys`.
- Raw source YAML mode remains untouched for power users.

## Design Notes

- Commands: API key mutations go through the existing `/api-keys` management
  API.
- Queries: the dedicated page loads API keys directly from `apiKeysApi.list()`.
- API: no backend contract changes.
- Tables: no schema changes.
- Domain rules: Postgres DB remains the runtime source of truth; `auth-dir`
  stays source-mode-only.
- UI surfaces: `ApiKeysPage`, main sidebar, config visual editor cleanup, and
  config-store refresh wiring for dependent pages.

## Validation

When updating durable proof status, use numeric booleans:
`scripts/bin/harness-cli story update --id <id> --unit 1 --integration 1 --e2e 0 --platform 0`.

| Layer | Expected proof |
| --- | --- |
| Unit | `cd web && bun run type-check` |
| Integration | `cd web && bun run build` |
| E2E | Not required; route/workflow change covered by build and static review in this slice |
| Platform | Not required; frontend-only management workflow change |
| Release | Not required |

## Harness Delta

No Harness policy changes.

## Evidence

- `cd web && bun run type-check` passed.
- `cd web && bun run build` passed.
- `git diff --check` passed.
