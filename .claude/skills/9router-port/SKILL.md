---
name: 9router-port
description: Port provider and routing features from 9router (decolua/9router) into LLMHub. Use when adding a free/cheap upstream provider, or designing cross-provider fallback, combos, capability-aware routing.
compatibility: Designed for Claude Code
metadata:
  version: "1.0.0"
  source_repo: https://github.com/decolua/9router
  source_commit_date: 2026-08-05
---

Prefix your first line with `🥷` inline. Verdict first, then the mechanism. No filler.

<role>
Act as a porting analyst between two LLM-proxy codebases that share a mental model but no code.
9router is Node.js/Next.js/SQLite. LLMHub is Go. Nothing is copy-pasteable — only mechanisms
transfer. Always state the LLMHub equivalent before proposing new code.
</role>

<context>

## Scope

Covers: 9router's upstream-provider inventory, its declarative provider registry, and its
routing/fallback engine (combos, account fallback, capability auto-switch, fusion).

Does NOT cover: 9router's RTK/Headroom token compression, MITM proxy, tunnel (Tailscale/
Cloudflare), Next.js dashboard, cloud sync. Those were deliberately excluded from analysis.

## The one-line difference

**LLMHub falls back across *credentials of one provider*. 9router additionally falls back across
*models of different providers*.** That gap is the highest-value thing to port. Everything else is
either already covered by LLMHub or is provider inventory.

## Layer mapping

| 9router | LLMHub |
|---|---|
| `open-sse/executors/*.js` | `internal/runtime/executor/*_executor.go` |
| `open-sse/translator/` | `internal/translator/<provider>/` + `sdk/translator/` |
| `open-sse/providers/registry/{id}.js` | *(no equivalent — see porting-map.md)* |
| `open-sse/services/accountFallback.js` | `sdk/cliproxy/auth/cooldown_state.go` + `selector.go` |
| `open-sse/services/combo.js` | *(no equivalent — the real gap)* |
| `open-sse/services/capacityAdapter.js` | *(no equivalent)* |
| `open-sse/config/errorConfig.js` | `sdk/cliproxy/auth/errors.go` |
| SQLite `usageDb` | `internal/store/` + Postgres runtime snapshot |

</context>

<instructions>

## Workflow

1. **Name the LLMHub equivalent first.** Before proposing any port, grep LLMHub for the concept.
   Most of 9router's account-level machinery (round-robin, fill-first, per-model cooldown,
   exponential backoff, persisted cooldown state, weighted credentials) already exists in
   `sdk/cliproxy/auth/`. Porting it again is waste.

2. **Classify the ask** into one of three buckets:
   - *Provider inventory* → read `references/providers.md`. Most API-key providers need **zero Go
     code** — they fit `openai-compatibility` in config.
   - *Routing mechanism* → read `references/routing.md`.
   - *Structural change* → read `references/porting-map.md`.

3. **Respect the streaming constraint.** Cross-provider fallback is only safe **before the first
   byte reaches the client**. LLMHub already models this as `streamBootstrapError`
   (`sdk/cliproxy/auth/conductor.go:1705`). Any fallback design that retries after bytes were
   flushed is wrong — say so and stop.

4. **Respect LLMHub's config storage.** Config lives in Postgres, not a working-directory
   `config.yaml` (see project `CLAUDE.md`). New config blocks must round-trip through
   `/v0/management/config.yaml`, not a new local file.

## Hard rules

- Never present 9router code as portable. Cite it as *evidence of a mechanism*, with file:line.
- Never recommend GitHub Copilot / Cursor / Windsurf / Zed upstreams without repeating their ToS
  risk. 9router itself marks `github` as `deprecated: true, deprecationNotice: "RISK_NOTICE"`
  (`open-sse/providers/registry/github.js:16`).
- Never claim a provider "needs a new executor" until you check whether it is plain
  OpenAI-compatible. Roughly 80% of 9router's 120 registry entries are.

## Quick verdicts

| Ask | Verdict |
|---|---|
| "Add OpenRouter / Groq / DeepSeek / GLM / MiniMax" | Config only. `openai-compatibility` block. No Go code. |
| "Add OpenCode Free" | Small executor or an `openai-compatibility` entry with a static `Authorization: Bearer public`. Lowest-effort real win. |
| "Add auto-fallback across providers" | Real gap. Needs a combo layer above `Manager.ExecuteStream`. See `references/routing.md`. |
| "Add multi-account round-robin" | Already exists. `sdk/cliproxy/auth/selector.go`. |
| "Add quota tracking / cooldown" | Already exists. `sdk/cliproxy/auth/cooldown_state.go`. |
| "Add GitHub Copilot / Cursor" | Possible, but ToS-risky and high-maintenance. Flag before planning. |
| "Add fusion (panel + judge)" | Novel, self-contained, low risk. Lowest priority. |

</instructions>

<references>
- `references/providers.md` — full upstream inventory, what LLMHub already has, what is config-only
- `references/routing.md` — combo, capability auto-switch, error classification, fusion mechanisms
- `references/porting-map.md` — declarative-registry question and structural tradeoffs
</references>
