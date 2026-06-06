# US-010 Postgres Reseed-Safe Installer Bootstrap

## Status

implemented

## Lane

normal

## Product Contract

`scripts/install-local.sh` and `llmhub init-db-from-env` require
`LLMHUB_INIT_CONFIG_YAML` or `LLMHUB_INIT_CONFIG_B64` only when the Postgres
runtime config row is still empty. Re-running the installer against an
already-seeded VPS must work with the existing `.env` file and DB state.

## Relevant Product Docs

- `README.md`
- `scripts/install-local.sh`
- `cmd/server/db_runtime.go`

## Acceptance Criteria

- `install-local.sh` reuses existing `PGSTORE_*` and `LLMHUB_INIT_CONFIG_*`
  values from the destination env file instead of ignoring them.
- Interactive installs may skip the YAML paste prompt when the operator is
  targeting an already-seeded Postgres runtime.
- `llmhub init-db-from-env` exits successfully without init config env when the
  Postgres config row already exists.
- First-boot behavior still fails clearly when Postgres has no config row and
  no init config env is provided.

## Design Notes

- Commands: shell installer and Go runtime bootstrap command.
- Queries: Postgres config version lookup before init seed.
- API: none.
- Tables: existing Postgres config table only.
- Domain rules: init config is bootstrap-only, not a permanent runtime
  requirement after the DB is seeded.
- UI surfaces: terminal prompt only.

## Validation

When updating durable proof status, use numeric booleans:
`scripts/bin/harness-cli story update --id US-010 --unit 1 --integration 0 --e2e 0 --platform 0`.

| Layer | Expected proof |
| --- | --- |
| Unit | `go test ./cmd/server` and `sh -n scripts/install-local.sh`. |
| Integration | Live Postgres smoke is still required outside this workspace. |
| E2E | Not applicable. |
| Platform | Linux VPS rerun smoke remains external to this macOS workspace. |
| Release | Not applicable. |

## Harness Delta

No Harness behavior changes expected.

## Evidence

- `go test ./cmd/server`
- `sh -n scripts/install-local.sh`
- Static review confirmed reruns can skip YAML paste, env-file values are
  reused, and `init-db-from-env` now checks current config version before
  requiring init config env.
