# US-003 Local Installer Same-Directory Binary

## Status

implemented

## Lane

normal

## Product Contract

`scripts/install-local.sh` supports local VPS installs when `install-local.sh`,
`llmhub`, and optionally `config.example.yaml` and `.env` are copied into the
same directory and run from there. The local installer keeps port `9090` as its
default while preserving existing environment overrides such as `DEFAULT_PORT`.

## Relevant Product Docs

- `README.md`
- `scripts/install-local.sh`

## Acceptance Criteria

- With no binary path argument, the installer first checks for `llmhub` next to
  the script, then `./llmhub`, then repo-root `dist/llmhub-linux-*` artifacts.
- An explicit binary path argument installs that exact binary after validation.
- New configs are rendered with host `0.0.0.0`, port `9090`, and
  `auth-dir: "/var/lib/llmhub/auths"` by default.
- Existing configs and env files preserve their content while normalizing the
  configured writable path, host, and port to the selected defaults.
- Error messages mention the same-directory package layout.

## Design Notes

- Commands: shell installer only.
- Queries: none.
- API: none.
- Tables: none.
- Domain rules: local installer defaults differ from release downloader
  installer defaults.
- UI surfaces: none.

## Validation

When updating durable proof status, use numeric booleans:
`scripts/bin/harness-cli story update --id US-003 --unit 1 --integration 0 --e2e 0 --platform 0`.

| Layer | Expected proof |
| --- | --- |
| Unit | Shell syntax and static behavior review. |
| Integration | Not expected in this workspace. |
| E2E | Not applicable. |
| Platform | Linux systemd smoke is required before claiming VPS runtime proof. |
| Release | Not applicable. |

## Harness Delta

No Harness behavior changes expected.

## Evidence

- `sh -n scripts/install-local.sh` passed.
- Static review confirmed no `SELECTED`-dependent config/env helper is called
  before explicit or discovered binary selection.
- `rg -n "DEFAULT_PORT|Found script-local|Found local binary|Place llmhub|install.sh|8317|9090" scripts/install-local.sh README.md scripts/install.sh`
  confirmed local default `9090`, release installer `8317`, and same-directory
  user-facing messages.
- User-reported Postgres mode systemd failure showed missing writable runtime
  base caused metadata to fall back to `/etc/llmhub/pgstore`; installer now
  normalizes `WRITABLE_PATH=/var/lib/llmhub` in existing env files.
- Full Linux systemd smoke was not run in this macOS workspace.
