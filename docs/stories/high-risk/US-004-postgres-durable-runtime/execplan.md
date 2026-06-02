# Exec Plan

## Goal

Make remote Postgres the only durable runtime surface for llmhub server mode
when `PGSTORE_DSN` is set.

## Work Items

1. Add a runtime-storage policy object for Postgres durable mode.
2. Wire that policy through server startup, hot reload, and embedded service
   startup.
3. Disable durable local app logging and request archive logging in that mode.
4. Skip local auth-dir creation in that mode.
5. Update operator docs and the durable decision record.
6. Prove the policy with focused tests plus the existing broader Go suite.

## Non-Goals

- no new env vars
- no Postgres log tables
- no `HOME_JWT` behavior changes
- no changes to helper export utilities outside server runtime
