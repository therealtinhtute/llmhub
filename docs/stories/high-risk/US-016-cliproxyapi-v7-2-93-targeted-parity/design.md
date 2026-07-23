# Design

## Source and Baseline

- Upstream: `router-for-me/CLIProxyAPI` `v7.2.93`, commit `01f387f4`.
- Local implementation baseline: `2960b690bf89232b8a23c5b8823fbe0ca831347f`.
- Strategy: targeted adaptation behind existing llmhub interfaces.

## Runtime Contract

Requests continue through existing handlers, translators, executor registry,
and auth manager. Postgres remains authoritative for runtime configuration,
auth records, cooldown snapshots, and usage. Amp and Kiro ownership is
preserved.

## Phase Contracts

### WebSocket close 1009

A Gorilla WebSocket close `1009` already observed for the active connection is
mapped to a typed request-scoped 413-equivalent `message_too_big` error. The auth
manager does not mark, cool down, refresh, or fall back credentials for that
classification. Ordinary write and non-1009 errors keep existing retry behavior.

### Auth reliability

Cooldown escalation advances at most once per active window. Jitter is clamped
before it is added. Structured or textual `invalid_grant` enters the existing
30-minute suspension path. Generic unsupported CountTokens endpoint failures
remain availability-neutral; explicit `model_not_found` does not.

### Translator fidelity

A shared pure helper normalizes raw base64 and explicit-MIME data URLs. Claude
tool-result arrays remain structured only when all members map to valid Claude
content blocks; other JSON values become deterministic compact JSON text.
Provider `output_index`, including zero, wins when present.

### Tool protocol

Codex and xAI use request-scoped declaration tables that retain original and
effective tool identity. Effective-name collisions fail before network I/O,
existing `mcp__` names stay unchanged, and undeclared provider-internal xAI
search lifecycles are suppressed completely.

### Model presentation

Approved Grok and Gemini IDs are additive. Configured display names are
presentation metadata only and never become routing, alias, auth, or upstream
identity.

## Evidence and Reversibility

Each implementation slice owns an exact file allowlist and produces a full-index
patch, test log, review result, and forward/reverse application proof. Product
fingerprints exclude append-only harness/control-plane records. Parallel slices
start from one immutable phase base in isolated worktrees and are applied
serially after review.
