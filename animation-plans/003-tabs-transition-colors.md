# 003 — Tabs: transition-all → transition-colors

- **Status**: DONE (applied 2026-08-14)
- **Commit**: bd292f03
- **Severity**: LOW
- **Category**: 5 (Performance) / 2 (Easing & duration)
- **Estimated scope**: 1 file, 1 class edit

## Problem

`src/components/ui/tabs.tsx:20` uses `transition-all` on the tab trigger:

```tsx
'ring-offset-background transition-all',
```

`transition-all` animates every animatable property (including layout-affecting
ones) even though the trigger only ever changes colors (ring, background, text).
Unintended properties off-GPU is always a finding; the correct, cheaper property
here is `transition-colors`.

## Target

```tsx
'ring-offset-background transition-colors',
```

The active-tab indicator and focus ring continue to transition (they are color /
box-shadow driven), and nothing else can accidentally animate.

## Repo conventions to follow

- Other transitions in the codebase already scope to the animated property: `ModelsPage.tsx:393` uses `transition-opacity`, `QuotaPage.tsx:189` uses `transition-colors` — match that style.

## Steps

1. In `src/components/ui/tabs.tsx:20`, replace `transition-all` with `transition-colors`. No other changes.

## Boundaries

- Do NOT touch the other classes on the trigger (`min-h-[38px]`, `font-[650]`, etc.).
- Do NOT edit any other file.
- If line 20 has drifted, STOP and report.

## Verification

- **Mechanical**: `cd web && bun run type-check && bunx vite build` — both pass.
- **Feel check**: `bun run dev`, open any tabbed page (Config page): switching tabs still transitions the active state color/indicator at the same feel; nothing else animates. DevTools: the trigger's transition property lists only colors, not `all`.
- **Done when**: class replaced and behavior unchanged.
