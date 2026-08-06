# Provider presets

The provider create form (openai-compatibility) offers a preset dropdown that prefills base URL,
headers, sign-up link, and free-tier note for a small, hand-picked set of upstreams. Presets are a
convenience for filling in the form — selecting one does not create or authenticate anything by
itself; you still add the provider entry yourself and supply your own credential.

## OpenCode Free

The `opencode` preset points at OpenCode's Zen free tier (`GET /zen/v1/models`), which answers with
no signup or API key at all. There is no published contract for this endpoint: no SLA, no
deprecation notice, no rate-limit documentation. Treat it as best-effort — it can slow down, change
shape, or disappear without notice. If it breaks, that's expected behavior for an undocumented free
endpoint, not a regression in LLMHub.

## What the `verified` field claims

Every preset carries `verified: true|false` and, when `true`, a `verified_at` date.

- `true` means the `base_url`/`models_url` shape was probed live and returned `200` on the date in
  `verified_at`. That is the entire claim.
- `true` does **not** mean a chat completion was billed and streamed successfully. For presets that
  require a key we don't hold (`openrouter`, `nvidia`), that deeper check isn't possible from our
  side. Only `opencode` is verifiable end-to-end without credentials.
- `false` means the preset was transcribed from an external registry but never probed by us. Any
  preset added without a live probe ships `false`, and the panel renders an **unverified** badge
  next to it.

`verified: true` is a claim about one day, not a standing guarantee — an endpoint that answered
today can stop answering tomorrow with no change to the catalog.
