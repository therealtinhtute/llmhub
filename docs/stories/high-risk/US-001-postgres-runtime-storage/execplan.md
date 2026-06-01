# Exec Plan

## Goal

Make Postgres a pure runtime storage backend for cliproxy config, auth, and
recent management usage when `PGSTORE_DSN` is configured.

## Scope

In scope:

- Idempotent Postgres schema for config, auth, and usage events.
- DB boot precedence and first-boot local seeding.
- DB-backed config/auth management behavior.
- DB polling for runtime config/auth reloads.
- Recent usage queue persistence.
- Supabase/Postgres documentation.

Out of scope:

- Logs migration.
- Model cache migration.
- Harness SQLite migration.
- Long-term analytics dashboard.
- A config YAML storage block.

## Risk Classification

Risk flags:

- Auth.
- Data model.
- Audit/security.
- External systems.
- Public contracts.
- Existing behavior.
- Weak proof.
- Multi-domain.

Hard gates:

- Auth.
- Data migration.
- External provider behavior.

## Work Phases

1. Create Harness story and ADR.
2. Replace Postgres spool behavior with direct DB store methods.
3. Wire Postgres boot, config parsing, auth seeding, and DB watcher.
4. Wire management config/auth and recent usage persistence.
5. Update docs.
6. Run focused and full verification.

## Stop Conditions

Pause for human confirmation if:

- Local mode behavior must change.
- DB startup failure should fall back to local files.
- Retention or token payload storage requirements need to exceed recent usage.
- Validation requirements need to be weakened.
