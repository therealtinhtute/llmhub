# Plan: Verification

Phase: verification
Status: ready
Wave Count: 2
Execution Owner: work
Updated At: 2026-05-28

## Goal
Validate the complete migration: build passes, no stale artifacts, token count correct, visual appearance matches warm cream/beige palette, Go binary builds successfully.

## Inputs
- Phases 1-3 complete
- `.kit/planning/SPEC.md` — validation expectations

## Wave 1
### T1 — Build verification
- type: test
- inputs:
  - Complete `web/` codebase after Phases 1-3
- touches:
  - None (read-only)
- avoid:
  - File modifications
- steps:
  1. Run `cd web && bun run build` — must succeed
  2. Verify `dist/` contains a single HTML file
  3. Run `make build` from project root — must succeed (Go binary with embedded frontend)
- expected outputs:
  - Both builds succeed
- verification:
  - `ls -la web/dist/*.html` shows output
  - Go binary exists at expected location
- stop if:
  - Either build fails
- escalate to:
  - plan phase token-foundation if CSS-related build failure
  - plan phase component-adoption if component-related build failure

### T2 — Grep cleanup proofs
- type: test
- inputs:
  - Complete codebase after Phases 1-3
- touches:
  - None (read-only)
- avoid:
  - File modifications
- steps:
  1. `grep -rn "\.dark" web/src/index.css` — expect zero `.dark` blocks
  2. `grep -rn "@custom-variant dark" web/src/index.css` — expect zero
  3. `grep -rn "dark:" web/src/components/ web/src/features/ web/src/pages/ --include="*.tsx" --include="*.ts"` — expect zero `dark:` utility classes (string content "dark" is OK)
  4. `grep -c "^  --" web/src/index.css` — expect ~90+ custom properties
  5. `grep -c "oklch" web/src/index.css` — expect high count
- expected outputs:
  - All grep checks pass
- verification:
  - Each grep command returns expected result
- stop if:
  - Any check finds stale artifacts
- escalate to:
  - Appropriate earlier phase for remaining cleanup

## Wave 2
### T3 — Visual inspection
- type: test
- inputs:
  - Running dev server
- touches:
  - None (read-only)
- avoid:
  - File modifications
- steps:
  1. Start dev server: `cd web && bun run dev`
  2. Check each route: Dashboard, Providers, AuthFiles, Config, Logs, OAuth, System, Quota
  3. Verify warm cream/beige background (#faf9f4 equivalent)
  4. Verify blue accent (#117dff equivalent) on primary buttons and links
  5. Verify no purple/cool tones from old palette
  6. Verify sidebar renders correctly
  7. Verify form inputs, buttons, badges, cards all render with new tokens
  8. Verify no visual regressions (broken layouts, invisible text, missing borders)
- expected outputs:
  - All routes render with warm cream palette, blue accent, correct typography
- verification:
  - Visual confirmation across all routes
- stop if:
  - Visual regression found
- escalate to:
  - Appropriate earlier phase for visual fix

## Risks / Watch-fors
- `make build` requires `make embed` first — ensure correct build sequence
- Dev server hot reload may not reflect all CSS changes — hard refresh needed
- Visual inspection is subjective — focus on obvious regressions, not pixel-perfect matching
