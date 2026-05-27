# IDEA: LLMHub Low-Delta Rebrand Monorepo

## Summary
Refactor the current fork into `LLMHub` as a monorepo where the Go backend and management web UI live in one repository and ship as one self-contained product.

The work is intentionally split into three phases:
- phase 1: rebrand the original project, import the web UI source into `web/`, and embed it into the Go binary while preserving behavior as closely as possible
- phase 2: refactor the web UI style and UX after phase 1 is stable
- phase 3: brainstorm later for new features, providers, and deeper product changes

## Why Now
- The current repo still reflects upstream branding and product identity.
- The current management panel is coupled to a runtime download model instead of living in this repo.
- There is currently no in-repo frontend source or toolchain.
- A one-repo, one-binary model simplifies ownership, build, and deployment.

## Chosen Direction
- Single repo
- Single distributable binary
- Frontend source under `web/`
- Upstream panel imported as-is first
- Frontend build output embedded with `go:embed`
- Product name `LLMHub`
- Module path `github.com/therealtinhtute/llmhub`
- Keep `/management.html` as the initial panel entrypoint
- Moderate cleanup only in phase 1

## Explicit Non-Goals For Phase 1
- Deep provider/runtime architecture redesign
- New proxy product capabilities
- UI redesign during the import step
- Broad cleanup that removes useful operational docs or changes too much source structure

## Rejected Alternatives
1. Aggressively clean the repo in phase 1.
   This would mix migration and redesign, making behavior parity harder to protect.
2. Keep the management panel in a separate repository or keep downloading it at runtime.
   This preserves the current external coupling and weakens monorepo ownership.
3. Redesign the web UI while importing it.
   This mixes phase 1 and phase 2 and creates too many moving parts at once.
4. Keep `panel-github-repository` as a deprecated config surface.
   Once the panel is embedded, that config becomes dead weight rather than compatibility value.
