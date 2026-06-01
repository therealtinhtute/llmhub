# Design

## Domain Model

Postgres mode introduces a storage boundary for runtime config, auth records,
and recent usage events. Local file mode keeps the existing file store and
watcher behavior.

## Application Flow

Boot precedence:

1. `PGSTORE_DSN` unset: unchanged local/git/object store behavior.
2. `PGSTORE_DSN` set and DB config exists: load config bytes from DB.
3. `PGSTORE_DSN` set and DB config is missing: seed DB from local config, then
   parse from DB bytes.
4. First DB boot with empty auth table: import existing auth JSON files from
   configured `auth-dir`.

Runtime updates:

- Management config writes save directly to Postgres and trigger in-process
  config reload.
- A storage watcher polls DB config/auth versions every 2 seconds.
- Management auth upload, patch, download, and delete operate through the
  DB-backed auth manager/store in Postgres mode.
- Recent usage payloads are appended to `usage_events`; management queue pops
  mark rows with `popped_at`.

## Interface Contract

Existing env activation is preserved:

- `PGSTORE_DSN` enables Postgres mode.
- `PGSTORE_SCHEMA` scopes tables to an optional schema.
- `PGSTORE_USAGE_RETENTION_SECONDS` overrides usage queue retention, clamped by
  existing retention normalization.

Existing management API routes and response shapes remain unchanged.

## Data Model

Tables:

- `config_store(id text primary key, content text not null, version bigint not null default 1, created_at timestamptz not null default now(), updated_at timestamptz not null default now())`
- `auth_store(id text primary key, provider text not null default 'unknown', content jsonb not null, created_at timestamptz not null default now(), updated_at timestamptz not null default now())`
- `usage_events(id bigserial primary key, payload jsonb not null, requested_at timestamptz not null, popped_at timestamptz, created_at timestamptz not null default now())`

Indexes:

- `auth_store(provider)`
- `usage_events(popped_at, requested_at, id)`
- `usage_events(created_at)`

## UI / Platform Impact

No frontend route changes. Operators configure Postgres through environment
variables.

## Observability

Startup logs identify Postgres runtime store activation and first-boot auth
imports. DB connection failure is startup-fatal and does not fall back to local
tokens.

## Alternatives Considered

1. Continue using the local spool mirror. Rejected because it keeps two mutable
   sources of truth.
2. Add a full analytics schema for usage. Rejected because this story only needs
   recent management queue durability.
