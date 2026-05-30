# Plan: Config Tabs Layout Fix

Phase: config-tabs-layout
Status: ready
Wave Count: 1
Execution Owner: work
Updated At: 2026-05-30

## Goal
Convert the config editor's tab navigation from a multi-column grid to a single horizontal scrollable row, remove the redundant info bar, and compact the tab triggers.

## Inputs
- `web/src/components/config/VisualConfigEditor.tsx` (current state)
- `.kit/planning/SPEC.md` Task 1

## Wave 1
### T1 — Replace TabsList grid with flex row
- type: refactor
- inputs:
  - `web/src/components/config/VisualConfigEditor.tsx`
- touches:
  - `Tabs.List` className (line 341)
- avoid:
  - `Tabs.Content` rendering behavior
  - `ConfigSection` component
- steps:
  1. Read `VisualConfigEditor.tsx`
  2. Replace `Tabs.List` className from `grid grid-cols-[repeat(3,minmax(0,1fr))] max-[1200px]:grid-cols-[repeat(2,minmax(0,1fr))] gap-2 min-w-0` to `flex flex-row overflow-x-auto gap-2 min-w-0`
  3. On each `Tabs.Trigger`, add `flex-none` so triggers don't shrink below their natural width
- expected outputs:
  - TabsList renders as a single horizontal row
  - Triggers scroll horizontally on narrow screens
- verification:
  - `cd web && npx tsc --noEmit` — no type errors
  - Visual: open config page in browser, verify single row at 1200px+ and horizontal scroll at 375px
- stop if:
  - Triggers collapse or overlap at desktop widths
- escalate to:
  - brainstorm refine (trigger sizing)

### T2 — Remove 快速跳转 info bar
- type: refactor
- inputs:
  - `web/src/components/config/VisualConfigEditor.tsx`
- touches:
  - The info bar div block (lines 323-339 in current file)
- avoid:
  - Validation logic — only remove the rendering, not the `hasValidationIssues` computed value (may be used elsewhere)
- steps:
  1. Delete the entire info bar block: the `<div className="grid grid-cols-[minmax(0,1fr)] gap-[10px] pb-[18px] ...">` containing "快速跳转", active section name, and validation badge
  2. Check if `hasValidationIssues` or `activeSection` variables are still referenced — remove if unused
- expected outputs:
  - No info bar rendered above the tabs
  - No unused variable warnings
- verification:
  - `cd web && npx tsc --noEmit` — no type errors
  - Visual: confirm no bar above tabs in browser
- stop if:
  - `hasValidationIssues` is used elsewhere in the component (check before removing)
- escalate to:
  - user clarification

### T3 — Compact tab triggers (remove index prefix)
- type: refactor
- inputs:
  - `web/src/components/config/VisualConfigEditor.tsx`
- touches:
  - `Tabs.Trigger` inner markup (lines 346-375)
- avoid:
  - Error badge behavior — keep identical
  - Icon rendering — keep identical
- steps:
  1. Remove the index number span (`String(index + 1).padStart(2, '0')`) from each trigger
  2. Simplify trigger inner layout: icon + label + optional error badge
  3. Adjust padding/gap if needed for the more compact trigger
- expected outputs:
  - Triggers show icon + label + error badge only
  - No `01`, `02` numbering
- verification:
  - `cd web && npx tsc --noEmit` — no type errors
  - Visual: triggers look clean without numbered prefix
- stop if:
  - Index prefix is used for accessibility (check aria attributes)
- escalate to:
  - user clarification

## Risks / Watch-fors
- T1-T3 all touch the same file — execute sequentially within the wave, not parallel edits
- Verify the validation-blocked badge info is still accessible to users after removing the info bar (error badges on triggers serve this role)
