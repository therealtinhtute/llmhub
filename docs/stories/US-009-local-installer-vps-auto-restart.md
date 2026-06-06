# US-009 Local Installer VPS Auto-Restart

## Status

implemented

## Lane

normal

## Product Contract

`scripts/install-local.sh` installs a `systemd` unit that automatically brings
`llmhub` back on a VPS after repeated crashes without requiring manual
intervention. The generated restart policy stays aligned with the README manual
service example.

## Relevant Product Docs

- `README.md`
- `scripts/install-local.sh`

## Acceptance Criteria

- The generated `systemd` unit restarts `llmhub` automatically when the main
  process exits or crashes unexpectedly on a VPS.
- Repeated crash loops do not get stuck on the default `systemd` start-rate
  limiter before an operator can investigate.
- Operators can still override restart defaults by exporting installer
  variables before running the script.
- The README manual service snippet matches the generated restart policy.

## Design Notes

- Commands: shell installer only; `systemctl`.
- Queries: none.
- API: none.
- Tables: none.
- Domain rules: explicit `systemd` restart policy is part of the VPS install
  contract.
- UI surfaces: terminal output only.

## Validation

When updating durable proof status, use numeric booleans:
`scripts/bin/harness-cli story update --id US-009 --unit 1 --integration 0 --e2e 0 --platform 0`.

| Layer | Expected proof |
| --- | --- |
| Unit | `sh -n scripts/install-local.sh` plus static review of generated unit content and README snippet. |
| Integration | Not expected in this workspace. |
| E2E | Not applicable. |
| Platform | Linux `systemd` smoke is required before claiming live crash-restart proof. |
| Release | Not applicable. |

## Harness Delta

No Harness behavior changes expected.

## Evidence

- `sh -n scripts/install-local.sh`
- Static review confirmed the installer now writes `Restart=always`,
  `RestartSec=3s`, and `StartLimitIntervalSec=0`, with env overrides exposed as
  `SERVICE_RESTART`, `SERVICE_RESTART_SEC`, and `SERVICE_START_LIMIT_INTERVAL`.
- README manual service setup now matches the generated restart policy.
- Live VPS crash-loop smoke was not run in this macOS workspace.
