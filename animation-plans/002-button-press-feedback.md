# 002 — Press feedback on buttons

- **Status**: DONE (applied 2026-08-14; mechanical checks pass — feel-check pending in browser)
- **Commit**: bd292f03
- **Severity**: MEDIUM
- **Category**: 3 (Physicality & origin) / 8 (Missed opportunities)
- **Estimated scope**: 1 file, 1 class-string edit

## Problem

Every button in the app (dialogs, toolbars, tables, forms) gives zero tactile
feedback on press — `src/components/ui/Button.tsx` has no `active:` transform and no
transition. In a dashboard with hundreds of click targets, a flat press reads as
dead UI. The page-level hover reveals (`ModelsPage.tsx:393`, `QuotaPage.tsx:189`)
already transition; buttons should too.

## Target

Subtle, crisp press: `transform: scale(0.97)` while `:active`, released with a
160ms ease-out — within the 100–160ms button-feedback budget, no bounce, gated by
`motion-safe` so reduced-motion users get no movement.

```
active → scale(0.97)        (immediate on press)
release → scale(0.97)→1     160ms var(--ease-out)
```

## Repo conventions to follow

- `Button.tsx` composes classes with `cn()` from `@/lib/utils` — add to the existing base class array; do not restructure.
- Easing tokens: `--ease-out` / `--ease-in-out` are defined in `src/index.css` `@theme` by plan 001 — if 001 has landed, the plain `ease-out` utility below automatically uses the strong curve. If 001 has not landed, `ease-out` is Tailwind's built-in (acceptable; plan 001's token override improves it later).
- Personality: crisp dashboard — fast, no bounce.
- Exemplar: `src/pages/AuthFilesPage.tsx:456-472` uses transform+opacity motion at 220–280ms with custom easings; press feedback is faster (100–160ms).

## Steps

1. In `src/components/ui/Button.tsx`, find the `base` class string (the one applied to every variant). Add to it:

   ```
   transition-transform duration-150 ease-out motion-safe:active:scale-[0.97]
   ```

   Resulting base classes (existing content unchanged, additions appended):

   ```tsx
   const base =
     "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-transform duration-150 ease-out motion-safe:active:scale-[0.97] disabled:pointer-events-none disabled:opacity-50 ...";
   ```

   (Keep every pre-existing class verbatim — only append the new segment.)

2. Do not touch variants, sizes, or `asChild` logic.

## Boundaries

- Do NOT edit any other file.
- Do NOT add `hover:` transforms (hover scale on toolbar buttons is not in this plan).
- Do NOT change the class ordering convention of the file.
- If `Button.tsx` structure differs from this description (drift since commit bd292f03), STOP and report.

## Verification

- **Mechanical**: `cd web && bun run type-check && bunx vite build` — both pass.
- **Feel check**: `bun run dev`, open any page with buttons (Dashboard toolbar, a dialog footer):
  - Press and hold: button shrinks to 97% instantly, no lag.
  - Release: returns to 100% in ~160ms with a slight ease-out, no bounce.
  - Spamming clicks: each press scales from current state (CSS transition retargets — never restarts from 1.0).
  - DevTools Rendering panel, emulate `prefers-reduced-motion: reduce`: pressing shows NO scale change (only any existing color feedback).
  - Slow-motion check (Animations panel at 10%): duration ≈160ms, curve `cubic-bezier(0.23,1,0.32,1)` if 001 landed, else Tailwind default ease-out.
- **Done when**: press feedback is visible on every button variant and reduced-motion emulation removes the movement.
