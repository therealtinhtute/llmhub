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

## Current Code Findings

Postgres mode is already the main runtime source for config, auth, and recent
usage, but the code still carries local-file concepts through several runtime
interfaces:

- `cmd/server/main.go` still builds `PGSTORE_LOCAL_PATH`, passes it as
  `PostgresStoreConfig.SpoolDir`, then uses `pgStoreInst.ConfigPath()` and
  `pgStoreInst.AuthDir()` as synthetic downstream values.
- `internal/store/postgresstore.go` still stores `spoolRoot`, `configPath`, and
  `authDir` even though config/auth data are no longer read from those paths in
  normal Postgres mode.
- `PostgresStore.Bootstrap` and `SeedAuthFromDirectory` correctly read local
  config/auth once for first-boot import. These are the only local file reads
  that should remain in Postgres mode.
- Management config handlers already prefer `ManagementConfigStore`, but the
  same handler type also owns local `configFilePath` writes. This keeps the
  boundary implicit.
- Management auth handlers branch on `PathlessAuthStore()` for DB behavior, but
  still use file-oriented names like `writeAuthFile`, `deleteAuthFileByName`,
  `tokenStoreWithBaseDir`, and `buildAuthFromFileData`.
- `sdk/auth.GetTokenStore()` remains a global fallback. It works, but it makes
  Postgres/local mode harder to reason about in handlers and builders.
- The local filesystem watcher remains correct for local, git, and object
  modes. Postgres mode uses `sdk/cliproxy/storage_watcher.go`, so cleanup should
  not remove the watcher package.
- Usage queue now has a clean `UsageStore` interface. The main cleanup need is
  endpoint-level proof, not structural removal.

## Cleanup Target

After cleanup, Postgres mode should have these properties:

- No runtime reads from synthetic config or auth paths after first boot import.
- No runtime writes to synthetic config or auth paths.
- No need for `PGSTORE_LOCAL_PATH` in normal operation.
- Management config operations route through an explicit DB config store.
- Management auth operations route through an explicit DB/pathless auth store.
- Local file mode remains the only mode that uses `config.yaml`, `auth-dir`, and
  filesystem watcher behavior as runtime sources.

## Non-Targets

- Do not remove `sdk/auth.FileTokenStore`; it is still the local default.
- Do not remove `internal/watcher`; it is still needed by local, git, and object
  backed modes.
- Do not remove `configFilePath` from the whole service in this track; some
  downstream code still needs a label or local-mode path.
- Do not change management API response shapes unless the web UI is changed in
  the same patch.

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

1. Replace `PostgresStoreConfig.SpoolDir` with an optional metadata label or
   remove it entirely if all callers can use a constant synthetic label.
2. Replace `PostgresStore.ConfigPath()` and `PostgresStore.AuthDir()` with
   explicit metadata methods, for example `DisplayConfigPath()` and
   `DisplayAuthDir()`, or move synthetic labels to server boot.
3. Stop assigning `cfg.AuthDir = pgStoreInst.AuthDir()` in Postgres mode unless
   a caller is proven to need the value only as a display label.
4. Remove `PGSTORE_LOCAL_PATH` from `cmd/server/main.go`, `Makefile`, and docs
   after tests prove it is not needed.
5. Keep first-boot local config/auth import paths as explicit parameters named
   for seeding, not runtime paths.

Exit criteria:

- `rg "PGSTORE_LOCAL_PATH|pgstore_local_path|SpoolDir|pgStoreLocalPath"`
  returns no runtime references.
- Postgres boot still seeds config from `-config` or `config.yaml` when DB is
  empty.
- Restarting with existing DB rows does not touch local config/auth paths.

## Track 3: Management API Consistency

1. Ensure list, upload, download, patch, delete, and batch operations all use
   the same DB-backed auth identity rules.
2. Add pathless-store tests for multipart upload, download, status patch, batch
   delete, and `delete all`.
3. Verify deleted Postgres auth rows do not reappear because runtime removal
   persisted a disabled replacement.
4. Keep response shapes compatible with the current web UI.
5. Rename file-oriented helpers or split them into local and pathless helpers:
   `writeAuthFile`, `deleteAuthFileByName`, `buildAuthFromFileData`, and
   `tokenStoreWithBaseDir`.
6. Move pathless-store behavior behind a small interface owned by the management
   handler, not inferred repeatedly from the global token store.

Exit criteria:

- DB-backed upload/list/download/patch/delete tests use a pathless store with
  no filesystem `AuthDir`.
- Local-file handler tests still prove actual file write/delete behavior.
- `buildAuthFileEntry` no longer needs pathless special cases based only on
  `Attributes["source"]`.

## Track 4: Config Source-Of-Truth Cleanup

1. Audit management config handlers for fallback writes to `config.yaml`.
2. Ensure Postgres mode saves through `ManagementConfigStore` only.
3. Ensure config reload can be triggered immediately after management writes
   and through DB version polling.
4. Add a regression test for DB-wins behavior when local `config.yaml` differs
   from the DB row.
5. Split local config persistence from DB config persistence so `persistLocked`
   does not need to know both backends.
6. Keep comment-preserving YAML writes only in local mode. DB mode can preserve
   raw YAML through `PutConfigYAML`, but field-level updates currently marshal
   `h.cfg`; document or fix this if comment preservation in DB is required.

Exit criteria:

- Management field updates in Postgres mode call only `SaveConfig`.
- `GetConfigYAML` in Postgres mode reads only `LoadConfigBytes`.
- DB-wins regression test covers startup and management reload.

## Track 5: Usage Queue Cleanup

1. Keep in-memory recent queue behavior for local mode.
2. Keep Postgres usage queue behavior behind `PGSTORE_DSN`.
3. Add endpoint-level tests proving `/v0/management/usage-queue?count=N`
   returns the same JSON shape from DB-backed usage rows.
4. Verify pruning honors the configured short retention and does not create
   long-term analytics state.

Exit criteria:

- Local mode uses in-memory queue when no `UsageStore` is configured.
- Postgres mode pops from `usage_events` and marks rows with `popped_at`.
- Retention proof covers old popped and unpopped rows.

## Recommended Execution Order

1. Add failing tests first for Postgres mode without usable local paths:
   management auth upload/list/download/delete, management config get/put, and
   restart DB-wins behavior.
2. Split mode-specific management dependencies:
   `ConfigPersistence` for config and `AuthFilePersistence` for auth.
3. Remove implicit global store lookups from management handlers.
4. Remove `PGSTORE_LOCAL_PATH` and Postgres spool fields.
5. Rename remaining local-only helpers to make the local boundary explicit.
6. Run full verification and update README to describe the simpler Postgres
   mode.

## Risk Controls

- Do not start cleanup by deleting `internal/watcher` or `FileTokenStore`; that
  would break local mode.
- Do not remove `configFilePath` from `Service` until all constructor callers
  have a replacement display label or local-mode path.
- Keep first-boot import tests separate from DB-wins tests so local seed reads
  cannot accidentally become runtime reads.
- Run a real Postgres integration test with `LLMHUB_POSTGRES_TEST_DSN` before
  declaring spool removal complete.

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
