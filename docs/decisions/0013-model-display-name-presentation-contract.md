# 0013 Model Display Name Presentation Contract

Date: 2026-07-23

## Status

Accepted

## Context

The registry already carries optional human-readable model display names, but
configured Claude, Codex, Gemini, Vertex-compatible, OpenAI-compatible, and OAuth
alias models could not supply them consistently. Some listing handlers omitted
the field, Codex client templates could overwrite configured presentation, and
Gemini CLI did not expose registry models.

A display name is presentation metadata. Using it as a routing key, upstream
model name, alias, provider selector, or authentication selector would change
request behavior and could break existing clients.

The audited upstream range also contains model expansion that is unsafe to port
as metadata alone. GPT-5.6 and Kimi K3 require coupled executor behavior and are
outside this targeted backport. Grok 4.5 and the selected Gemini production IDs
can be represented by the existing local model contracts.

## Decision

Add optional `display-name` fields to configured Claude, Codex, Gemini,
Vertex-compatible, OpenAI-compatible, and OAuth alias model entries. Trim and
preserve the value through sanitization, registration, prefix clones, Codex
built-in replacement, and OAuth alias forks.

Populate only `ModelInfo.DisplayName`. Keep the configured alias as `ModelInfo.ID`
and keep the upstream name, provider, model type, auth selection, prefixes, and
routing keys unchanged. When no explicit display name is configured, retain the
existing protocol-appropriate fallback.

Expose presentation metadata through the existing listing contracts:

- OpenAI and Codex client catalogs use `display_name`;
- Claude uses `display_name`, falling back to the unchanged model ID;
- Gemini and Gemini CLI use `displayName` while retaining the existing model
  `name`.

Add Grok 4.5 and the selected Gemini production IDs to the static registry. Keep
the corresponding Gemini preview IDs alongside the production IDs. Do not add
GPT-5.6 or Kimi K3 in this phase.

## Alternatives Considered

1. Replace model IDs with friendly names. Rejected because IDs are public routing
   keys and must remain stable.
2. Derive a display name only inside each HTTP handler. Rejected because aliases,
   prefix clones, and Codex built-in replacement would still lose configured
   presentation before listing.
3. Remove preview Gemini IDs when production IDs are added. Rejected because both
   identifiers may remain valid client-visible routing targets.
4. Add all upstream model IDs as static metadata. Rejected because GPT-5.6 and
   Kimi K3 require behavior not included in this backport.

## Consequences

Positive:

- Model catalogs can show configured human-readable names consistently across
  OpenAI, Codex, Claude, Gemini, and Gemini CLI protocols.
- Display customization does not alter routing, auth, provider selection, or
  upstream request names.
- Production and preview Gemini IDs coexist additively.
- Grok 4.5 is advertised with the existing xAI capability schema.

Tradeoffs:

- Configuration schemas gain one optional presentation field per model family.
- Handlers must preserve protocol-specific field casing.
- Static model metadata still requires explicit coupled review when a new model
  needs executor or header behavior.
