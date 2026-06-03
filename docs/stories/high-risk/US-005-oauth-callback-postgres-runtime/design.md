# Design

## Callback Handoff

The OAuth session store now carries one optional callback payload for each
pending state. Callback endpoints submit code/state/error into that session, and
the provider login goroutine waits for and consumes the payload.

This removes the previous dependency on writing `.oauth-*.oauth` marker files
under `cfg.AuthDir` for management OAuth flows.

## Persistence Boundary

The callback payload is short-lived process memory only. Durable credential
persistence remains unchanged:

- provider goroutine exchanges the OAuth code for tokens
- auth record is built with provider metadata and storage
- `h.saveTokenRecord` calls the configured token store
- Postgres mode persists through `PostgresStore.Save`

## Compatibility

The legacy callback-file helper remains available for callers that still expect
a file marker, but management callbacks and browser redirect routes now use the
session-store handoff directly.
