# SPEC: LLMHub Rebrand Monorepo With Embedded Web UI

Status: draft
Input Type: new-initiative
Lane: high-risk
Risk Flags: auth, external-systems, public-contract, cross-platform, existing-behavior, multi-domain
Affected Surfaces: api, browser, desktop, provider, docs
Downstream: plan full
Updated At: 2026-05-27

## Source Mode
refine

## Source Inputs
- Existing `.kit/planning/SPEC.md` draft for LLMHub monorepo refactor
- Existing `.kit/planning/IDEA.md`
- User clarification that the work must be split into three phases:
  - phase 1: rebrand the original project, keep source behavior as close to upstream as possible, import the web UI source, and embed it into this repo
  - phase 2: refactor the web UI style and UX after phase 1 is stable
  - phase 3: brainstorm later for feature additions, provider additions, and larger product evolution
- Repo facts gathered during refinement:
  - there is currently no in-repo frontend source or frontend build tooling
  - the management UI is currently served from a downloaded `management.html`
  - `/management.html` is the current management entrypoint route
  - the current config includes `remote-management.panel-github-repository`
  - the current management asset updater still depends on upstream release and fallback URLs

## Scenario
project bootstrap

## Goal
Turn this fork into `LLMHub` as a monorepo product with one Go backend and one embedded web UI, while keeping phase 1 as a low-delta upstream rebrand and packaging migration rather than a broad product rewrite.

## Users / Actors
- Primary maintainer: the fork owner evolving `LLMHub`
- Operators: people running the proxy locally or on servers
- Management UI users: operators managing auths, config, logs, and runtime status
- Downstream developers: developers building from source or importing packages from this fork

## Requirements
1. The repository must become the single source of truth for both backend and management frontend, with frontend source living under `web/`.
2. Phase 1 must preserve upstream runtime behavior and structure as much as possible, with changes limited to rebranding, monorepo packaging, and embedded panel delivery.
3. Phase 1 must import the upstream management panel source into `web/` with minimal restructuring, changing only what is required to build inside this repo and embed into the Go binary.
4. The production build must package frontend assets into the Go server binary using `go:embed`, so the management UI no longer depends on a separate repository, downloaded asset, or second deployment unit.
5. The server must stop serving a runtime-downloaded `management.html` file and must instead serve embedded frontend assets from the Go binary.
6. `/management.html` must remain the management UI entrypoint in phase 1 for compatibility, even if it is internally backed by an embedded app rather than a local downloaded file.
7. The Go module path must change from `github.com/router-for-me/CLIProxyAPI/v7` to `github.com/therealtinhtute/llmhub`, and source imports in this repository must be updated accordingly.
8. Product-visible identity must be changed from upstream naming to `LLMHub` in runtime banners, README and core docs, packaged UI branding, binary naming, and other visible product surfaces.
9. The default auth storage path must change from `~/.cli-proxy-api` to `~/.llmhub`.
10. Phase 1 must keep existing management API routes and existing config keys unless a key is made obsolete by the embedded-panel architecture.
11. `remote-management.panel-github-repository` must be removed from the target architecture in phase 1 because embedded panel delivery makes that config path meaningless.
12. `remote-management.disable-control-panel` must remain meaningful in phase 1, because operators may still need to disable the bundled UI.
13. Phase 1 cleanup must be moderate, not aggressive: remove sponsor blocks, promotional content, and ecosystem listing noise from core product docs, while keeping operational docs and source surfaces still relevant to the fork.
14. Phase 2 must be explicitly reserved for UI and UX refactoring after the imported panel is stable and embedded.
15. Phase 3 must be explicitly reserved for later brainstorming about new features, new providers, and deeper runtime or product changes.
16. Phase 1 must not include deep executor, provider, routing, auth, or thinking-pipeline redesign beyond the minimum required to support rebranding and embedded panel delivery.

## Boundaries
### In Scope
- Introduce an in-repo `web/` frontend as the management UI source of truth
- Import the upstream management panel source with minimal change
- Replace runtime panel download with embedded asset delivery
- Keep `/management.html` as the panel entrypoint in phase 1
- Rename module path and visible product identity to `LLMHub`
- Rename packaging defaults such as auth directory and product naming
- Keep core operational docs while removing obvious promo and ecosystem noise
- Preserve management API compatibility in phase 1
- Reserve phase 2 and phase 3 explicitly in the spec so implementation does not drift

### Out of Scope
- Rebuilding the management UI from scratch in phase 1
- Refactoring the web UI visual style in phase 1
- Changing provider selection, retry behavior, auth scheduling, thinking pipeline logic, or executor architecture in phase 1
- Adding new management API capabilities in phase 1
- Adding new providers or new product features in phase 1
- Removing every upstream artifact regardless of usefulness
- Changing `/management.html` to a brand new route in phase 1

## Constraints
- Follow a monorepo mental model: one repo, one product, one build story.
- Keep phase 1 as a low-delta fork migration rather than a broad cleanup or architecture rewrite.
- Preserve operator-facing compatibility for proxy behavior and management workflows wherever practical.
- There is currently no in-repo frontend toolchain, so the spec must assume new frontend tooling will be introduced and owned by this repo.
- The current management serving path is coupled to runtime download helpers and a physical file path; implementation planning must account for removing that coupling safely.
- Existing auth and management flows are sensitive surfaces and must avoid regressions.
- The local worktree does not provide a committed baseline, so planning should assume a fork-owned reshape rather than history-sensitive surgical commits.

## Phase Structure
### Phase 1
- Full product rebrand to `LLMHub`
- Import upstream panel source into `web/`
- Build and embed frontend assets into the Go binary
- Preserve behavior and routes wherever practical
- Do moderate doc cleanup only

### Phase 2
- Refactor web UI style, UX, and information architecture after phase 1 is stable
- Keep backend contracts from phase 1 stable unless a later spec says otherwise

### Phase 3
- Separate future brainstorm for new features, providers, and deeper runtime or product evolution

## Acceptance Criteria
- The spec explicitly defines phase 1 as a low-delta rebrand and embed effort, not a deep product rewrite.
- The spec explicitly defines phase 2 as the UI/UX refactor phase and phase 3 as later product expansion brainstorming.
- The spec locks the web UI baseline for phase 1 to an upstream panel import into `web/`.
- The spec locks production delivery to embedded frontend assets served by the Go binary.
- The spec locks `/management.html` as the phase-1 panel entrypoint.
- The spec locks module path migration to `github.com/therealtinhtute/llmhub`.
- The spec locks product naming to `LLMHub`.
- The spec states that `remote-management.panel-github-repository` is removed from the target architecture in phase 1.
- The spec states that `disable-control-panel` remains supported.
- The spec states that phase 1 cleanup is moderate and preserves core docs while removing obvious promo noise.
- In Scope and Out of Scope make it unambiguous that runtime/provider redesign is deferred.

## Validation Expectations
- Frontend proof:
  - `web/` exists as the in-repo source of truth for the management UI.
  - the imported panel source can build successfully inside the monorepo.
  - the resulting build output can be embedded into the Go binary.
- Backend proof:
  - Go code serves embedded frontend assets rather than fetched `management.html`.
  - `/management.html` still opens the management UI after the migration.
  - the binary builds successfully after module-path migration and embedded asset integration.
- Compatibility proof:
  - management API routes used by the imported panel remain functional in phase 1.
  - proxy runtime behavior is unchanged except for intentional branding, packaging, and panel-hosting changes.
  - `disable-control-panel` still disables panel access.
- Config proof:
  - runtime behavior no longer depends on `panel-github-repository`.
  - obsolete updater/config logic tied only to external panel download is removed or bypassed.
- Docs proof:
  - primary docs describe `LLMHub`, not the upstream product.
  - obvious sponsor, promo, and ecosystem list noise is removed from core docs.
  - operational docs still needed to build and run the fork remain available.

## Dependencies / Assumptions
- The canonical repository for this fork is `git@github.com:therealtinhtute/llmhub.git`.
- The fork brand name is `LLMHub`.
- The desired module path is `github.com/therealtinhtute/llmhub`.
- The phase-1 web UI baseline is the upstream CPAMC source imported into `web/`.
- `go:embed` is the preferred production distribution mechanism over runtime disk serving.
- Frontend tooling required by the imported panel will be introduced into this repo and maintained here.
- License attribution will be preserved where required even if sponsor and ecosystem content is removed.

## Key Decisions
- Chosen path: low-delta phase 1 rebrand plus in-repo web UI import and embed.
  - Rationale: this matches the user’s intent to keep the original project behavior and structure as intact as possible before any stylistic or product evolution work.
- Rejected alternative: aggressive repo cleanup in phase 1.
  - Why not: it would make phase 1 drift into a broad rewrite, increase migration risk, and obscure whether behavior parity was preserved.
- Rejected alternative: keep the panel in another repository and fetch or vendor build output.
  - Why not: it preserves the weakest coupling in the current design and fails the one-repo ownership goal.
- Rejected alternative: redesign the web UI during import.
  - Why not: it mixes phase 1 and phase 2, making it harder to prove that the imported UI still works before stylistic changes.
- Rejected alternative: keep `panel-github-repository` as a deprecated config key.
  - Why not: once the panel is embedded, the key no longer has real meaning and only keeps dead conceptual surface around.
- Rejected alternative: replace `/management.html` with a brand new route in phase 1.
  - Why not: the current code and tests already treat `/management.html` as a live contract, so changing it now adds risk without real product benefit.

## Open Questions
- None currently blocking this spec.

## Deferred Ideas
- Full UI redesign and style system work after the embedded panel is stable
- New providers, new product features, and broader runtime changes
- SDK/package-surface redesign after the module migration settles
- Larger doc reorganization beyond the moderate cleanup required in phase 1

## Ambiguity Report
- Goal clarity: high
- Scope clarity: high
- Constraints clarity: high
- Acceptance clarity: high
