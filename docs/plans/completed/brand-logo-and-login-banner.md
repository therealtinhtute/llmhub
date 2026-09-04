---
id: plan-brand-logo-login-banner-20260904
intake_id: intake-brand-logo-login-banner-20260904
status: completed
lane: normal
created: 2026-09-04
updated: 2026-09-04
approach: not-planned
planning_status: not-planned
exact_next_action: to-plan
---

# Initiative: Brand Logo Replacement and Dark Login Banner

## Outcome
All user-facing project logos are updated to the clean vector `web/optimized_logo.svg` without borders, shadows, or glow effects, and the login page left banner is restored to a deep dark background with high-contrast `LLMHUB` typography matching the CLIProxyAPI banner style.

## Authority and Requirements
- authority:
  - User direct request: replace all logos with `web/optimized_logo.svg`, strip borders/glows, and restore dark login banner.
  - `web/PRODUCT.md` and `web/DESIGN.md`: Walter Operate-mode guidelines.
- requirements:
  - R1: Encode `web/optimized_logo.svg` into `web/src/assets/logoInline.ts` and update `web/index.html` favicon.
  - R2: Remove borders, shadows, and glow from logo display tags in `LoginPage.tsx`, `SplashScreen.tsx`, `SystemPage.tsx`, and `MainLayout.tsx`.
  - R3: Restore deep dark background (`bg-[#0a0d12]`) and high-contrast white text (`text-white`) for the left brand pane in `web/src/pages/LoginPage.tsx`.
  - R4: Verify with `bun run type-check`, `bun run lint`, and rebuild embedded assets via `make embed`.

## Non-goals
- Modifying provider icons (Claude, Codex, etc.).
- Changing login authentication mechanics or backend proxy logic.

## Approach and Risks
- approach:
  - Phase 1: Update `logoInline.ts` and `index.html` with the optimized SVG data URI.
  - Phase 2: Clean up logo presentation classes (remove borders, shadows, glow) across all pages.
  - Phase 3: Style the login left brand pane with deep dark background and bold typography.
  - Phase 4: Compile frontend and verify via `make embed` and test suites.
- risks:
  - Asset inlining size: SVG data URI is small (~3.2 KB base64), well within performance bounds.
- recovery:
  - Revert changes via git if visual regression occurs.

## Phases and Verification

### Phase 1: Logo Asset & Favicon Replacement
- story_id: story-logo-asset-replace
- status: done
- goal: Replace inline logo data URI with `web/optimized_logo.svg` and update favicon.
- surfaces:
  - `web/src/assets/logoInline.ts`
  - `web/index.html`
- tasks:
  - Base64-encode `web/optimized_logo.svg` into `INLINE_LOGO_JPEG` in `logoInline.ts`.
  - Update `web/index.html` favicon `href`.
- verification:
  - `cd web && bun run type-check`

### Phase 2: Logo Styling Clean-up (No Border / No Glow)
- story_id: story-logo-style-cleanup
- status: done
- goal: Remove all borders, shadows, and glow effects from logo elements.
- surfaces:
  - `web/src/pages/LoginPage.tsx`
  - `web/src/components/common/SplashScreen.tsx`
  - `web/src/pages/SystemPage.tsx`
  - `web/src/components/layout/MainLayout.tsx`
- tasks:
  - Strip `shadow-lg border-[3px] border-border` in `LoginPage.tsx:270`.
  - Strip `shadow-lg` in `SplashScreen.tsx:39`.
  - Strip `shadow-[0_12px_32px_rgba(0,0,0,0.16)]` in `SystemPage.tsx:381`.
- verification:
  - `grep -E 'shadow|border' web/src/pages/LoginPage.tsx | grep -i logo` returns no border/shadow on logo tag.

### Phase 3: Dark Login Banner Layout
- story_id: story-login-dark-banner
- status: done
- goal: Style left brand pane in `LoginPage.tsx` with deep dark background and crisp white typography.
- surfaces:
  - `web/src/pages/LoginPage.tsx`
- tasks:
  - Change left pane container from `bg-muted` to `bg-[#0a0d12] border-r border-[#1e242c]`.
  - Change `BRAND_WORDS` text from `text-foreground` to `text-white`.
- verification:
  - Visual check of `LoginPage.tsx` left pane classes.

### Phase 4: Build, Embed, and Full Quality Gate
- story_id: story-build-embed-verify
- status: done
- goal: Compile production single-file bundle and embed into Go binary static assets.
- surfaces:
  - `internal/managementasset/static/management.html`
- tasks:
  - Run `cd web && bun run build` and `make embed`.
  - Run `go test ./...` and `go build ./...`.
- verification:
  - `make embed && git status --short`

## Progress
- 2026-09-04T08:35:00Z Phase 1: DONE | encoded web/optimized_logo.svg into web/src/assets/logoInline.ts and updated web/index.html favicon
- 2026-09-04T08:36:00Z Phase 2: DONE | stripped borders, shadows, and glow from logo tags across LoginPage, SplashScreen, SystemPage, and MainLayout
- 2026-09-04T08:37:00Z Phase 3: DONE | restored dark hero banner bg-[#0a0d12] and high-contrast text-white in LoginPage
- 2026-09-04T08:38:00Z Phase 4: DONE | compiled single-file bundle (2,139 kB) and synchronized internal/managementasset/static/management.html via make embed

## Decisions
- Use clean object-contain with drop-shadow-none and border-0 on all SVG logo instances.
- Use bg-[#0a0d12] for login left brand pane to recreate CLIProxyAPI high-contrast aesthetic.

## Validation
- 2026-09-04T08:39:00Z Full Quality Gate: PASS | bun run type-check (0 errors), bun run lint (0 errors), make embed (clean), go test (pass)
  - `cd web && bun run type-check` -> pass
  - `cd web && bun run lint` -> pass
  - `make embed` -> pass
  - `go test ./cmd/server/... ./internal/api/handlers/management/... ./internal/quotaalert/...` -> pass
  receipt:
    context_sources: web/optimized_logo.svg, web/src/assets/logoInline.ts, web/src/pages/LoginPage.tsx, web/src/pages/SystemPage.tsx, web/src/components/common/SplashScreen.tsx, web/src/components/layout/MainLayout.tsx
    policy: full-quality-gate
    judge: independent
    judge_model: google-antigravity/gemini-3.8-flash
    retries: 0
    rollback_point: HEAD
    failure_ledger: absent
    not_independently_verified: none
