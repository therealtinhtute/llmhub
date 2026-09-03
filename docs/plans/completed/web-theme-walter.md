---
id: plan-20260903-web-theme-walter
type: plan
intake_id: intake-20260903-web-theme-walter
lane: tiny
status: completed
created: 2026-09-03
updated: 2026-09-03
---

# Plan: Update web UI to tweakcn Walter theme

## Outcome
- result: The LLMHub React management panel (`@web/`) adopts the tweakcn "Walter" design system (theme `cmssq6yz7000004kw9woi94p0`), delivering a warm cream/charcoal aesthetic with deep teal accents, Bricolage Grotesque typography, refined 0.3rem border radii, and a functioning light/dark mode switcher.
- success_signals:
  - `web/src/index.css` `:root` (light) and `.dark` CSS tokens match the Walter theme values.
  - `web/index.html` loads `Bricolage Grotesque` and `Xanh Mono` from Google Fonts; `--font-sans` is updated.
  - `web/src/stores/useThemeStore.ts` fixes the `.dark` class toggling so users can switch between Light and Dark modes.
  - `bun run build` in `web/` passes type checking and Vite build cleanly.
  - `make embed` successfully synchronizes `internal/managementasset/static/management.html`.

## Authority and Requirements
- authority:
  - Tweakcn theme payload `https://tweakcn.com/themes/cmssq6yz7000004kw9woi94p0` (ID: `cmssq6yz7000004kw9woi94p0`, Name: "Walter").
  - `web/src/index.css` (Tailwind CSS v4 token declarations and `@theme inline` variables).
  - `web/index.html` (Google Fonts imports and font links).
  - `web/src/stores/useThemeStore.ts` (Zustand theme store).
  - `CLAUDE.md` (Frontend verification policy: no new test files under `web/`, verify via build & browser runtime).
- rejected_alternatives:
  - Light-mode only adaptation — rejected because Walter provides calibrated light & dark styling; omitting dark mode would degrade dark mode usability.
  - Hardcoding hex values across individual component TSX files — rejected in favor of centralized CSS custom properties in `index.css`.
  - Replacing Geist fonts without fallback — rejected; font stacks will keep graceful system fallbacks.
- requirements:
  - R1 [accepted]: Update `web/src/index.css` `:root` light mode variables to match Walter: background `#faf8f5`, foreground `#141414`, card `#ffffff`, card-foreground `#141414`, popover `#ffffff`, popover-foreground `#141414`, primary `#124b68`, primary-foreground `#f7f9fa`, secondary `#faf8f5`, secondary-foreground `#141414`, muted `#f7f9fa`, muted-foreground `#444444`, accent `#d9e6ee`, accent-foreground `#124b68`, destructive `#ef393e`, destructive-foreground `#ffffff`, border `#e5e5e5`, input `#e5e5e5`, ring `#124b68`, charts 1-5, radius `0.3rem`.
  - R2 [accepted]: Update `web/src/index.css` `.dark` mode variables to match Walter: background `#262624`, foreground `#c3c0b6`, card `#262624`, card-foreground `#faf9f5`, popover `#30302e`, popover-foreground `#e5e5e2`, primary `oklch(0.52 0.105 223.128)`, primary-foreground `#ffffff`, secondary `#faf9f5`, secondary-foreground `#30302e`, muted `#1b1b19`, muted-foreground `#b7b5a9`, accent `#1a1915`, accent-foreground `#f5f4ee`, destructive `#ef4444`, destructive-foreground `#ffffff`, border `#3e3e38`, input `#52514a`, ring `oklch(0.52 0.105 223.128)`, charts 1-5, radius `0.3rem`.
  - R3 [accepted]: Add Google Fonts link for `Bricolage Grotesque` and `Xanh Mono` in `web/index.html` and update font variables in `web/src/index.css` to prioritize `Bricolage Grotesque, ui-sans-serif, sans-serif, system-ui`.
  - R4 [accepted]: Restore `setTheme` and `initializeTheme` in `web/src/stores/useThemeStore.ts` so switching between `light` and `dark` toggles the `.dark` class on `<html>`.
  - R5 [accepted]: Verify frontend build with `bun run build` in `web/` and verify `make embed` updates `internal/managementasset/static/management.html`.

## Non-goals
- NG1: Changes to Go backend or proxy routes.
- NG2: Component DOM rewrites or layout changes.
- NG3: Introducing new test files in `web/` (violates `CLAUDE.md` frontend rule).

## Approach and Risks
- approach: Implement the Walter theme in dependency order across three focused phases:
  1. Update design tokens and typography: align `web/src/index.css` font variables and radius (`0.3rem`), and add Google Fonts link for Bricolage Grotesque and Xanh Mono in `web/index.html`.
  2. Restore theme switcher logic in `web/src/stores/useThemeStore.ts` to toggle `.dark` class on `document.documentElement` and persist the user's light/dark preference.
  3. Validate and embed: run `bun run build` in `web/` to ensure clean TypeScript compilation and asset bundling, then synchronize `internal/managementasset/static/management.html` via `make embed`.
- constraints:
  - Do not introduce new test files under `web/` (per `CLAUDE.md`).
  - Preserve component DOM structures and props; styling must flow through CSS custom properties.
  - Maintain graceful font fallbacks in font stacks (`ui-sans-serif, sans-serif, system-ui` and `ui-monospace, monospace`).
- rejected_alternatives:
  - Light-only theme: Rejected because Walter defines calibrated dark mode tokens; dropping dark mode breaks UI when users expect system/dark preference.
  - Inline style hacks: Rejected in favor of Tailwind v4 `@theme inline` and CSS variables in `web/src/index.css`.
- risks:
  - Font loading flicker (FOUT) before Google Fonts loads → mitigated by keeping preconnect tags and system font fallbacks in CSS.
  - Stale embedded HTML served by Go binary → mitigated by running `make embed` and checking diff of `management.html`.
- recovery:
  - If `bun run build` fails, inspect TypeScript errors or CSS syntax in `index.css` and fix immediately.
  - If dark mode styling has unintended contrast issues, adjust token values against Walter theme payload.

## Phases and Verification
<!-- Phase and task definitions are immutable after to-plan. Do not add task status fields. Append-only Progress is the sole task execution-status source. Only each phase lifecycle status changes to mirror DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. Each planned phase records phase_slug, story_id, status, goal, depends_on, waves, tasks, and checks. -->
- planning_status: planned
- phases:
  - phase_slug: phase-1-tokens-and-typography
    story_id: story-20260903-walter-tokens
    status: done
    goal: Align font imports, CSS font variables, and border radius (0.3rem) across index.html and index.css (color tokens verified matching Walter in OKLCH).
    depends_on: none
    allowed_surfaces:
      - `web/index.html`
      - `web/src/index.css`
    waves:
      - wave: 1
        tasks:
          - task: Add Google Fonts imports for Bricolage Grotesque and Xanh Mono to web/index.html.
            touches: `web/index.html`
            expected: `<link>` tag in `web/index.html` loads Bricolage Grotesque (wght 400..800) and Xanh Mono.
            check: `grep -E "Bricolage\+Grotesque" web/index.html`
          - task: Update font variables and radius (0.3rem) in web/src/index.css (:root, .dark, and @theme inline).
            touches: `web/src/index.css`
            expected: `--radius: 0.3rem;` and `--font-sans: 'Bricolage Grotesque', ui-sans-serif, sans-serif, system-ui;` in index.css.
            check: `grep -E "Bricolage Grotesque|0\.3rem" web/src/index.css`

  - phase_slug: phase-2-theme-store-toggle
    story_id: story-20260903-walter-theme-store
    status: done
    goal: Restore light/dark theme toggling and class manipulation in useThemeStore.ts so dark mode can be activated.
    depends_on: phase-1-tokens-and-typography
    allowed_surfaces:
      - `web/src/stores/useThemeStore.ts`
    waves:
      - wave: 1
        tasks:
          - task: Implement applyTheme and update setTheme, cycleTheme, and initializeTheme in useThemeStore.ts.
            touches: `web/src/stores/useThemeStore.ts`
            expected: `document.documentElement.classList.add('dark')` / `remove('dark')` called dynamically based on theme.
            check: `grep -E "classList\.add\('dark'\)|classList\.toggle\('dark'\)" web/src/stores/useThemeStore.ts`

  - phase_slug: phase-3-build-and-embed
    story_id: story-20260903-walter-build-embed
    status: done
    goal: Build the React web application and embed the updated bundle into internal/managementasset/static/management.html.
    depends_on: phase-2-theme-store-toggle
    allowed_surfaces:
      - `web/dist/`
      - `internal/managementasset/static/management.html`
    waves:
      - wave: 1
        tasks:
          - task: Compile the React application with bun run build in web/.
            touches: `web/dist/`
            expected: `web/dist/index.html` generated without TypeScript or Vite errors.
            check: `cd web && bun run build`
          - task: Embed compiled HTML into Go binary static assets via make embed.
            touches: `internal/managementasset/static/management.html`
            expected: `internal/managementasset/static/management.html` updated with new build.
            check: `make embed && git status --short internal/managementasset/static/management.html`

## Progress
- 2026-09-03T10:00:00Z phase-1-tokens-and-typography wave-1 phase-start: in-progress | starting font and radius alignment
- 2026-09-03T10:05:00Z phase-1-tokens-and-typography wave-1 task-1: DONE | added Bricolage Grotesque and Xanh Mono to Google Fonts in web/index.html
- 2026-09-03T10:08:00Z phase-1-tokens-and-typography wave-1 task-2: DONE | updated --font-sans, --font-serif, --font-mono, and --radius (0.3rem) in web/src/index.css
- 2026-09-03T10:10:00Z phase-1-tokens-and-typography wave-1 wave-summary: DONE | typography and radius aligned to Walter theme
- 2026-09-03T10:11:00Z phase-2-theme-store-toggle wave-1 phase-start: in-progress | starting theme store dark mode restore
- 2026-09-03T10:14:00Z phase-2-theme-store-toggle wave-1 task-1: DONE | implemented applyTheme and dynamic setTheme/cycleTheme/initializeTheme in web/src/stores/useThemeStore.ts
- 2026-09-03T10:15:00Z phase-2-theme-store-toggle wave-1 wave-summary: DONE | theme store light/dark toggling restored and persistent
- 2026-09-03T10:16:00Z phase-3-build-and-embed wave-1 phase-start: in-progress | starting web build and embed
- 2026-09-03T10:18:00Z phase-3-build-and-embed wave-1 task-1: DONE | compiled React app with bun run build in web/ without errors (dist/index.html 2,130.42 kB)
- 2026-09-03T10:19:00Z phase-3-build-and-embed wave-1 task-2: DONE | embedded build into internal/managementasset/static/management.html via make embed
- 2026-09-03T10:20:00Z phase-3-build-and-embed wave-1 wave-summary: DONE | frontend compiled and embedded static assets synchronized
- 2026-09-03T10:25:00Z phase-1-tokens-and-typography wave-1 polish: DONE | added Google Sans Code (wght@400..700) to Google Fonts in web/index.html to ensure --font-mono first family is loaded, re-built and re-embedded
## Decisions
- 2026-09-03: phase-1-tokens-and-typography / R1, R2: Verified existing :root and .dark tokens in web/src/index.css already match Walter theme hex values converted to OKLCH with high precision. Avoided unnecessary hex rewriting; focus on typography and radius delta.
- 2026-09-03: phase-1-tokens-and-typography / Fonts: Verified Google Sans Code is available on Google Fonts API; loaded alongside Bricolage Grotesque and Xanh Mono so --font-mono does not fall back to secondary families.
absorb: memory web-theme-store-runtime-check
## Validation
- 2026-09-03T10:30:00Z phase-3-build-and-embed check-full: APPROVED | judge: independent | judge_model: reviewer
  - `git diff --check`
  - `cd web && bun run lint`
  - `cd web && bun run build`
  - `make embed`
  - `go test ./internal/managementasset/...`
  - `go build -o /dev/null ./cmd/server`
  - browser runtime evaluation: computed styles and visual screenshots in both light and dark modes verified
  receipt:
    context_sources: web/index.html, web/src/index.css, web/src/stores/useThemeStore.ts, docs/PROJECT.md
    policy: targeted-semantic-ports
    judge: independent
    judge_model: reviewer
    retries: 0
    rollback_point: HEAD
    failure_ledger: absent
    not_independently_verified: none (visually and programmatically verified in browser runtime for both light and dark states)

## Current State
- active_story: none
- active_phase: none
- lifecycle_status: done
- active_task: none
- blockers: none
- open_items: none
- exact_next_action: none
