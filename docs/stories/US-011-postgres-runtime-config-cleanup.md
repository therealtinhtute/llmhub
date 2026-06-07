# US-011 Postgres Runtime Config Cleanup

## Status

implemented

## Lane

normal

## Product Contract

The repo no longer ships root-level `config.yaml` artifacts for normal server
startup. Local docs and container examples point at the Postgres plus env
bootstrap flow instead of a working-directory YAML file.

## Relevant Product Docs

- `README.md`
- `CLAUDE.md`
- `docker-compose.yml`

## Acceptance Criteria

- Root-level `config.yaml` and `config.example.yaml` are removed from the repo.
- Agent guidance and local runtime docs describe `init-db-from-env` plus
  Postgres as the server startup path.
- Docker examples do not mount a working-directory `config.yaml`.
- Helper fetch commands no longer require a local config file just to resolve
  the default auth directory.

## Design Notes

- Commands: `llmhub init-db-from-env`, `llmhub`
- Queries: none
- API: none
- Tables: Postgres runtime config snapshot only
- Domain rules: keep legacy `migrate-local-to-db` and management `/config.yaml`
  payload paths intact
- UI surfaces: none

## Validation

When updating durable proof status, use numeric booleans:
`scripts/bin/harness-cli story update --id US-011 --unit 1 --integration 1 --e2e 0 --platform 0`.

| Layer | Expected proof |
| --- | --- |
| Unit | Static review of config-path references in changed docs and Docker assets. |
| Integration | Focused Go package tests still pass after cleanup. |
| E2E | Not required. |
| Platform | Local binary startup remains DB/env driven; Docker runtime not live-smoked in this workspace. |
| Release | Not applicable. |

## Harness Delta

No Harness behavior changes expected.

## Evidence

- `ls -l config.yaml config.example.yaml` reports both files missing from the
  repo root after cleanup.
- `docker compose config` passed after switching the local container example to
  `init-db-from-env` plus `.env`.
- `go test ./cmd/...` passed after removing stale config-file lookups from the
  model-fetch helper commands.
- `env GOCACHE=/private/tmp/llmhub-gocache go test ./cmd/server ./internal/runtime/executor ./sdk/cliproxy ./internal/registry`
  passed on 2026-06-07.
