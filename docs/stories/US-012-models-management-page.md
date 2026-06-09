# US-012 Models Management Page

## Status

implemented

## Lane

normal

## Product Contract

Management Center exposes a dedicated Models page where operators can review
currently available proxy models and manage OAuth model alias mappings without
going through Management Center Info.

## Relevant Product Docs

- `web/README.md`
- `docs/FEATURE_INTAKE.md`

## Acceptance Criteria

- Sidebar includes a Models item that opens `/models`.
- Dashboard Available Models links to `/models`.
- Management Center Info no longer renders the Available Models section.
- `/models` shows the `/v1/models` inventory with refresh, status, empty/error,
  grouping, and search behavior.
- `/models` includes OAuth model alias management using the existing
  `/oauth-model-alias` APIs.
- Opening the alias editor from `/models` returns to `/models` after save/back;
  opening it from Auth Files keeps the existing Auth Files return behavior.

## Design Notes

- Commands: OAuth model alias saves/deletes remain owned by the existing
  management API.
- Queries: model inventory uses the existing `/v1/models` fetch path and saved
  API key resolution.
- API: no backend API changes.
- Tables: no schema changes.
- Domain rules: alias scope is OAuth model alias only.
- UI surfaces: `ModelsPage`, main sidebar, Dashboard quick stat, and the
  existing OAuth model alias editor.

## Validation

When updating durable proof status, use numeric booleans:
`scripts/bin/harness-cli story update --id <id> --unit 1 --integration 1 --e2e 0 --platform 0`.

| Layer | Expected proof |
| --- | --- |
| Unit | `cd web && bun run type-check` |
| Integration | `cd web && bun run build` |
| E2E | Browser smoke of `/models`, `/system`, dashboard link, and alias editor return flow |
| Platform | Not required; frontend-only route/workflow change |
| Release | Not required |

## Harness Delta

No Harness policy changes.

## Evidence

- `cd web && bun run type-check` passed.
- `cd web && bun run build` passed.
- `git diff --check` passed.
- Playwright smoke with mocked management endpoints passed for `/models`
  inventory + alias rendering and confirmed `/system` no longer renders the
  old Available Models description.
