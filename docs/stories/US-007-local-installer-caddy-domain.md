# US-007 Local Installer Optional Caddy Domain

## Status

implemented

## Lane

normal

## Product Contract

`scripts/install-local.sh` can optionally configure a Debian/Ubuntu VPS domain
through Caddy while preserving the existing local-binary install path. The
installer prompts for a domain in interactive runs, accepts `CADDY_DOMAIN` for
automation, and leaves the current direct IP/port flow unchanged when the
domain is blank.

## Relevant Product Docs

- `README.md`
- `scripts/install-local.sh`

## Acceptance Criteria

- After binary selection and before install, the script prompts for a domain
  when stdin is interactive and `CADDY_DOMAIN` is unset.
- Blank domain input skips Caddy and preserves the current app-port exposure.
- Non-blank domain input must pass hostname validation before any Caddy setup.
- In domain mode, the installer installs Caddy on Debian/Ubuntu, writes a
  reverse-proxy `Caddyfile`, validates it, restarts `caddy`, and reports HTTPS
  endpoints.
- Existing `Caddyfile` content is backed up before overwrite and restored if
  generated config validation fails.
- In domain mode, `ufw` rules for `80/tcp` and `443/tcp` are added only when
  `ufw` is already installed and active.

## Design Notes

- Commands: shell installer only; `apt-get`, `systemctl`, `caddy`, `curl`.
- Queries: none.
- API: none.
- Tables: none.
- Domain rules: one hostname only; no wildcard or multi-site support.
- UI surfaces: terminal prompt only.

## Validation

When updating durable proof status, use numeric booleans:
`scripts/bin/harness-cli story update --id US-007 --unit 1 --integration 0 --e2e 0 --platform 0`.

| Layer | Expected proof |
| --- | --- |
| Unit | `sh -n scripts/install-local.sh` plus static review for prompt and domain branch behavior. |
| Integration | Not expected in this workspace. |
| E2E | Not applicable. |
| Platform | Ubuntu VPS smoke is required before claiming Caddy/domain runtime proof. |
| Release | Not applicable. |

## Harness Delta

No Harness behavior changes expected.

## Evidence

- `sh -n scripts/install-local.sh`
- Static review of `valid_domain`, `prompt_domain`, Debian/Ubuntu Caddy
  install path, and `configure_caddy` rollback handling.
- README examples updated for prompted domain flow and `CADDY_DOMAIN` override.
- Full VPS/domain smoke was not run in this macOS workspace.
