# Context: Documentation Cleanup + Branding

Phase: doc-cleanup
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: low
Expected Proof: visual inspection

## Goal
Update all user-facing documentation to reflect LLMHub branding. Remove sponsor blocks, promotional content, and ecosystem listing noise. Update the example config to reflect embedded panel (no `panel-github-repository`). Preserve operational documentation needed to build, configure, and run the product.

## Scope Boundary
### Allowed Surfaces
- `README.md` — full rebrand, promo removal
- `README_CN.md` — full rebrand, promo removal
- `README_JA.md` — full rebrand, promo removal
- `config.example.yaml` — remove `panel-github-repository`, update `auth-dir` default, update comments
- `assets/` — remove sponsor/promo images no longer referenced
- `LICENSE` — preserve (attribution required)
- `.github/` — update issue templates or CI config if they reference old branding

### Forbidden Surfaces
- Go source code — completed in prior phases
- `web/` — Phase 2
- `internal/` — completed in prior phases
- `go.mod`, `go.sum` — completed in prior phases

## Spec Hooks
- Requirement 8: product-visible identity changed to LLMHub in README and core docs
- Requirement 13: moderate cleanup — remove sponsor/promo/ecosystem noise, keep operational docs
- Constraint: keep phase 1 as low-delta, not aggressive cleanup

## Locked Decisions
- README rebrand replaces product name only where it appears as branding, not as technical identifiers in code examples (those should already reference LLMHub from prior phases)
- Sponsor image blocks and donation/funding sections are removed
- Ecosystem partner listings and comparative feature matrices are removed
- Installation and usage instructions are updated to reference `llmhub` binary name
- Config examples referencing `panel-github-repository` are removed
- `auth-dir` example value updated to `~/.llmhub`
- License file is preserved with original attribution
- Non-English READMEs get the same branding treatment as English

## Assumptions
- README structure is similar across all three language versions
- Sponsor images in `assets/` are only used by README files
- No external documentation site needs updating (spec says `help.router-for.me` is upstream)

## Canonical Refs
- `README.md` — primary docs
- `README_CN.md`, `README_JA.md` — translated docs
- `config.example.yaml` — user-facing config template
- `assets/` — images referenced by READMEs

## Rejected Options
- Full documentation rewrite — rejected: spec says moderate cleanup only
- Remove all upstream references including operational docs — rejected: some upstream docs are still operationally relevant
- Create new documentation site — rejected: Phase 3 scope

## Deferred Ideas
- Larger doc reorganization and information architecture — Phase 3
- Documentation site or wiki — Phase 3
- Contribution guide updates — Phase 3

## Escalate If
- README contains embedded links to upstream services that operators depend on (login URLs, API endpoints)
- Non-English READMEs have substantially different structure that requires separate treatment
