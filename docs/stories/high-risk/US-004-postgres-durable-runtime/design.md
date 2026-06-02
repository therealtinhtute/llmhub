# Design

## Runtime Policy

Introduce one explicit runtime-storage policy gate for "Postgres durable mode".
Downstream components branch on that gate instead of re-deriving behavior from
config fields or store types.

Policy effects in Postgres durable mode:

- skip local auth-dir creation during server startup
- force app logging to stdout/stderr even if `logging-to-file: true`
- replace request/error archive logging with a no-op logger
- report management log surfaces as disabled

## Ownership Contract

- durable config owner: Postgres `config_store`
- durable auth owner: Postgres `auth_store`
- durable usage owner: Postgres `usage_events`
- bootstrap-only local inputs: `-config` / `config.yaml`, configured `auth-dir`
- allowed local temp files: transient temp payload files only

Synthetic Postgres config/auth paths remain compatibility labels for components
that require non-empty strings, but they are not durable mirrors.

## Failure Semantics

- DB connection failure remains startup-fatal
- there is no fallback to local durable runtime storage in Postgres mode
- removing local durable logging is explicit and documented; no DB log table is
  added in this iteration
