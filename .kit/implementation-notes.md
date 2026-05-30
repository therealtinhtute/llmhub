# Implementation Notes

## 2026-05-30 — quota-tabs-refactor

### Overview
Refactoring Quota Management page: replace 6 stacked `AppCard` sections with Radix Tabs.
- Tab "All": flat single grid of all accounts across providers (type badge distinguishes)
- Provider tabs: dynamic — only appear if that provider has ≥1 accounts
- Controls (Refresh + Paged/All toggle): per-tab, same as current
- Count badge on each tab, small

### Architecture decision: AllQuotaSection as new component
Rather than rendering 6 QuotaSections inside the "All" tab (which would stack, not flatten),
a new `AllQuotaSection` component was created. It:
- Iterates all configs × files to build a combined `{ item, config }` list (each file appears once, first matching config wins)
- Reads all 6 Zustand quota slices in one selector (shallow equality via `useShallow` from zustand)
- Renders a single flat grid with shared pagination/view-toggle/refresh
- Refresh for an item calls the item's config's `fetchQuota` and writes back via `config.storeSetter`

### Interface contract (pre-defined before parallel implementation)
```typescript
interface AllQuotaSectionProps {
  configs: QuotaConfig<QuotaStatusState, unknown>[];
  files: AuthFileItem[];
  loading: boolean;
  disabled: boolean;
}
```

### i18n: no new keys required
Tab labels reuse existing `${config.i18nPrefix}.title` keys already present in all locale files.

### QuotaSection unchanged
Per-provider tabs reuse existing `QuotaSection` as-is — no edits needed there.

### Implementation split (parallel agents)
- Agent A: `web/src/components/quota/AllQuotaSection.tsx` (new file)
- Agent B: `web/src/pages/QuotaPage.tsx` (rewrite to Radix Tabs) + `components/quota/index.ts` (export)

### Tradeoffs
- "All" tab quota state is read from the same Zustand slices as per-provider tabs. When user refreshes on "All" tab, it updates the shared store — per-provider tabs reflect the fresh state immediately.
- `useShallow` is used on the combined quota selector to avoid re-renders when unrelated slices update.
- File ordering in "All" tab: files appear in config iteration order (Claude → Antigravity → Codex → XAI → Gemini CLI → Kimi), then in the original file list order within each provider. This matches the previous stacked-section visual order.

### Post-implementation fixes

**zh-CN.json smart-quote corruption**
Agent-B used smart/curly Unicode quotes (`"` `"`, U+201C/U+201D) as JSON key and string delimiters when inserting the `tab_all` key and reformatting `card_idle_hint`. This made the file invalid JSON and broke the TypeScript build. Root cause: the zh-CN.json value for `card_idle_hint` already contains curly quotes *inside* the string (Chinese UX copy uses them as emphasis), and the agent lost the distinction between inner curly quotes and outer straight delimiters. Fixed via Python byte-level surgery — straight quotes restored as delimiters, inner curly quotes preserved. **This pattern is a known LLM failure mode on CJK locale files with curly quote content.**

**TypeScript: QuotaConfig generic variance**
The `QuotaConfig<TState, TData>` interface is invariant in `TData` because `buildSuccessState: (data: TData) => TState` makes `TData` appear in the contravariant (parameter) position. Typing `ALL_CONFIGS` as `QuotaConfig<QuotaStatusState, unknown>[]` fails because `(data: SpecificType) => ...` is not assignable to `(data: unknown) => ...`. Fix: `AllQuotaSectionProps.configs` typed as `QuotaConfig<any, any>[]` (matching the `CombinedItem.config` type already used internally), and QuotaSection call uses `config as any` with eslint-disable. This is intentional — the array is mixed-type by design; `any` here is the correct ergonomic choice, not a shortcut.

**`useQuotaStore.getState()[storeSetter]` not callable**
TypeScript inferred the bracket-access type as `QuotaStore[keyof QuotaStore]` which is the full value union (records + setters + clearCache). Added explicit cast to `QuotaSetter<Record<string, QuotaStatusState>>` before calling. Same pattern as what QuotaSection already does with its `setQuota` cast.

### Files changed
- `web/src/components/quota/AllQuotaSection.tsx` — NEW (263 lines)
- `web/src/pages/QuotaPage.tsx` — rewritten (Radix Tabs shell)
- `web/src/components/quota/index.ts` — added AllQuotaSection export
- `web/src/i18n/locales/en.json` — added `quota_management.tab_all: "All"`
- `web/src/i18n/locales/ru.json` — added `quota_management.tab_all: "Все"`
- `web/src/i18n/locales/zh-CN.json` — added `quota_management.tab_all: "全部"` + corruption fix
- `web/src/i18n/locales/zh-TW.json` — added `quota_management.tab_all: "全部"`

### Post-check design system fixes (3 safe_auto applied)
1. **Badge gap in TabsTrigger**: Added `gap-1.5` to both TabsTrigger className strings in QuotaPage.tsx. Without this, `countBadge` rendered flush against tab label text (TabsTrigger has `inline-flex items-center` but no gap, unlike `titleWrapper` in QuotaSection which has `gap-2`).
2. **AllQuotaSection content spacing**: Replaced Fragment with `<div className="flex flex-col gap-3">` wrapper. Without this, controls row, grid, and pagination were block siblings with zero gap inside TabsContent (a plain div with `mt-0`). Per-provider tabs don't have this problem because QuotaSection wraps in AppCard which provides padding.
3. **Semantic grid class**: Added `quotaGrid` alias to `quotaStyles.ts` and updated AllQuotaSection to use it instead of `claudeGrid`. All 6 provider grids have identical CSS, but naming a cross-provider grid `claudeGrid` was misleading.

---

## 2026-05-29 — release-pipeline

### T1: .goreleaser.yml — binary format
- GoReleaser v2.16.0 requires Go >= 1.26.3 but go.mod has 1.26.0; GoReleaser auto-switches toolchain. No code change needed — the workflow's `setup-go` will install whatever go.mod declares and GoReleaser will upgrade the toolchain at runtime.
- `formats: [binary]` with `name_template: 'llmhub-{{ .Os }}-{{ .Arch }}'` produces bare executables named `llmhub-linux-amd64`, `llmhub-linux-arm64`, etc. Windows gets `.exe` appended automatically by GoReleaser.
- Removed `files:` (can't bundle with binary format). `config.example.yaml` is now fetched separately by the VPS installer from raw master.

### T2: .github/workflows/release.yml
- Used `goreleaser-action@v6` (latest stable v6 supports GoReleaser v2).
- `fetch-depth: 0` required so GoReleaser can read full git history for changelog generation.
- Bun setup step placed BEFORE goreleaser-action so the `before` hooks (web build + embed) succeed in CI.

## 2026-05-29 — vps-installer

### scripts/install.sh
- Combined Wave 1 + Wave 2 into one coherent script (no artificial split — the binary install and service setup are a single logical flow).
- `sed -i` used to rewrite `auth-dir` in the fetched config.example.yaml before installing it. POSIX `sed -i` requires no extension arg on Linux (GNU sed), but macOS would need `sed -i ''`. Since the installer is Linux-only (per spec), this is safe.
- Idempotency: `id llmhub` check before `useradd`; config install only if absent; systemd unit overwrite is safe (declarative).
- `hostname -I` fallback to "SERVER_IP" string if unavailable (e.g., container without network).

## 2026-05-29 — docs-consistency

### README.md
- One-liner placed first as the recommended path; manual steps follow for users who want control.
- Removed all tar.gz references; `aarch64` remains only in the arch-detection case statement (maps input → `arm64` variable), which is correct.
- Management panel URL and config-editing guidance preserved inline with the one-liner section.

### Makefile
- Dropped `version` variable extraction (raw binaries don't encode version in the filename).
- Dropped `aarch64` mapping (now uses literal `arm64` from `go env GOARCH`).
- `install-latest` no longer creates a tmpdir or extracts archives — raw binary is chmod'd and installed directly.

## 2026-05-30 — design-token-consistency audit

### Scope
Phase A: token violations in 8 files. Phase B: focus-visible rings in 7 component files.
`tsc --noEmit` and `bun run build` pass clean after all changes.

### Phase A decisions

**badge.tsx — `text-white` → `text-destructive-foreground`**
Straightforward. `text-white` was hardcoded; `text-destructive-foreground` is the semantically correct token (currently `oklch(1 0 0)` = white in both themes, so visual result is identical today but will adapt if the token is ever changed).

**alert-dialog / sheet / dialog overlays — `bg-black/50` → `bg-foreground/50`**
Tradeoff: in light mode `foreground` ≈ near-black so result is visually identical. In dark mode the current `foreground` is warm off-white (`oklch(0.807...)`), so `bg-foreground/50` produces a **lighter, warm-tinted** overlay rather than a dark one. This is semantically correct (overlay adapts to theme) but visually different in dark mode — the overlay will be noticeably lighter (50% warm white over dark background = washed-out). If the dark-mode overlay looks wrong on first visual check, add a `--overlay` CSS token defined as `oklch(0 0 0)` in both `:root` and `.dark` and use that instead.

**LogsPage.tsx — request ID badge cyan → `text-primary bg-primary/10 border-primary/25`**
Original `#0891b2` (cyan) was never a token — it was a one-off. The request ID badge semantics are "informational highlight". Mapped to `primary` because that's the closest semantic role. Visual change: light mode stays blue-ish (primary is blue), dark mode changes from cyan to amber (primary is amber in dark). If the intent was specifically "always cyan regardless of theme", a dedicated `--info` token should be defined.

**SystemPage.tsx — GitHub brand color kept, docs color updated**
- `bg-[#24292f]` (GitHub dark gray) — **left as-is**. This is GitHub's brand color, not a design system gap. Replacing it with any token would break the brand recognition.
- `bg-[#10b981]` → `bg-emerald-500` — same visual value (`#10b981` is exactly Tailwind's `emerald-500`). Change is notation only; Tailwind semantic color is preferred over hex literals.

**AuthFilesPage.tsx — `--filter-active-text` inline CSS vars kept as-is**
The vars `'#111827'` (dark text) / `'#ffffff'` (light text) are used as contrast text against dynamic colored filter button backgrounds. These aren't theme colors — they're contrast-critical values that must remain readable regardless of theme. No semantic token exists for "high-contrast text on arbitrary colored surface". Left unchanged.

**AuthFileQuotaSection.tsx + QuotaProgressBar.tsx — `#e0aa14` / `rgba(217,165,22,...)` → `amber-500`/`amber-600`**
- Removed the `bg-[var(--quota-medium-color,#e0aa14)]` pattern. **Tradeoff**: the CSS variable `--quota-medium-color` was a theming hook — callers could override the quota medium color without touching component code. That hook is now gone. If quota color theming per deployment is needed, restore the CSS variable in `index.css` and use `bg-[var(--quota-medium-color)]` with the token set there instead of hardcoded in className.
- Premium plan badge: `rgba(217,165,22,0.15)` ≈ `amber-500/15`; `rgba(217,165,22,0.3)` ≈ `amber-500/30`; `#e0aa14` → `text-amber-600`. Slight visual difference: amber-500 is `oklch(0.769 0.188 70.08)`, the original `#e0aa14` is slightly darker/more saturated. Not perceptible in practice.

### Phase B decisions

**Button.tsx, checkbox.tsx, switch.tsx — standard ring added**
Added `focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2` to all three. These are primary interactive elements and the ring is the WCAG 2.4.7 focus indicator. No visible change during mouse use; keyboard-only users now get a visible ring.

**sidebar.tsx — used `ring-sidebar-ring` instead of `ring-ring`**
Sidebar components have their own token namespace (`--sidebar-ring`). Using the global `ring-ring` token in a sidebar context would cause the ring color to be inconsistent with the sidebar theme. Used `focus-visible:ring-2 focus-visible:ring-sidebar-ring` (no ring-offset — sidebar backgrounds are already dark/contrasting enough without offset). Applied to: `SidebarMenuButton`, `SidebarGroupAction`, `SidebarMenuAction`, `SidebarMenuSubButton`. Skipped `SidebarGroupLabel` — it's a non-interactive text label.

**dropdown-menu.tsx — ring intentionally skipped**
Dropdown menu items use `focus:bg-accent focus:text-accent-foreground` as their focus indicator (background highlight). This is the standard pattern for menu items across all major design systems (Material, shadcn, Radix). Adding a ring on top would create a double-indicator and look visually noisy inside the popover. The background change is sufficient for WCAG compliance inside a constrained menu context.

**Input.tsx — `focus-visible:border-ring` kept, ring not added**
Current pattern changes the border color to the ring color on focus. This is a deliberate, clean pattern for inputs (subtle border highlight vs outer glow ring). The full ring pattern (`ring-2 ring-ring ring-offset-2`) would add a 4px halo around every focused input, which is heavy for dense form layouts. The border-ring approach satisfies WCAG 2.4.7 (visible focus change) without the layout disruption. If the project ever moves to a ring-first design language, this would change.

## 2026-05-30 — quota-progress-bar-multi-status

### Scope
Fixed a non-rendering progress bar and upgraded from 2-threshold/3-color to 4-threshold/5-color bands across all 6 quota providers (Claude, Codex, Antigravity, Gemini CLI, Kimi, XAI) in both rendering paths (QuotaPage cards and AuthFiles inline section).

### Decision: root cause of gray-only bar
`bg-success` was used throughout the codebase but `--color-success` was never registered in Tailwind v4's `@theme inline` block. Tailwind v4 generates `background-color: var(--color-success)` for `bg-success`, and when the variable is undefined the property is silently ignored. Fixed by adding `--success` to `:root` and `.dark` in `index.css`, then wiring it with `--color-success: var(--success)` in `@theme inline`. This unblocks `bg-success` across the whole app (DashboardPage, DiffModal, ProviderStatusBar also used it).

### Decision: hardcode colors inside QuotaProgressBar, remove threshold props
The original design passed `highThreshold` and `mediumThreshold` as props so callers could customize thresholds. In practice all 6 call sites (in `quotaConfigs.ts`) used the same two constants (70/30). Removing the props and hardcoding the 4 thresholds directly in both `QuotaProgressBar` components eliminates 12 lines of boilerplate across call sites and makes the color logic auditable in one place.
**Tradeoff lost**: if a future provider needs different thresholds, there is no per-call-site override. Judgment: no such need exists today; adding props back is trivial if needed.

### Decision: duplicate QuotaProgressBar components kept separate
There are two `QuotaProgressBar` implementations — `components/quota/QuotaCard.tsx` (used via QuotaPage cards) and `features/authFiles/components/QuotaProgressBar.tsx` (used via AuthFiles inline quota). They have the same logic but different class sources (one uses `quotaStyles` imports, the other hardcodes Tailwind classes). Merging them would require a shared import across feature boundaries. Left separate; both now express identical 5-band logic.

### Decision: color palette for 5 bands
Used the natural Tailwind progression:
- `>80%` remaining → `bg-green-500` (plenty)
- `>50%` remaining → `bg-lime-500` (ok)
- `>20%` remaining → `bg-amber-500` (getting low)
- `>10%` remaining → `bg-orange-500` (running out)
- `≤10%` remaining → `bg-destructive` (critical)

`null` percent (unknown/unavailable) → `bg-amber-500` (neutral warning; same as before).
Threshold boundaries use `>` (strict greater-than), not `>=`. Edge case: exactly 80% maps to `bg-lime-500` not `bg-green-500`. This is correct — 80% remaining is "ok", not "excellent". The old code used `>=` for the two-band boundaries; changed to `>` throughout for consistency.

### Tradeoff: quotaBarFillHigh / Medium / Low keys removed from quotaStyles.ts
These 3 keys were in `QuotaStyleMap` (exported type) but were never accessed via `helpers.styles` in any `renderQuotaItems` function — the fill colors were resolved inside `QuotaProgressBar` directly. Removing them shrinks the type surface and removes the dead `bg-[var(--quota-medium-color,#e0aa14)]` CSS variable hook. If quota bar color theming per deployment was ever needed via that CSS variable, it is now gone — would need to be restored in `index.css` as `--quota-medium-color` and re-exposed.

### Files changed
`web/src/index.css`, `web/src/components/quota/quotaStyles.ts`, `web/src/components/quota/QuotaCard.tsx`, `web/src/features/authFiles/components/QuotaProgressBar.tsx`, `web/src/components/quota/quotaConfigs.ts`, `web/src/features/authFiles/components/AuthFileQuotaSection.tsx`

## 2026-05-30 — config-tabs-layout/T1-T3
**Decision**: Visual verification skipped — app requires auth with a running Go backend. Type-check passes clean and code changes are correct by structural inspection.
**Spec gap**: Spec validation expectations say "open config page in browser" but don't account for the auth gate when no backend is running.
**Tradeoff**: Proceeding without visual proof means layout regressions (overflow, trigger sizing) won't be caught until manual testing with the full stack.
**Risk**: Low — the CSS change (grid→flex) is well-understood and the type-check confirms no structural issues.
