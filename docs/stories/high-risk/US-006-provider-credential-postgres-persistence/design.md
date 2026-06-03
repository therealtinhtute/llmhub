# Design

## Audited Surfaces

Store-backed auth paths:

- CLI OAuth logins use `sdkAuth.GetTokenStore` through the shared login manager.
- Gemini CLI login performs provider onboarding, then calls the global token
  store directly.
- CLI Vertex import calls the global token store directly.
- Management OAuth and Vertex import call `h.saveTokenRecord`.
- Raw auth JSON upload uses `authManager.Register` or `Update`, backed by the
  configured store.

Config-backed provider paths:

- Gemini, Claude, Codex, Vertex-compatible, OpenAI-compatible, Amp, and proxy
  API-key management endpoints mutate `h.cfg` then call `h.persist`.
- In Postgres mode, `h.persist` writes through `configStore.SaveConfig`.

## Persistence Failure Handling

`coreauth.Manager.Register` and `Update` now return `m.persist` errors instead
of discarding them. This makes database write failures visible to management
upload/import/edit handlers that already propagate the returned error.

## Runtime Visibility After Save

Management OAuth and Vertex import use `h.saveTokenRecord` to write through the
configured token store. After a successful store write, `saveTokenRecord` now
upserts the same auth record into the runtime auth manager with
`coreauth.WithSkipPersist(ctx)`. This makes the new credential visible
immediately in the management list and runtime manager while avoiding a duplicate
database write.

## Boundaries

Standalone model-fetch commands still intentionally use explicit local auth
directories and output files. They are local utility/export surfaces, not server
runtime credential persistence.
