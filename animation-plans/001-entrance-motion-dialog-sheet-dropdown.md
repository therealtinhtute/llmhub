# 001 — Entrance motion for dialogs, sheets, dropdowns

- **Status**: DONE (applied 2026-08-14; mechanical checks pass — feel-check pending in browser)
- **Commit**: bd292f03
- **Severity**: HIGH
- **Category**: 3 (Physicality & origin) / 8 (Missed opportunities)
- **Estimated scope**: 4 files, ~90 lines of CSS + small class edits

## Problem

The app's most frequent state changes teleport in with zero motion. `dialog.tsx:63`,
`sheet.tsx:60`, and `dropdown-menu.tsx:44` contain no entrance/exit animation classes
at all (no `animate-in`/`data-[state=open]` CSS exists anywhere in `src/` — grep for
`@keyframes` and `animate-in` returns nothing). Every config edit, API-key dialog,
filter sheet, and context menu appears instantly and disappears instantly.

Current code, verbatim:

```tsx
{/* src/components/ui/dialog.tsx:37-46 — overlay, no fade */}
<DialogPrimitive.Overlay
  data-slot="dialog-overlay"
  className={cn(
    "fixed inset-0 z-50 bg-foreground/50",
    className
  )}
  {...props}
/>

{/* src/components/ui/dialog.tsx:62-65 — content, centered translate, no scale/fade */}
className={cn(
  "fixed top-[50%] left-[50%] z-50 grid w-full max-w-[calc(100%-2rem)] translate-x-[-50%] translate-y-[-50%] gap-4 rounded-lg border border-border bg-background p-6 shadow-lg outline-none sm:max-w-lg",
  className
)}

{/* src/components/ui/sheet.tsx:60-66 — content, side-positioned, no slide */}
className={cn(
  "fixed z-50 flex flex-col gap-4 bg-background shadow-lg",
  side === "right" && "inset-y-0 right-0 h-full w-3/4 border-l sm:max-w-sm",
  ...
)}

{/* src/components/ui/dropdown-menu.tsx:44-50 — popper content, no scale/fade */}
className={cn(
  "z-50 max-h-(--radix-dropdown-menu-content-available-height) min-w-[8rem] overflow-x-hidden overflow-y-auto rounded-md border bg-popover p-1 text-popover-foreground shadow-md",
  className
)}
```

`alert-dialog.tsx` has the same overlay/content slots and gains the dialog treatment for free via shared selectors.

## Target

Pure-CSS entrance/exit using Radix's built-in `data-state="open|closed"` attributes
(Radix defers unmount until CSS animations finish, so exit animations work without JS).
Values are from the audit catalog — do not approximate:

| Element | Entry | Exit |
|---|---|---|
| Dialog/AlertDialog overlay | `opacity 0→1`, 200ms `--ease-out` | `opacity 1→0`, 150ms `--ease-out` |
| Dialog/AlertDialog content | `scale(0.95)→1` + `opacity 0→1`, 200ms `--ease-out`, `transform-origin: center` (modal — exempt from trigger-origin rule) | `scale(1)→0.97` + `opacity 1→0`, 150ms `--ease-out` |
| Sheet overlay | `opacity 0→1`, 250ms `--ease-out` | `opacity 1→0`, 150ms `--ease-out` |
| Sheet content | slide in from `data-side` edge, 250ms `--ease-out` | slide out toward `data-side` edge, 200ms `--ease-out` |
| Dropdown content | `scale(0.96)→1` + `opacity 0→1`, 150ms `--ease-out`, `transform-origin: var(--radix-dropdown-menu-content-transform-origin)` (fallback `top left`) | `opacity 1→0`, 100ms `--ease-out` |

Under `prefers-reduced-motion: reduce`: keep the opacity fades, drop ALL transforms
(movement off, feedback on). Override exit animation-duration to 0 so elements unmount
immediately.

## Repo conventions to follow

- No motion tokens exist yet — this plan introduces them in `src/index.css` via Tailwind 4 `@theme`. Overriding `--ease-out` / `--ease-in-out` there makes every existing `ease-out`/`ease-in-out` utility in the codebase inherit the stronger curve (desired cohesion, zero churn elsewhere).
- Motion personality: crisp dashboard — fast, no bounce, no stagger on these elements.
- Exemplar of good motion in this repo: `src/pages/AuthFilesPage.tsx:456-472` (`motion/mini` `animate()` on transform+opacity, 220–280ms, interruptible via `.stop()`). CSS is preferred for these predetermined entrances.

## Steps

1. **`src/index.css` — add motion tokens** inside the existing `@theme inline` block (lines ~113+). Add to the theme:

   ```css
   --ease-out: cubic-bezier(0.23, 1, 0.32, 1);
   --ease-in-out: cubic-bezier(0.77, 0, 0.175, 1);
   ```

2. **`src/index.css` — append keyframes + state selectors** after the theme blocks (top-level, not inside a media query):

   ```css
   /* Entrance/exit motion — Radix mounts these with data-state="open|closed" */
   @keyframes llmhub-fade-in { from { opacity: 0; } to { opacity: 1; } }
   @keyframes llmhub-fade-out { from { opacity: 1; } to { opacity: 0; } }
   @keyframes llmhub-scale-in { from { opacity: 0; transform: scale(0.95); } to { opacity: 1; transform: scale(1); } }
   @keyframes llmhub-scale-out { from { opacity: 1; transform: scale(1); } to { opacity: 0; transform: scale(0.97); } }
   @keyframes llmhub-slide-in-right { from { opacity: 0; transform: translateX(100%); } to { opacity: 1; transform: translateX(0); } }
   @keyframes llmhub-slide-out-right { from { opacity: 1; transform: translateX(0); } to { opacity: 0; transform: translateX(100%); } }
   /* …and the same pairs for -left, -top, -bottom with translateX(-100%) / translateY(-100%) / translateY(100%) */

   [data-slot='dialog-overlay'][data-state='open'],
   [data-slot='alert-dialog-overlay'][data-state='open'] { animation: llmhub-fade-in 200ms var(--ease-out); }
   [data-slot='dialog-overlay'][data-state='closed'],
   [data-slot='alert-dialog-overlay'][data-state='closed'] { animation: llmhub-fade-out 150ms var(--ease-out); }

   [data-slot='dialog-content'][data-state='open'],
   [data-slot='alert-dialog-content'][data-state='open'] { animation: llmhub-scale-in 200ms var(--ease-out); }
   [data-slot='dialog-content'][data-state='closed'],
   [data-slot='alert-dialog-content'][data-state='closed'] { animation: llmhub-scale-out 150ms var(--ease-out); }

   [data-slot='sheet-overlay'][data-state='open'] { animation: llmhub-fade-in 250ms var(--ease-out); }
   [data-slot='sheet-overlay'][data-state='closed'] { animation: llmhub-fade-out 150ms var(--ease-out); }

   [data-slot='sheet-content'][data-side='right'][data-state='open'] { animation: llmhub-slide-in-right 250ms var(--ease-out); }
   [data-slot='sheet-content'][data-side='right'][data-state='closed'] { animation: llmhub-slide-out-right 200ms var(--ease-out); }
   /* …same pattern for left / top / bottom */

   [data-slot='dropdown-menu-content'][data-state='open'] {
     animation: llmhub-scale-in 150ms var(--ease-out);
     transform-origin: var(--radix-dropdown-menu-content-transform-origin, top left);
   }
   [data-slot='dropdown-menu-content'][data-state='closed'] { animation: llmhub-fade-out 100ms var(--ease-out); }

   @media (prefers-reduced-motion: reduce) {
     [data-slot='dialog-content'][data-state='open'],
     [data-slot='alert-dialog-content'][data-state='open'],
     [data-slot='sheet-content'][data-state='open'] {
       animation-name: llmhub-fade-in;  /* opacity only — movement dropped */
     }
     [data-slot='dialog-content'][data-state='closed'],
     [data-slot='alert-dialog-content'][data-state='closed'],
     [data-slot='sheet-content'][data-state='closed'],
     [data-slot='dropdown-menu-content'][data-state='closed'] {
       animation-duration: 0s;  /* unmount immediately */
     }
   }
   ```

   Note: the dropdown uses `llmhub-scale-in` (name mentions scale) but under reduced
   motion `animation-name: llmhub-fade-in` is NOT applied to it — instead give it its
   own fade-in override inside the media query: `[data-slot='dropdown-menu-content'][data-state='open'] { animation-name: llmhub-fade-in; }`. The above is the target pattern; apply consistently.

3. **Do NOT edit `dialog.tsx`, `sheet.tsx`, `dropdown-menu.tsx`, `alert-dialog.tsx` markup** — the existing `data-slot` attributes and Radix `data-state`/`data-side` attributes are sufficient. If a selector doesn't match in DevTools, adjust the selector, not the markup.

## Boundaries

- Do NOT touch `Select.tsx`, `collapsible.tsx`, `sidebar.tsx`, or any page files.
- Do NOT add dependencies (no tailwindcss-animate, no tw-animate-css).
- Do NOT change any duration/easing from the table above.
- Do NOT add JS motion to these components — CSS only.
- If `data-state`/`data-side` attributes differ from the spec in DevTools, STOP and report instead of improvising.

## Verification

- **Mechanical**: `cd web && bun run type-check && bunx vite build` — both must pass. (CSS-only changes; build output must stay a single HTML file.)
- **Feel check**: `bun run dev` (or `make dev-web`), then:
  - Open any dialog (e.g. edit API key): content scales 0.95→1 from center with overlay fade, under ~250ms. Close: brief fade+shrink, no lag before unmount.
  - Open the auth-files batch bar? No — instead: open a sheet (filter/config panel): it slides from its edge with the overlay fading, 250ms.
  - Open any dropdown menu (row actions): it scales from its trigger's corner (not from center) — verify `transform-origin` computed style resolves to the Radix var; if it falls back to `top left`, that's acceptable.
  - DevTools Animations panel at 10% playback: confirm scale starts at 0.95–0.96, opacity from 0, and exit curves are `cubic-bezier(0.23,1,0.32,1)`.
  - DevTools Rendering panel, emulate `prefers-reduced-motion: reduce`: dialogs fade (no scale), sheets fade (no slide), exits unmount instantly.
- **Done when**: all five components animate per the table, reduced-motion gates movement, and type-check + build pass.

## Fallbacks for the executor

- If `--radix-dropdown-menu-content-transform-origin` is not defined by the unified `radix-ui` package at runtime: keep the `top left` fallback in the `var()` call (already in spec).
- If a `data-side` attribute does not exist on sheet content: derive side from the position classes (`inset-y-0 right-0` → right) and add the selector accordingly — but report this in your diff notes.
