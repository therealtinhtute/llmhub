# Cleanup Plan

## Goal

Remove or retire legacy storage paths that still imply local files are the
runtime source of truth when `PGSTORE_DSN` is enabled, while preserving local
file mode as the default when `PGSTORE_DSN` is unset.

## Cleanup Principles

- Keep local file mode working exactly as today.
- Keep Git and object token stores unless a separate story explicitly removes
  them.
- In Postgres mode, avoid local spool files for config, auth, or usage data.
- Prefer explicit mode boundaries over scattered `if postgres` checks.
- Do not migrate logs, model cache, Harness SQLite, or long-term analytics in
  this cleanup track.

## Track 1: Runtime Boundary Tightening

1. Audit code paths that call `sdkAuth.GetTokenStore()` from management
   handlers and service boot.
2. Replace implicit global-store lookups with injected store dependencies where
   the caller already knows runtime mode.
3. Keep `tokenStoreWithBaseDir` only for local-compatible handlers, or rename it
   to describe the mixed-mode behavior.
4. Add tests proving Postgres mode never requires `AuthDir` filesystem access
   after first-boot seed.

## Track 2: Legacy Spool Removal

1. Identify all remaining `PGSTORE_LOCAL_PATH`, `ConfigPath`, `AuthDir`, and
   spool directory references.
2. Separate metadata paths used for labels/logs from actual read/write paths.
3. Remove any Postgres-mode writes to local spool files.
4. Update docs to mark `PGSTORE_LOCAL_PATH` as deprecated if it is no longer
   operationally needed.

## Track 3: Management API Consistency

1. Ensure list, upload, download, patch, delete, and batch operations all use
   the same DB-backed auth identity rules.
2. Add pathless-store tests for multipart upload, download, status patch, batch
   delete, and `delete all`.
3. Verify deleted Postgres auth rows do not reappear because runtime removal
   persisted a disabled replacement.
4. Keep response shapes compatible with the current web UI.

## Track 4: Config Source-Of-Truth Cleanup

1. Audit management config handlers for fallback writes to `config.yaml`.
2. Ensure Postgres mode saves through `ManagementConfigStore` only.
3. Ensure config reload can be triggered immediately after management writes
   and through DB version polling.
4. Add a regression test for DB-wins behavior when local `config.yaml` differs
   from the DB row.

## Track 5: Usage Queue Cleanup

1. Keep in-memory recent queue behavior for local mode.
2. Keep Postgres usage queue behavior behind `PGSTORE_DSN`.
3. Add endpoint-level tests proving `/v0/management/usage-queue?count=N`
   returns the same JSON shape from DB-backed usage rows.
4. Verify pruning honors the configured short retention and does not create
   long-term analytics state.

## Verification

Minimum proof for the cleanup track:

```text
go test ./internal/store ./internal/redisqueue ./internal/api/handlers/management ./sdk/cliproxy/...
go test ./cmd/server
go test ./...
LLMHUB_POSTGRES_TEST_DSN=... go test ./internal/store ./internal/api/handlers/management
```

Operational proof:

- Start with `make dev-pg`.
- Upload an auth JSON file from the management UI.
- Confirm `auth_store` contains the row.
- Reload the web UI and confirm the auth entry is still visible.
- Restart llmhub and confirm the same DB auth entry loads without local file
  access.
