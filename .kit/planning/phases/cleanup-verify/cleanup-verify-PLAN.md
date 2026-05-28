# Plan: Cleanup & Verify

Phase: cleanup-verify
Status: ready
Wave Count: 3
Execution Owner: work
Updated At: 2026-05-28

## Goal
Delete all dead CSS and run full verification suite to prove migration is complete.

## Inputs
- `web/src/index.css` (SCSS aliases + global classes to delete)
- `web/src/App.css` (to delete)
- All prior phase work complete

## Wave 1 — Pre-deletion verification
### T1 — Verify zero consumers of SCSS aliases
- type: test
- steps:
  1. `grep -rE "var\(--(text-primary|text-secondary|bg-primary|bg-secondary|bg-tertiary|bg-hover|glass-|success-badge|failure-badge|count-badge|amber-|filter-color|filter-surface|warning-bg|warning-text|primary-color|error-color|border-color)" web/src/ --include="*.tsx" --include="*.ts"` — must return empty
  2. If any matches found, list them for fixing before proceeding
- expected outputs:
  - Zero matches
- verification:
  - grep returns empty
- stop if:
  - any consumer found — route back to style-alignment phase
- escalate to:
  - plan phase style-alignment

### T2 — Verify zero consumers of global utility classes
- type: test
- steps:
  1. `grep -rE "\"(status-badge|error-box|hint|item-list|item-row|item-meta|item-actions|item-title|item-subtitle|form-group|pill)\"" web/src/ --include="*.tsx"` — must return empty
  2. Also check: `grep -r "className=\"input\"" web/src/ --include="*.tsx"` — must return empty
- expected outputs:
  - Zero matches
- verification:
  - grep returns empty
- stop if:
  - any consumer found
- escalate to:
  - plan phase style-alignment

### T3 — Verify App.css is unused
- type: test
- steps:
  1. `grep -r "App.css" web/src/ --include="*.tsx" --include="*.ts"` — must return empty
- expected outputs:
  - Zero matches
- verification:
  - grep returns empty

## Wave 2 — Deletion
### T4 — Delete SCSS compatibility aliases from index.css
- type: refactor
- touches:
  - `web/src/index.css`
- steps:
  1. Read index.css
  2. Delete the second `:root` block containing SCSS aliases (--text-primary, --bg-primary, etc.)
  3. Delete the second `.dark` block containing dark overrides for aliases
- expected outputs:
  - No SCSS alias definitions in index.css
- verification:
  - `grep -E "text-primary|bg-primary|glass-bg|success-badge-bg|failure-badge-bg|count-badge-bg|amber-10|filter-color|warning-bg" web/src/index.css` returns no matches

### T5 — Delete global utility classes from index.css
- type: refactor
- touches:
  - `web/src/index.css`
- steps:
  1. Delete `.status-badge`, `.error-box`, `.hint`, `.item-list`, `.item-row`, `.item-meta`, `.item-actions`, `.item-title`, `.item-subtitle`, `.main-content`, `.request-log-modal`, `.form-group`, `.input`, `.pill` class definitions
- expected outputs:
  - No global utility class definitions in index.css
- verification:
  - `grep -E "\.status-badge|\.error-box|\.hint\b|\.item-list|\.item-row|\.form-group|\.pill\b" web/src/index.css` returns no matches

### T6 — Delete App.css
- type: refactor
- touches:
  - `web/src/App.css`
- steps:
  1. Delete `web/src/App.css`
- expected outputs:
  - File removed
- verification:
  - `ls web/src/App.css` returns "No such file"

## Wave 3 — Full verification suite
### T7 — Build verification
- type: test
- steps:
  1. `cd web && bun run build` — must succeed
  2. `ls web/dist/index.html` — must exist
  3. `grep "oklch" web/dist/index.html` — must have matches
- expected outputs:
  - Successful single-file build
- verification:
  - build exits 0, index.html exists

### T8 — Legacy reference audit
- type: test
- steps:
  1. `grep -r "Legacy" web/src/` — zero matches
  2. `grep -rE "VT323|Source Serif|JetBrains Mono" web/src/` — zero matches
  3. `grep -r "shadow-hard" web/src/` — zero matches
  4. `grep -r "font-display\|font-body" web/src/ --include="*.tsx"` — zero matches (if @theme removed them)
  5. `grep -r "rounded-none" web/src/components/ui/` — zero matches
- expected outputs:
  - All greps return empty
- verification:
  - zero matches on all checks

### T9 — Functional build test
- type: test
- steps:
  1. `cd /home/tinhpt/Lab/llmhub && make build` — full Go binary build
- expected outputs:
  - Binary builds successfully with new frontend embedded
- verification:
  - make build exits 0

## Risks / Watch-fors
- Deleting SCSS aliases before confirming all consumers are migrated will break styling — Wave 1 verification is critical
- `main-content` class might be used outside of TSX (e.g., in index.html) — verify
- App.css import might be in a file not covered by grep pattern — check thoroughly
