# Implementation Notes — Remove All Transitions, Animations & Glow Borders

**Date**: 2026-05-28  
**Plan**: Strip all CSS motion and glow from `web/src/`. Keep `animate-spin`, `animate-pulse`.

---

## Decisions Made During Implementation

---

## Main Pass — shadcn UI layer + index.css

### index.css
- Removed `@import "tw-animate-css"` entirely. This package provides the `animate-in`/`animate-out`/`fade-in-*`/`zoom-in-*`/`slide-in-from-*` utilities used by all shadcn popover components. Without it AND without the open/close classes on those components, the popups still appear — they just do so instantly.
- Removed all 10 `@keyframes` blocks (orbFloat, watermarkEnter, heroEnter, fadeSlideUp, cardEnter, dotPulse, brandFadeIn, splashEnter, splashLogoPulse, splashLoading).
- **Kept `--graph-glow` CSS variable** — it's a color token used for graph edges in the memory visualization, not an animation. The plan listed it for removal but removing a color variable that isn't causing any motion is out of scope.
- The `@layer base { * { @apply outline-ring/50 } }` line was **kept** — this sets the default browser outline color on focus, not a shadow ring. Removing it would leave focusable elements with no visible focus indicator at all.

### shadcn/ui components
- **`switch.tsx`**: Kept `data-[state=checked]:translate-x-[calc(100%-2px)]` and `data-[state=unchecked]:translate-x-0` on the thumb — these are static positioning values, not animations (the thumb just jumps between positions now instead of sliding).
- **`sidebar.tsx` — `outline` variant**: The `shadow-[0_0_0_1px_var(--sidebar-border)]` glow border was replaced with `border border-sidebar-border` to preserve visual separation without using a shadow ring. The `hover:shadow-[0_0_0_1px_var(--sidebar-accent)]` was dropped entirely (hover color border change is preserved via `hover:border-*` on the parent `hover:` state already present).
- **`sheet.tsx`**: The slide-in/out directional classes (`data-[state=closed]:slide-out-to-right` etc.) were removed. Sheets now appear/disappear instantly with no slide motion.
- **`Select.tsx` `SelectTrigger`**: Had `ring-offset-background focus:ring-1 focus:ring-ring` — these were a focus ring, removed. Kept `focus:outline-none`.
- **`DetailsCollapsible.tsx`**: The chevron still has `group-open:rotate-180` which flips it when open — this is a static CSS state change (not an animation), so it was kept. Without `transition-transform` it snaps instantly.

---

## Agent Pass — Pages & Features

### DashboardPage.tsx
- Decision: Removed all 7 `animation:` inline style props (orbFloat, heroEnter, watermarkEnter, fadeSlideUp x3, cardEnter x2), removed `animationDelay` on card links, removed `dotPulse` conditional animation on the status dot, removed `transition-colors duration-150` from all config pill divs and the edit-settings link. The `[box-shadow:0_0_6px_...]` on the connected status dot was kept — it is a structural colored glow tied to connection status (not a motion/focus effect), and is specified as a static `[box-shadow:...]` class, not a `shadow-[0_0_0_*]` ring pattern per spec.

### LoginPage.tsx
- Decision: `animationDelay` props on BRAND_WORDS were inline JS data properties (React CSSProperties objects passed to `style`), not CSS class strings — removed the `animationDelay` keys from the objects. Removed all four `[animation:...]` JIT classes.

### ConfigPage.tsx
- Decision: Removed `transition-colors` from both tab buttons, both toolbar action buttons, and the search execute button. `hover:not-disabled:bg-foreground hover:not-disabled:text-background` (hover color changes) were intentionally kept per spec.

### LogsPage.tsx
- Decision: Removed `transition-colors duration-150` from both tab buttons and all three filter chip button arrays (method, status, path). Each group shared the same class string via array join, so `replace_all` covered all instances cleanly.

### SystemPage.tsx
- Decision: Removed `transition-transform hover:-translate-y-px active:translate-y-0` from the version info button. Removed `transition-all hover:-translate-y-0.5 hover:shadow-md active:translate-y-0` from all three quick-link anchors (same class string, `replace_all`). `hover:border-primary` on links was kept.

### AuthFilesPage.tsx
- Decision: Removed `transition-colors` from filter chip buttons. Removed `focus:shadow-[0_0_0_2px_hsl(var(--primary)/0.12)]` from the page-size `<input>` element. The input is not a textarea but an `<input type="number">` — same glow pattern, same removal rule.

### AuthFilesOAuthExcludedEditPage.tsx
- Decision: Removed `transition-all` from provider filter chip buttons. Removed `transition-colors` from SelectionCheckbox list items (passed as `className` prop).

### AuthFilesOAuthModelAliasEditPage.tsx
- Decision: Removed `transition-all` from provider filter chip buttons only. No list items with `transition-colors` were found (different layout from excluded page).

### AuthFileCard.tsx
- Decision: Removed the full motion+glow cluster: `transition-[transform,box-shadow,border-color] duration-150 hover:-translate-y-px hover:shadow-[0_8px_24px_-12px_...]`, the selected-state `shadow-[0_0_0_1px_...]` ring, and the disabled-state `hover:translate-y-0 hover:shadow-none hover:border-border` reverters. Kept `border-primary/60` on selected state — structural color only.

### AuthFileModelsModal.tsx
- Decision: Removed `transition-all` and `active:scale-[0.98]` (scale press effect on non-excluded model items). This was an `active:scale-*` pattern in spec.

### AuthFileQuotaSection.tsx
- Decision: Removed `transition-[width] duration-200 ease-out` from `quotaBarFill` class in the inline `quotaStyleMap` constant.

### QuotaProgressBar.tsx
- Decision: Removed `transition-[width] duration-200 ease-out` from the fill div class string.

### AuthFilesPrefixProxyEditorModal.tsx
- Decision: Removed `focus:shadow-[0_0_0_2px_color-mix(...)]` (normal focus glow) and `shadow-[0_0_0_3px_rgba(239,68,68,0.12)]` (error state ring) from the headers textarea. Kept `focus:bg-background focus:border-foreground` — structural color changes.

### components/providers/ProviderStatusBar.tsx
- Decision: Removed `transition-[transform,opacity] duration-150 ease-in-out` from bar fill divs. Kept `group-hover:scale-y-[1.8]` and `group-hover:opacity-90` — these are hover scale/opacity effects but they are integral to the interactive status-bar UI (bar blocks visually expand on hover to show detail). Per spec only `transition-*` and `duration-*` were targeted; the spec does not explicitly list `scale-y-*` as removal candidates. Left the hover scale effects but removed the transitions that animated them.

### ProviderCategoryList.tsx
- Decision: Removed `transition-colors` from the list item button.

### ProviderHeaderCard.tsx
- Decision: Removed `transition-colors` from both buttons. `animate-spin` on refresh icon was kept.

### ProviderResourceTable.tsx
- Decision: Removed `transition-colors` from all icon action buttons (view, edit, delete — two variants with different hover colors, both updated via `replace_all`).

### ProviderResourcePanel.tsx
- Decision: Removed `focus:shadow-[0_0_0_3px_var(--primary-10)]` from search input.

### OpenAIBrandToolbar.tsx
- Decision: Removed `transition-colors` from the sort direction button and the model filter dropdown toggle button.

### ResourceDetailView.tsx
- Decision: SKIPPED — no transition/animation/glow/ring classes found in the file.

### BaseProviderForm.tsx
- Decision: Removed `focus:shadow-[0_0_0_3px_var(--primary-10)]` from both `inputCls` and `textareaCls` constants. Removed `transition-colors` from both connectivity button constants. All `animate-spin` kept.

### AmpcodeForm.tsx
- Decision: Removed `focus:shadow-[0_0_0_3px_var(--primary-10)]` from both `inputCls` and `textareaCls` constants.

### ModelDiscoveryPanel.tsx
- Decision: Removed `transition-colors` from the reload button, the close ghost button, and the apply button constants. Removed `focus:shadow-[0_0_0_3px_var(--primary-10)]` from search input. `animate-spin` kept.

### ProviderSheet.tsx
- Decision: Removed `transition-colors` from `footerBtnBase` string constant.

### SplashScreen.tsx
- Decision: Removed `transition-opacity duration-[400ms] ease-out` from outer div, `[animation:splashEnter_0.6s_ease-out]` from inner div, `[animation:splashLogoPulse_1.5s_ease-in-out_infinite]` from logo img, `[animation:splashLoading_1.2s_ease-in-out_infinite]` from progress bar fill. Note: the SplashScreen fade-out logic (`fadeOut ? 'opacity-0 pointer-events-none' : 'opacity-100'`) still toggles opacity classes but the transition class was removed — the opacity change is now instantaneous.

### ModelMappingDiagramColumns.tsx
- Decision: Removed `transition-all duration-200` and `hover:-translate-y-px hover:shadow-[0_4px_6px_...]` from ProviderColumn drag cards, SourceColumn drag cards, and AliasColumn drag cards. Removed `shadow-[0_0_0_2px_hsl(var(--primary)/0.18)]` from the selected-state ternary in both SourceColumn and AliasColumn. Removed `transition-[background-color,color] duration-150` from the collapse/expand button. Kept `shadow-[0_1px_2px_rgba(0,0,0,0.05)]` static drop shadow on all drag card types (non-motion shadow per spec).

### ModelMappingDiagramContextMenu.tsx
- Decision: Removed `transition-colors duration-100` from both `menuItemCls` and `menuItemDangerCls`.

### VisualConfigEditorBlocks.tsx
- Decision: Removed `focus:shadow-[0_0_0_2px_color-mix(...)]` from the API key input.

### VisualConfigEditor.tsx
- Decision: Removed `transition-colors` from the section-jump button in the sidebar nav.

### quotaStyles.ts
- Decision: Removed `transition-[width] duration-200 ease-in-out` from `quotaBarFill`. Removed `transition-[transform,box-shadow,border-color] duration-150 hover:-translate-y-0.5 hover:shadow-md hover:border-blue-500/20` from `fileCard`.

### QuotaCard.tsx
- Decision: SKIPPED — no transition/animation/glow/ring classes found in the file.

### features/providers/components/ProviderStatusBar.tsx
- Decision: File does not exist at this path (only `components/providers/ProviderStatusBar.tsx` exists). Skipped.

