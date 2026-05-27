# Check Report: tailwind-foundation

Phase: tailwind-foundation
Mode: full
Date: 2026-05-28 00:15
Verdict: APPROVED

## Scope Drift
**On target.** All changed files fall within the phase's allowed surfaces:
- `web/package.json` — deps added ✅
- `web/bun.lock` — lockfile updated ✅
- `web/vite.config.ts` — Tailwind plugin added ✅
- `web/src/index.css` — replaced with Tailwind + design tokens ✅
- `web/src/main.tsx` — import added ✅
- `web/index.html` — Google Fonts CDN ✅
- `web/components.json` — shadcn config (new) ✅
- `web/src/lib/utils.ts` — cn() helper (new) ✅
- `.kit/` artifacts — planning/run/notes ✅

No forbidden surfaces touched (no TSX components, no stores, no API, no router, no i18n, no SCSS deleted).

## Artifact Alignment
**Aligned.**
- Spec requirements covered: Req 1 (Tailwind v4 ✅), Req 2 (shadcn init ✅), Req 3 (design tokens ✅), Req 4 (Google Fonts ✅), Req 5 (light/dark themes ✅)
- Constraint: single-file build works ✅ (dist/index.html, 2.0MB, single file)
- Constraint: `@/` path alias preserved ✅
- Locked decisions: all honored (v4 CSS-first, Google Fonts CDN, two themes, design-token.json source)

## Gate Results
| Check | Result |
|-------|--------|
| type-check | ✅ pass — `tsc --noEmit` exit 0 |
| build | ✅ pass — `bun run build` → single file 2,022 KB |
| Tailwind inlined | ✅ — `--tw-` vars found in output |
| Google Fonts | ✅ — 2 matches for `fonts.googleapis.com` |
| Single file | ✅ — 1 file in dist/ |

## Review Findings

### 🟡 Minor: `--radius` calc produces negative values
`--radius-sm: calc(var(--radius) - 4px)` with `--radius: 0px` yields `-4px`. CSS `border-radius` clamps negative to 0, so this is functionally correct. shadcn default pattern — no action needed.

### 💡 Suggestion: `verbatimModuleSyntax` in tsconfig.app.json
`src/lib/utils.ts` uses `import { type ClassValue, clsx }` which works with `verbatimModuleSyntax`. Good — no issue.

### 💡 Suggestion: `.dark` vs `[data-theme='dark']` transition
Noted in implementation-notes.md. Will need `useThemeStore` update in Phase 3. Already tracked — no action now.

## Security
- No secrets, credentials, or API keys in diff
- No injection surfaces
- Google Fonts CDN links are to trusted `fonts.googleapis.com` / `fonts.gstatic.com`

## Doc Debt
None. No new invariants requiring documentation beyond what's captured in implementation-notes.md.

## Sign-off

```
scope:              on target
depth:              standard
artifact_alignment: ✅ aligned
gate:               ✅ pass (type-check, build)
review:             APPROVED
blockers:           0 critical, 0 major
autofix:            0 safe applied, 0 awaiting confirmation
verification:       bun run build → pass, bun run type-check → pass
```
