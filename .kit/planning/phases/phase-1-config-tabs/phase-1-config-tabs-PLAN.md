# Plan: Config Tabs Refactor

Phase: phase-1-config-tabs
Status: ready
Wave Count: 2
Execution Owner: work
Updated At: 2026-05-29

## Goal
Replace the sticky 2/3-column aside navigation in `VisualConfigEditor` with a Radix UI horizontal Tabs structure. All 6 sections accessible via tab triggers; no IntersectionObserver scroll-spy.

---

## Inputs
- `src/components/config/VisualConfigEditor.tsx` — current implementation with custom button nav + IntersectionObserver
- `src/components/config/ConfigSection.tsx` — section wrapper
- `package.json` — check Radix packages
- Radix UI Tabs documentation

---

## Wave 1 — Install dependency + inspect

### T1 — Verify/add @radix-ui/tabs package
- type: implementation
- inputs:
  - `package.json`
- touches:
  - `package.json`
  - `bun.lock` / `pnpm-lock.yaml`
- steps:
  1. Read `package.json` — check if `@radix-ui/tabs` is already listed
  2. If not present, run `bun add @radix-ui/tabs` (project uses Bun)
  3. Verify the package is installed and the import is resolvable
- expected outputs:
  - `@radix-ui/tabs` present in `package.json`
- verification:
  - `grep -r "from '@radix-ui/tabs'" web/src/` resolves without error
- stop if:
  - version conflict with existing `@radix-ui` packages
- escalate to:
  - user clarification

### T2 — Read ConfigSection.tsx for scroll-snap
- type: refactor
- inputs:
  - `src/components/config/ConfigSection.tsx`
- touches:
  - none (read-only inventory)
- steps:
  1. Read `ConfigSection.tsx` fully
  2. Note all CSS classes related to scroll-snap, horizontal overflow, flex-none width
  3. Identify if ConfigSection itself has any horizontal scroll-snap styles that would break after the nav refactor
- expected outputs:
  - Inventory of scroll-snap-related CSS on ConfigSection
- verification:
  - Read complete; no changes made
- stop if:
  - ambiguity about which CSS is owned by ConfigSection vs. the parent container in VisualConfigEditor
- escalate to:
  - user clarification

---

## Wave 2 — Implement Radix Tabs in VisualConfigEditor

### T3 — Refactor VisualConfigEditor.tsx — replace nav with Radix Tabs
- type: implementation
- inputs:
  - Updated `package.json` with `@radix-ui/tabs`
  - `ConfigSection.tsx` scroll-snap inventory
- touches:
  - `src/components/config/VisualConfigEditor.tsx`
- avoid:
  - ConfigPage routing or other pages
  - Visual | Source toggle
  - Any backend code
- steps:
  1. Import Radix Tabs: `import * as Tabs from '@radix-ui/react-tabs'`
  2. Wrap entire editor content in `<Tabs.Root>` (keep existing structure)
  3. Replace the sticky aside and mobile nav with a single `<Tabs.List>`:
     - Desktop: single row, no grid-column trick; use Radix Tabs `TabsList` with `Trigger` × 6
     - Mobile: `<Tabs.List>` naturally scrollable via `overflow-x-auto`; no 2-col grid
  4. Replace `<ConfigSection>` horizontal scroll container with vertical flex flow:
     - Remove `[scroll-snap-type:x_mandatory]` from the section wrapper div
     - Make each section a normal block (not width-restricted flex-none)
  5. Move each `<ConfigSection>` into a `<Tabs.Content>` with matching `value={sectionId}`
  6. Update `handleSectionJump` — now calls `setActiveTab(sectionId)` instead of `scrollIntoView`
  7. Remove all `IntersectionObserver` scroll-spy logic (no longer needed with tab-controlled active state)
  8. Move error badge rendering onto `Tabs.Trigger` — Radix provides `data-state="active"` to target with CSS
  9. Remove the mobile-specific duplicate nav (`isMobile ? ... : null` sidebar aside block)
- expected outputs:
  - `VisualConfigEditor.tsx` — Radix Tabs replaces custom button nav + scroll-spy
- verification:
  - `bun run dev` → navigate to `/config` → all 6 tabs clickable, content swaps on click
  - Mobile: tabs scroll horizontally, no broken layout
  - Error badge visible on the correct TabsTrigger
  - No `IntersectionObserver` code remaining
- stop if:
  - `ConfigSection` CSS breaks badly without horizontal scroll-snap (need to decide: keep section widths or fix ConfigSection)
- escalate to:
  - brainstorm refine
- notes:
  - The sticky aside on desktop is gone; tabs are just at the top of the content, not sticky
  - Mobile nav duplicate (`isMobile ? ...`) is removed — tabs handle both

### T4 — Verify ConfigSection layout after scroll-snap removal
- type: verification
- inputs:
  - Refactored `VisualConfigEditor.tsx`
- touches:
  - none (read-only verification)
- steps:
  1. Run `bun run dev`, inspect config page sections on desktop at ≥768px and <768px
  2. Check each section: does it still fill the content area correctly? Are inputs readable?
  3. Verify no horizontal overflow issues
- expected outputs:
  - Sections render as normal blocks; no broken layout
- verification:
  - Manual inspection complete
- stop if:
  - Sections collapse or overflow unexpectedly — may need `ConfigSection` style fix
- escalate to:
  - user clarification if widespread

---

## Risks / Watch-fors
- **Scroll-snap removal** — ConfigSection width depends on being in a horizontal scroll container; after refactor each section must be a standalone block. May need ConfigSection width adjusted.
- **Mobile nav removal** — the `isMobile ? sticky-top-mobile-nav : null` pattern is gone. Tabs handle both — verify on mobile.
- **@radix-ui/tabs version** — Radix UI primitives often release minor-versioned packages. The project's Radix v1.4.3 may or may not include tabs. Confirm via `bun add`.
