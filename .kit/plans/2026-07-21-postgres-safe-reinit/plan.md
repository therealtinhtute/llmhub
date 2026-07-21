# PostgreSQL safe re-initialization plan

## Decisions

- Use the repository `.env` without printing secrets.
- Perform additive, idempotent schema initialization only; do not drop or truncate anything.
- Import local `config.yaml` and `auths/` while preserving existing database config and skipping duplicate auth IDs.

## Steps

1. Validate prerequisites
   - Confirm required PostgreSQL variables are available through `.env`.
   - Build or locate the current `llmhub` binary.
   - Verify: connection/bootstrap command reaches PostgreSQL successfully.

2. Initialize schema safely
   - Run `init-db-from-env -env-file .env`.
   - This creates missing schema objects and seeds config only when the database has no config row.
   - Verify: command exits successfully and a second run also succeeds, proving idempotency.

3. Migrate local data safely
   - Run `migrate-local-to-db -env-file .env -config config.yaml -auth-dir auths` without `-overwrite-auth`.
   - Existing database config remains unchanged; duplicate auth IDs are skipped.
   - Verify: command reports successful migration/import with no overwrite flag.

4. Validate runtime
   - Start a bounded smoke test using the same PostgreSQL environment, or run focused PostgreSQL integration checks when the test DSN is available.
   - Verify: runtime loads its config from PostgreSQL and no startup/schema error appears.

## Safety constraints

- No `DROP`, `TRUNCATE`, schema reset, or auth overwrite.
- Do not print `.env`, DSN, credentials, tokens, or auth payloads.
- Stop immediately if connection, schema initialization, or migration fails; report the exact failing stage without continuing.
