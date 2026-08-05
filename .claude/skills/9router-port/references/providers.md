# 9router upstream providers — inventory and gap vs LLMHub

Source: `decolua/9router`, `open-sse/providers/registry/` (≈120 entries), read 2026-08-05.

## Categories 9router uses

`category` field on each registry entry drives UI grouping and routing tier:

| category | meaning | examples |
|---|---|---|
| `oauth` | subscription account via OAuth | claude, codex, github, cursor, windsurf, zed, trae, qoder, iflow, kiro, antigravity |
| `apikey` | user-supplied API key | openai, anthropic, deepseek, groq, cerebras, together, fireworks, nvidia, siliconflow, llm7 |
| `freeTier` | API key, but has a usable free tier | openrouter, chutes, blackbox |
| `free` | **no credentials at all** | opencode, mimo-free |

## What LLMHub already has

claude · codex · gemini · gemini-cli · vertex · aistudio · xai · kimi · kiro · antigravity ·
openai-compat (generic)

## Config-only — no Go code needed

These are plain OpenAI-compatible `POST /chat/completions` with a bearer key. They fit LLMHub's
existing `openai-compatibility` config block (`internal/config/config.go:569`) verbatim:

openrouter · deepseek · groq · cerebras · together · fireworks · nvidia · siliconflow · mistral ·
cohere · perplexity · hyperbolic · sambanova · nebius · featherless · venice · chutes · blackbox ·
huggingface · glm · glm-cn · minimax · minimax-cn · llm7 · tokenrouter · vercel-ai-gateway ·
volcengine-ark · byteplus · baidu · tencent · morph · ollama · kilo-gateway

Example — OpenRouter (`open-sse/providers/registry/openrouter.js`):

```
baseUrl: https://openrouter.ai/api/v1/chat/completions
headers:  HTTP-Referer: <your app url>
          X-Title: <your app name>
free tier: 27+ free models, no card, 200 req/day (1000 after any credit purchase)
models endpoint: https://openrouter.ai/api/v1/models   (filter id endswith ":free")
passthroughModels: true   ← forward the client's model id untouched
```

`passthroughModels: true` is the notable bit: for aggregators, do not maintain a model list —
forward whatever the client asked for and let upstream 404 it. LLMHub's
`OpenAICompatibilityModel` currently requires an explicit `name`/`alias` pair, so an aggregator
means hand-listing models. A `passthrough: true` flag on the config block would remove that chore.

## No-auth free providers (the "free model" ask)

### OpenCode Free — `open-sse/providers/registry/opencode.js`, `open-sse/executors/opencode.js`

```
base:    https://opencode.ai
chat:    POST /zen/v1/chat/completions          (OpenAI format)
alt:     POST /zen/v1/messages                  (Claude format, currently unused)
models:  GET  /zen/v1/models
headers: Authorization: Bearer public
         x-opencode-client: desktop
         Accept: text/event-stream
auth:    none
passthroughModels: true
```

Lowest-effort real win. No OAuth, no token refresh, no signing. Either a ~60-line executor or an
`openai-compatibility` entry with a hardcoded key of literally `public`.

### MiMo Free — `open-sse/providers/registry/mimo-free.js`

Marked `hidden: true`. Xiaomi shut the free channel down ("MiMo free API service has ended").
**Do not port.** Kept here so nobody re-discovers it.

## OAuth subscription providers LLMHub lacks

| provider | auth flow | risk |
|---|---|---|
| `github` (Copilot) | OAuth **device code**, `deviceCodeUrl` + poll | 9router marks it `deprecated` + `RISK_NOTICE`. ToS/ban risk. |
| `cursor` | OAuth + **protobuf** wire format (`open-sse/utils/cursorProtobuf.js`, 904 lines) | High effort, high breakage rate |
| `windsurf`, `zed`, `trae`, `qoder`, `iflow`, `codebuddy-cn/intl`, `clinepass`, `kilocode`, `kimchi`, `devin-cli`, `grok-cli`, `commandcode` | per-vendor OAuth | Each is a maintenance treadmill; endpoints and client versions rotate |

GitHub Copilot's transport pins spoofed editor identity headers:

```
copilot-integration-id: vscode-chat
editor-version: vscode/1.110.0
editor-plugin-version: copilot-chat/0.38.0
user-agent: GitHubCopilotChat/0.38.0
x-github-api-version: 2025-04-01
X-Initiator: user
usage probe: GET https://api.github.com/copilot_internal/user
```

Those pinned versions are exactly the maintenance cost: they go stale and requests start failing.

## Non-LLM services 9router also routes

Out of scope for LLMHub today, listed so the inventory is honest: TTS (elevenlabs, cartesia,
playht, google-tts, edge-tts, deepgram), STT (assemblyai, deepgram, selfhosted), embeddings
(voyage-ai, jina-ai, nvidia), image (black-forest-labs, fal-ai, runwayml, recraft, stability-ai,
nanobanana), search (brave, tavily, exa, serper, linkup, searxng, google-pse), scrape (firecrawl,
jina-reader).
