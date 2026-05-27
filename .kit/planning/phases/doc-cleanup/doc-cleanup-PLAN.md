# Plan: Documentation Cleanup + Branding

Phase: doc-cleanup
Status: ready
Wave Count: 3
Execution Owner: work
Updated At: 2026-05-27

## Goal
Rebrand all user-facing docs to LLMHub. Remove sponsor/promo content. Update example config. Preserve operational docs.

## Inputs
- Phase module-rebrand completed (product identity settled)
- Phase embed-panel completed (config reflects embedded panel)
- `README.md`, `README_CN.md`, `README_JA.md`
- `config.example.yaml`
- `assets/` directory

## Wave 1
### T1 — Rebrand README.md
- type: docs
- inputs:
  - `README.md`
- touches:
  - `README.md`
- avoid:
  - removing installation, configuration, or build instructions
  - removing API usage documentation
  - changing code examples that are still valid
- steps:
  1. Read `README.md` fully to understand structure
  2. Replace product name: "CLI Proxy API" → "LLMHub" in title, description, and prose
  3. Replace repository references: `router-for-me/CLIProxyAPI` → `therealtinhtute/llmhub`
  4. Replace binary name references: `cli-proxy-api` → `llmhub`
  5. Replace auth directory references: `~/.cli-proxy-api` → `~/.llmhub`
  6. Remove sponsor/donation image blocks and links (likely in `assets/` references)
  7. Remove ecosystem partner listings and competitive feature matrices
  8. Remove links to `help.router-for.me` or replace with a note that docs are forthcoming
  9. Keep: installation steps, config reference, API route documentation, build instructions
- expected outputs:
  - `README.md` describes LLMHub, not CLI Proxy API
  - No sponsor/promo content remains
  - Operational documentation preserved
- verification:
  - `grep -in "CLI Proxy API\|router-for-me\|cli-proxy-api" README.md` — zero branding hits (code example references may remain if they show config files)
  - `grep -c "sponsor\|donation\|funding" README.md` — zero
- stop if:
  - README is >500 lines and requires major restructuring
- escalate to:
  - user clarification (scope of rewrite)

### T2 — Rebrand README_CN.md and README_JA.md
- type: docs
- inputs:
  - `README_CN.md`, `README_JA.md`
- touches:
  - `README_CN.md`
  - `README_JA.md`
- avoid:
  - changing Chinese/Japanese prose beyond branding replacements
  - removing operational content
- steps:
  1. Read both files to understand structure
  2. Apply same branding replacements as T1: product name, repo URL, binary name, auth dir
  3. Remove sponsor/promo sections matching English version
  4. Preserve operational content
- expected outputs:
  - Both files describe LLMHub
  - No sponsor/promo content
- verification:
  - `grep -in "CLI Proxy API\|router-for-me\|cli-proxy-api" README_CN.md README_JA.md` — zero branding hits
- stop if:
  - file structure is radically different from English README
- escalate to:
  - user clarification

## Wave 2
### T3 — Update config.example.yaml
- type: docs
- inputs:
  - `config.example.yaml`
  - Config changes from module-rebrand phase
- touches:
  - `config.example.yaml`
- avoid:
  - changing config keys that are still valid
  - removing operational config documentation
- steps:
  1. Remove the entire `panel-github-repository` entry and its comment (lines 33-34)
  2. Change `auth-dir: "~/.cli-proxy-api"` to `auth-dir: "~/.llmhub"` (line 37)
  3. Update any comments referencing "CLI Proxy API" to "LLMHub"
  4. Remove or update comments about "GitHub panel download" since panel is now embedded
  5. Update `disable-control-panel` comment to say "bundled management UI" instead of "asset download"
  6. Remove `disable-auto-update-panel` entry and its comment (no longer relevant with embedded panel)
- expected outputs:
  - No `panel-github-repository` in example config
  - No `disable-auto-update-panel` in example config
  - `auth-dir` defaults to `~/.llmhub`
  - Comments reflect embedded panel, not downloaded asset
- verification:
  - `grep "panel-github-repository\|disable-auto-update-panel\|cli-proxy-api" config.example.yaml` — zero matches
- stop if:
  - config.example.yaml has complex structure that could break with removals
- escalate to:
  - plan phase

### T4 — Clean up assets/ directory
- type: docs
- inputs:
  - `assets/` directory
  - Updated `README.md` (from T1)
- touches:
  - `assets/` — remove files no longer referenced
- avoid:
  - removing images still referenced in any README or doc
- steps:
  1. List all files in `assets/`
  2. Grep all markdown files for each asset filename
  3. Remove unreferenced sponsor/promo images (use `trash` not `rm`)
  4. Keep any images still referenced in docs
- expected outputs:
  - No orphaned sponsor images in `assets/`
- verification:
  - For each remaining file in `assets/`, `grep -rn <filename> *.md` returns at least one match
- stop if:
  - images are referenced from external URLs or non-obvious locations
- escalate to:
  - user clarification

## Wave 3
### T5 — Final branding verification
- type: test
- inputs:
  - All doc changes from Waves 1-2
- touches:
  - none (verification only)
- avoid:
  - making code changes
- steps:
  1. Full branding scan: `grep -rn "CLI Proxy API\|CLIProxyAPI\|cli-proxy-api\|router-for-me\|router-for\.me" --include="*.md" --include="*.yaml" --include="*.yml"` — should return zero for primary docs
  2. Check that `README.md` renders correctly (review markdown structure)
  3. Verify LICENSE file is preserved and unmodified
  4. Verify no broken image links in READMEs
- expected outputs:
  - Zero upstream branding in user-facing docs
  - LICENSE preserved
  - All markdown renders correctly
- verification:
  - grep returns zero lines for primary doc files
  - No broken references
- stop if:
  - branding references found in files outside allowed surfaces
- escalate to:
  - check (run `/check` for final review)

## Risks / Watch-fors
- Non-English READMEs may use different formatting conventions or have content not in the English version
- Some "upstream" links (to API docs, login endpoints) may be operationally required — don't remove blindly
- `config.example.yaml` is the first thing operators see — ensure all changes are correct and comments are clear
- `assets/` cleanup should use `trash` per project rules, not `rm`
