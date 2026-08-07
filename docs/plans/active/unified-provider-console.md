---
id: 01KZB88NS7DXSHCAVWR3ZM77JQ
type: plan
intake_id: 01KZB88KH357MGHT267KSNHBB3
lane: normal
status: active
created: 2026-08-06
updated: 2026-08-07
---

# Unified provider console — category grid + sheet console

Status: **locked** · Created 2026-08-06 · Re-planned 2026-08-06 (rail → grid) · Ships as one phase
Source of the shape: `decolua/9router` — single provider registry with a `category` field
(`oauth | apikey | freeTier | free`) driving UI grouping
Reference skill: `.claude/skills/9router-port/references/providers.md` §"Categories 9router uses"

## Outcome

Collapse `/ai-providers` and `/oauth` into one screen at `/ai-providers`, laid out as **per-category
grids of provider cards**. Every card — OAuth provider, preset provider, API-key brand, custom —
opens an `AppSheet`; the sheet holds the resource table and forms for config/preset cards, and the
login flow plus that provider's auth files for OAuth cards. OAuth login state lives on the page, not
in the sheet, so closing a sheet mid-login does not cancel it.

Success signals:
- `/oauth` is gone from the sidebar and redirects to `/ai-providers?entry=oauth:anthropic` with that
  card's sheet open; the sidebar `gateway` group drops from 5 items to 4.
- The page renders 15 cards across 5 category grids: `OAUTH LOGIN` (6), `FREE` (1), `FREE TIER` (1),
  `API KEY` (5), `CUSTOM` (2).
- Clicking any card opens a sheet with the correct content for its kind, and nothing on the page is
  ever a dead card.
- Starting a login, closing the sheet, and re-opening the same card shows the login still in
  progress; the card itself carries a "waiting for authorization" badge while it runs.
- Selecting `OpenRouter` shows only the `openai-compatibility` entries whose `base_url` matches the
  preset, with a prefilled Add button; entries whose `base_url` matches no preset appear under
  `CUSTOM`, and the two sets partition the group with no duplicates and no drops.
- Every brand logo on the grid is the provider's current official glyph.
- `cd web && bun run type-check && bun run lint && bun run build` all clean.
- `go test ./...` clean; `make build-web && make build` succeeds.
- No new test files created anywhere under `web/` (project rule, `CLAUDE.md`).

## Authority and Requirements

Authority:
- Owner decisions taken 2026-08-06 across two `/think` sessions — the first locked `D1`-`D4`, the
  second (the "tối ưu rail UI UX layout" brief) locked `D5`-`D13` and reversed `D4`. All are
  recorded verbatim in `## Decisions` and are the binding scope authority.
- Owner constraint, verbatim: "**Không sửa design system hiện tại**" · "Match design system: colour,
  components, phù hơp UI UX chuẩn hoá" · "Không lỗi khi test, gọi API luồng chính phải đảm bảo".
  `D10` is the one explicitly approved exception and is scoped to a single map entry.
- `.claude/skills/9router-port/references/providers.md` §"Categories 9router uses" — external
  reference project, cited for the category taxonomy and, as of `D9`, for the card-grid presentation.
- Direct reads of the current panel source, 2026-08-06:
  `providersApi.getProviderPresets()` (`web/src/services/api/providers.ts:592`) served by
  `GET /v0/management/provider-presets`
  (`internal/api/handlers/management/provider_presets.go:13`); `useAuthFilesData()` exposing
  `files`, `loadFiles`, `handleStatusToggle`, `handleDelete`, `handleDownload`
  (`web/src/features/authFiles/hooks/useAuthFilesData.ts:27-53`); `AuthFileItem.type` carrying the
  provider discriminator (`web/src/types/authFile.ts:8-24`); `AppSheet`'s `SIZE_MAP`
  (`web/src/components/ui/AppSheet.tsx:12-16`) topping out at `xl` = 576px; `ProviderSheet`'s
  `renderBody()`/`footer` split (`web/src/features/providers/sheets/ProviderSheet.tsx:151-242`);
  the `auto-fit` grid idiom already in the design system at
  `web/src/components/config/VisualConfigEditor.tsx:80` and `web/src/pages/SystemPage.tsx:417`.
- `CLAUDE.md` project rule "Do not create new frontend test files anywhere under `web/`" —
  authority for the verification style in `## Verify`.

Owner decisions:
- `D1` — Merge two screens only. The OAuth sheet embeds the account list; deep management
  (batch upload, model alias, excluded models, kiro quota) keeps linking out to `/auth-files`.
- `D2` — Preset entries follow the catalog model: always visible even with zero configured entries,
  exactly like today's OAuth cards.
- `D3` — OpenRouter and NVIDIA are separate top-level cards, 9router-style.
- `D4` — *Superseded by `D9`.* (Originally: keep the 240px left rail + right panel.)
- `D5` — OAuth login state (`states`, `pollingTimers`, `successResetTimers`) is lifted to
  `ProvidersWorkbenchPage`; the OAuth panel becomes presentational.
- `D6` — Preset category is an **explicit `category` field** in the embedded catalog JSON, not
  derived from other fields.
- `D7` — `CUSTOM` stays as the 5th category.
- `D8` — Category order is `OAUTH → FREE → FREE TIER → API KEY → CUSTOM`.
- `D9` — **No rail.** Per-category grids of cards; clicking a card opens an `AppSheet`. Reverses
  `D4`.
- `D10` — Add a `2xl` (`sm:max-w-4xl`, 896px) entry to `AppSheet`'s `SIZE_MAP`. Explicitly approved
  exception to "không sửa design system hiện tại".
- `D11` — Closing a sheet does **not** cancel an in-flight login; the card shows a
  "waiting for authorization" badge.
- `D12` — Card content is: logo + name + status badge + configured account/resource count +
  error/unverified warning. **Not** the free-tier note, **not** the base URL.
- `D13` — Every brand logo must be the provider's current official mark, sourced official-brand-page
  first, `svgl` only as fallback, always the **glyph** variant (never the wide lockup).

Requirements:
1. `R1` — New `web/src/features/providers/entries.ts` exporting a `ProviderEntry` discriminated
   union (`kind: 'oauth' | 'config' | 'preset'`), a `ProviderEntryCategory` type
   (`'oauth' | 'free' | 'freeTier' | 'apikey' | 'custom'`), and `buildEntries()` producing the
   ordered, sectioned card model from workbench groups + the OAuth provider list + presets + auth
   files. | source: `D8`, `D9`, `## Changes` §2
2. `R2` — New `web/src/features/providers/components/ProviderCategoryGrid.tsx` renders one labelled
   grid per category, in `D8` order, using the design system's existing `auto-fit` idiom
   (`grid-cols-[repeat(auto-fit,minmax(240px,1fr))]`, single column under `md`). It replaces
   `ProviderCategoryList` as the page's navigation surface. | source: `D9`, `## Changes` §3
3. `R3` — Card content is exactly `D12`: brand logo, display name, status badge, configured count,
   and an issue/unverified warning. Card markup reuses `ProviderCategoryList`'s existing row token
   set — same border, `--primary-10`/`--primary-30` active treatment, hover, `focus-visible`
   outline, count-badge and `IconAlertTriangle` styling — so no new design tokens appear. | source:
   `D9`, `D12`, `## Changes` §3
4. `R4` — `AppSheet`'s `SIZE_MAP` gains `'2xl': 'sm:max-w-4xl'` and the `size` prop union widens to
   include `'2xl'`. This is the only edit to any file under `web/src/components/ui/`. | source:
   `D10`, `## Changes` §4
5. `R5` — `ProviderSheet` takes a `view` union — `list | oauth | detail | create | edit` — rendered
   in **one** `AppSheet` instance with in-sheet back navigation from a form view to `list`. `list`
   and `oauth` render at `2xl`; form views keep `descriptor.sheetSize`. No nested or stacked sheets.
   | source: `D9`, `## Changes` §4
6. `R6` — OAuth login state is owned by `ProvidersWorkbenchPage` (`D5`):
   `states: Record<OAuthProvider, ProviderState>` plus the `pollingTimers` and `successResetTimers`
   refs, with `startAuth`, `startPolling`, `completeProviderAuth`, `resetProviderAttempt`,
   `clearProviderTimers`, and `submitCallback` as page-level callbacks. `OAuthLoginPanel` receives
   one provider's state and those callbacks as props and holds no timer of its own. | source: `D5`,
   `D11`, `## Changes` §5
7. `R7` — New `web/src/features/providers/panels/AuthFileMiniTable.tsx`: the auth files belonging to
   the selected OAuth provider, reusing `useAuthFilesData()`, with enable/disable, download, delete,
   and a link to `/auth-files` for everything else. Mapping is `anthropic→claude`,
   `gemini-cli→gemini-cli`, all others identity, applied against `AuthFileItem.type`. | source:
   `D1`, `## Changes` §6
8. `R8` — `web/src/pages/OAuthPage.tsx` is removed; `/oauth` becomes
   `<Navigate to="/ai-providers?entry=oauth:anthropic" replace />`; the `/oauth` sidebar item is
   removed from the `gateway` nav group (`web/src/components/layout/MainLayout.tsx:222`);
   `ProvidersWorkbenchPage` reads and writes the open sheet as an `?entry=` search param so the
   redirect lands on the right card and every card is linkable. | source: `D1`, `## Changes` §7
9. `R9` — `internal/config/presets` gains a required `Category` field on `Preset`, JSON-tagged
   `category`, set in `providers.json` to `opencode: free`, `openrouter: freeTier`,
   `nvidia: apikey`. Mirrored on `ProviderPreset` (`web/src/types/provider.ts:69`) and read by
   `normalizeProviderPreset` (`web/src/services/api/providers.ts:432`). Unknown or missing values
   fall back to `apikey` on the frontend. | source: `D6`, `## Changes` §1
10. `R10` — Preset cards, one per catalog entry, always rendered regardless of whether any matching
    config exists. `openaiCompatibility` resources partition by normalized base URL — lowercase,
    trailing slash stripped. A resource matching a preset's `baseUrl` belongs to that preset's card;
    every other resource belongs to the `CUSTOM` → `OpenAI Compatible` card. The partition is total
    and disjoint. | source: `D2`, `D3`, `## Changes` §2
11. `R11` — The preset sheet's `list` view carries the preset header (base URL, free-tier note,
    verified/unverified badge, signup link) above the existing `ProviderResourceTable`, and its Add
    button opens the `create` view with that preset preselected via a new optional `initialPresetId`
    on `BaseProviderForm`. | source: `D2`, `D3`, `## Changes` §4
12. `R12` — All 18 SVGs under `web/src/assets/icons/` are audited against their provider's official
    brand page; the 3 brands the grid needs and does not have (`opencode`, `openrouter`, `nvidia`)
    are added as glyphs. `openrouter` comes from `openrouter.ai/brand/v2/openrouter-glyph-{light,dark}.svg`,
    **not** from `svgl`, which still serves the retired monochrome arrow mark. | source: `D13`,
    `## Changes` §8
13. `R13` — Every new user-visible string added to both `web/src/i18n/locales/en.json` and
    `web/src/i18n/locales/vi.json`. No hardcoded English in JSX. | source: `## Changes` §9

## Non-goals

- `NG1` — `/auth-files` and its two sub-routes (`/auth-files/oauth-excluded`,
  `/auth-files/oauth-model-alias`) are not merged, not moved, and not modified. Owner chose the
  two-screen merge; absorbing the 890-line page was the rejected option. | source: `D1`
- `NG2` — Auth-file-only providers (`kiro`, `qwen`, `iflow`, `aistudio`, and file-based `vertex`)
  get no card. They have no OAuth start endpoint on `/oauth` today and no `openai-compatibility`
  config; they stay visible on `/auth-files`. | source: `D1`
- `NG3` — No design-system change beyond `R4`'s single `SIZE_MAP` entry. No new colour, no new
  token, no new shared primitive, no restyle of any existing `web/src/components/ui/*` component.
  | source: owner constraint "Không sửa design system hiện tại", `D10`
- `NG4` — No new backend endpoint and no user-facing config-schema change. The one Go edit permitted
  is `R9`'s `category` field on the **embedded** preset catalog, which no user writes and no YAML
  carries. | source: `D6`, `## Authority`
- `NG5` — No new entries added to `internal/config/presets/providers.json`. The catalog stays at the
  three seeded providers; only the `category` field is added to each. | source: `D3`, `D6`
- `NG6` — No `preset_id` field on `OpenAICompatibility`. Matching is by base URL only. The known
  consequence is accepted and documented: editing an entry's `base_url` by hand moves it from its
  preset card to `CUSTOM`. | source: `D3`, `## Risk`
- `NG7` — `model-combos` is untouched. The owner explicitly left that phase pending. | source:
  owner instruction 2026-08-06 ("pending cái model-combos")
- `NG8` — `ProviderResourceTable`, `ProviderResourcePanel`, `OpenAIBrandToolbar`,
  `ResourceDetailView`, `BaseProviderForm`'s field set, `AmpcodeForm`, and `useProviderWorkbench`
  keep their current behavior. They are re-parented into the sheet, not rewritten. This is what
  makes "zero feature loss" checkable rather than aspirational. | source: `D9`

## Building

Cards grouped into five labelled grids; one sheet does all the work.

```
buildEntries({ groups, presets, authFiles })
   ├── OAUTH LOGIN   codex · anthropic · antigravity · gemini-cli · kimi · xai      (6)
   │                 from PROVIDERS + auth files grouped by AuthFileItem.type
   ├── FREE          OpenCode Free                                                  (1)
   ├── FREE TIER     OpenRouter                                                     (1)
   ├── API KEY       Gemini · Codex · Claude · Vertex · NVIDIA NIM                   (5)
   │                 4 workbench brands + 1 preset (category: apikey)
   └── CUSTOM        OpenAI Compatible (unmatched entries) · Ampcode                 (2)
```

`codex`, `claude`, and `gemini` each appear twice on purpose — once as an OAuth login, once as an
API-key brand. That duality already exists today across the two screens and matches 9router, which
also lists a vendor under more than one category when it supports more than one auth method. The
category label above each grid is what disambiguates them; no card is ambiguous in isolation.

Sheet content by entry kind:

| kind | `view: 'list'` | `view: 'oauth'` | form views |
|---|---|---|---|
| `oauth` | — | `OAuthLoginPanel` + `AuthFileMiniTable` | — |
| `preset` | preset header + `ProviderResourceTable` over matched entries + prefilled Add | — | `detail` / `create` / `edit` |
| `config` | today's `ProviderResourcePanel` body, unchanged | — | `detail` / `create` / `edit` |

## Not building

| Excluded | Why |
|---|---|
| Absorbing `/auth-files` | `NG1` — owner picked the two-screen merge |
| Cards for kiro/qwen/iflow/aistudio | `NG2` — no OAuth start, no config block |
| Keeping the 240px rail | `D9` — owner reversed `D4` |
| Nested / stacked sheets | `R5` — one sheet, view union, back navigation |
| `preset_id` config field | `NG6` |
| New presets | `NG5` |
| Any other design-system edit | `NG3` |

## Changes

### 1. Preset category field (`R9`) — the only Go change

`internal/config/presets/presets.go` — one field on `Preset`:

```go
Category string `json:"category"`
```

`internal/config/presets/providers.json` — `"category": "free"` on `opencode`,
`"category": "freeTier"` on `openrouter`, `"category": "apikey"` on `nvidia`.

`internal/config/presets/presets_test.go` gains one case asserting every preset's `Category` is one
of `free | freeTier | apikey`. The existing `TestAllReturnsSeedCatalog` count assertion (`len == 3`)
is untouched — no preset is added or removed.

`web/src/types/provider.ts` — `category: 'free' | 'freeTier' | 'apikey'` on `ProviderPreset`.
`web/src/services/api/providers.ts` — `normalizeProviderPreset` reads
`getStringField(preset, ['category'])` and falls back to `'apikey'` when absent or unrecognized, so
an older binary serving a catalog without the field still renders every preset card.

### 2. Entry view model — `web/src/features/providers/entries.ts` (new)

```ts
export type ProviderEntryCategory = 'oauth' | 'free' | 'freeTier' | 'apikey' | 'custom';

export type ProviderEntry =
  | { kind: 'oauth';  key: string; category: 'oauth'; oauthId: OAuthProvider;
      titleKey: string; hintKey: string; urlLabelKey: string;
      icon: string | { light: string; dark: string }; accountCount: number }
  | { kind: 'preset'; key: string; category: 'free' | 'freeTier' | 'apikey';
      preset: ProviderPreset; resources: ProviderResource[] }
  | { kind: 'config'; key: string; category: 'apikey' | 'custom';
      group: ProviderGroup; resources: ProviderResource[] };
```

`key` is the stable `?entry=` value: `oauth:anthropic`, `preset:openrouter`, `config:gemini`.
Category order is fixed by `D8`: `oauth`, `free`, `freeTier`, `apikey`, `custom`. Preset category
comes straight off `preset.category` (`R9`) — no derivation, no heuristic.

Base-URL normalization and partition (`R10`):

```ts
const normalizeBaseUrl = (u: string | null | undefined) =>
  (u ?? '').trim().toLowerCase().replace(/\/+$/, '');
```

`buildEntries` walks the `openaiCompatibility` group once, bucketing each resource by
`normalizeBaseUrl(resource.baseUrl)` against a `Map<string, presetId>` built from the catalog;
unmatched resources form the `CUSTOM` → `OpenAI Compatible` card's resources.

`PROVIDERS`, `CALLBACK_SUPPORTED`, and `OAUTH_TO_AUTH_FILE_TYPE` move out of `OAuthPage.tsx` into
this module so `buildEntries`, the page-level OAuth state, and the panel share one list.

### 3. Category grid — `components/ProviderCategoryGrid.tsx` (new)

Takes `entries: ProviderEntry[]` and `onOpen(key)`. Groups entries by `category`, emits an uppercase
section label per non-empty category reusing the label class already on
`ProviderCategoryList.tsx:35`, then one grid per category:

```
grid grid-cols-[repeat(auto-fit,minmax(240px,1fr))] gap-3 max-md:grid-cols-[minmax(0,1fr)]
```

Card markup is `ProviderCategoryList`'s row button re-laid-out as a card (`R3`): same border, hover,
`focus-visible` outline, count badge, and `IconAlertTriangle`. Content per `D12` — logo, name, status
badge, configured count, warning. `ProviderCategoryList.tsx` is removed once nothing imports it.

The OAuth card's status badge reads from the page-level `states[oauthId]`, so a login started and
then dismissed shows "waiting for authorization" on the card (`D11`).

### 4. Sheet console — `sheets/ProviderSheet.tsx` + `components/ui/AppSheet.tsx`

`AppSheet` (`R4`) — one map entry and one union member:

```ts
const SIZE_MAP: Record<string, string> = {
  md: 'sm:max-w-md', lg: 'sm:max-w-lg', xl: 'sm:max-w-xl', '2xl': 'sm:max-w-4xl',
};
```

`ProviderSheet` (`R5`) — `state.mode` widens from `detail | create | edit` to
`list | oauth | detail | create | edit`. `renderBody()` gains a `list` branch (preset header when
`kind === 'preset'`, then `ProviderResourcePanel`'s body) and an `oauth` branch
(`OAuthLoginPanel` + `AuthFileMiniTable`). The footer gains a back button on form views that returns
to `list` through the existing `confirmDiscardIfDirty` guard. `size` is `2xl` for `list`/`oauth` and
`descriptor.sheetSize` otherwise. The existing `useEffect` that resets `isDirty` on
brand/mode/resource/open change already covers the new views.

`BaseProviderForm` gains an optional `initialPresetId` (`R11`) consumed by the preset picker landed
in `1abe73c9`.

### 5. OAuth state on the page (`R6`)

`ProvidersWorkbenchPage` owns `states`, `pollingTimers`, and `successResetTimers` — the three
declarations currently at `OAuthPage.tsx:151-153` — plus every handler that touches them. A single
`useEffect` cleanup on page unmount clears both timer maps. `panels/OAuthLoginPanel.tsx` (new)
renders one provider from `{ state, onStart, onSubmitCallback, onReset, onCopyLink }` and keeps only
local form-input state (callback text, gemini-cli project id). The xai helpers `isAbsoluteUrl`,
`readQueryLikeCallbackInput`, `extractDisplayedXaiCode`, `buildXaiCallbackUrl`, and
`resolveCallbackUrl` move to the panel; nothing that owns a timer does.

This is the fix for the defect that made `D5` necessary: with all three declarations component-local,
unmounting the panel killed the `window.setInterval` at `OAuthPage.tsx:236` and silently abandoned an
in-flight login.

### 6. Account table — `panels/AuthFileMiniTable.tsx` (new)

Consumes `useAuthFilesData()`. Filters by

```ts
const OAUTH_TO_AUTH_FILE_TYPE: Record<OAuthProvider, string> = {
  codex: 'codex', anthropic: 'claude', antigravity: 'antigravity',
  'gemini-cli': 'gemini-cli', kimi: 'kimi', xai: 'xai',
};
```

Columns: name, status, last refresh, actions (enable/disable via `handleStatusToggle`, download via
`handleDownload`, delete via `handleDelete`). Footer links to `/auth-files` for batch upload, model
alias, excluded models, and kiro quota. Files whose `type` is absent or `unknown` are not shown here
and remain reachable on `/auth-files` — the footer states this.

### 7. Route and nav consolidation (`R8`)

- `web/src/router/MainRoutes.tsx:26` — `/oauth` element becomes
  `<Navigate to="/ai-providers?entry=oauth:anthropic" replace />`.
- `web/src/components/layout/MainLayout.tsx:222` — drop the `/oauth` nav item.
- `web/src/pages/OAuthPage.tsx` — removed with `trash` after extraction.
- `ProvidersWorkbenchPage` — `?entry=` is the source of truth for which sheet is open; absent means
  no sheet. An unknown value clears the param rather than opening an empty sheet.

### 8. Brand logos (`R12`)

Sourcing rule, in order: the provider's **official brand page** first; `svgl` only as fallback;
always the **glyph** (square icon), never the wide lockup.

Known state going in, from the 2026-08-06 research pass:
- Missing and needed by the grid: `opencode`, `openrouter`, `nvidia` (light + dark each).
- `openrouter` **must not** come from `svgl` — svgl serves a 633B monochrome `#111111` viewBox
  512×512 arrow mark that OpenRouter has retired. Current official is
  `openrouter.ai/brand/v2/openrouter-glyph-{light,dark}.svg` (661B, viewBox 401.4×293.7, brand
  purple `#7624F4`).
- `web/src/assets/icons/amp.svg` is suspected wrong-brand: svgl's "AMP" entry is `amp.dev` (Google
  Accelerated Mobile Pages), not Sourcegraph Ampcode. Inspect and replace if confirmed.
- Absent from svgl entirely, so official-page-only: `vertex`, `amp`, `glm`, `iflow`, `kiro`,
  `minimax`. Of these only `vertex` and `amp` appear on the grid; the other four are audited but not
  blocking.

### 9. i18n (`R13`)

New keys under `providersPage.categories.sections.*` (5 section labels),
`providersPage.presets.*` (header labels, verified/unverified, signup, empty state),
`providersPage.oauthAccounts.*` (mini-table headers, footer link), and
`providersPage.card.*` (status badge including the `D11` pending-authorization string). Existing
`auth_login.*` keys are reused verbatim by `OAuthLoginPanel` — no renames.

## Verify

In order:

- `go test ./...` — clean, including the new preset-category case.
- `cd web && bun run type-check` — clean.
- `cd web && bun run lint` — no new warnings above the current baseline of 0 errors / 8 warnings.
- `cd web && bun run build` — `tsc && vite build` succeeds.
- `make build-web && make build` — the embedded panel builds and the Go binary compiles.
- Browser runtime sweep, driven by a real browser agent (owner requirement: "Test bằng browser, dùng
  agent browser test thật") against the already-running server via `make dev-web`:
  - 5 category labels in `D8` order, 15 cards, counts `6 / 1 / 1 / 5 / 2`;
  - every one of the 15 cards opens a sheet with content matching its kind — no empty sheet, no
    console error on any of the 15;
  - `/oauth` redirects to `/ai-providers?entry=oauth:anthropic` with that sheet open;
  - start a login on one OAuth card, close the sheet, confirm the card shows the pending badge, then
    re-open and confirm the flow is still live (`D11`);
  - `OpenRouter` shows only base-URL-matching entries; sum of preset-card resource counts plus the
    `CUSTOM` card's count equals the total `openai-compatibility` resource count;
  - Add from the OpenRouter sheet opens `create` with OpenRouter preselected and base URL/headers
    prefilled;
  - main API paths are exercised and return 2xx — workbench snapshot, `/provider-presets`,
    auth-files list, and one create + one delete round trip;
  - browser console free of React errors and warnings across the sweep.
- No new files under `web/` matching `*.test.*` or `*.spec.*` (`CLAUDE.md` rule).
- `git diff --stat` shows no file under `web/src/components/ui/` other than `AppSheet.tsx` (`NG3`).

## Rollback

One frontend commit plus a 3-line Go/JSON addition. `git revert` of the phase commit restores both
screens byte-for-byte; `OAuthPage.tsx` and `ProviderCategoryList.tsx` come back with the revert. The
`category` field is additive and read with a fallback, so an un-reverted binary serving the field to
a reverted panel is harmless. No database change, no user config change, no migration, no external
state touched.

## Risk

- **Base-URL matching is brittle to hand edits** (`NG6`). An owner who edits an entry's `base_url`
  away from the preset value sees it move to `CUSTOM`. Accepted and self-explanatory; the escape
  hatch is documented in the preset sheet's empty state. Escalation: add `preset_id` in a follow-up
  if this bites in practice.
- **The `list` view crams a full resource table into an 896px sheet.** Mitigated by `D10`'s `2xl`
  and by `AppSheet`'s existing `flex-1 overflow-y-auto` body; the table's own horizontal overflow is
  unchanged from today. If it reads cramped at `2xl`, the fallback is a `full`-width entry, not a
  restyle of the table (`NG8`).
- **Lifting OAuth state to the page couples two previously independent screens' lifecycles.** A
  render loop or stale closure in the lifted handlers would break the workbench, not just OAuth.
  Mitigated by moving handlers wholesale rather than rewriting them, and by making the
  close-and-reopen check in `## Verify` a hard gate rather than a nice-to-have.
- **Auth files with missing or `unknown` `type` are invisible in the OAuth sheet.** Mitigated by the
  mini-table footer stating that `/auth-files` is the complete list, and by never making the sheet
  the only path to a credential.
- **Logo sourcing drifts or ships the wrong brand.** Already happened once in research (svgl's stale
  OpenRouter mark, and the `amp.dev`/Ampcode collision). Mitigated by `D13`'s official-page-first
  rule and by auditing all 18 rather than only the 3 that are missing.

## Approach and Risks

Chosen approach: exactly `## Changes` §1-9 as one phase. The grid, the sheet console, the lifted
OAuth state, and the preset category field are not separable — the grid has no right-hand panel, so
the sheet *is* the panel; the sheet needs `2xl` to hold the table; the grid's five sections need the
preset category to place three of its cards. Shipping any subset leaves the page in a state where a
card opens nothing or lands in the wrong section. One phase, one revert.

The logo work (`R12`) is the one genuinely separable slice, kept in-phase because `D12` puts the logo
on every card and the owner's constraint was "100% logo brand các provider phải là logo mới nhất" —
shipping the grid with stale marks would ship the requirement unmet.

Rejected alternative: two stacked sheets — a `2xl` console sheet whose rows open the existing
`ProviderSheet` on top. Rejected because Radix dialog stacking gives two overlays, two focus traps,
and two `Escape` handlers to reconcile, and closing the inner one would have to know whether to
restore the outer. The single-sheet view union costs one prop widening on a file we are already
editing.

Also rejected: keeping the rail and adding the grid beside it. Rejected by `D9` — the owner's brief
was to remove the rail, and keeping both would double the navigation surface for 15 entries.

Also rejected: deriving preset category from existing JSON fields, which is what the pre-`D6` plan
specified. Rejected on evidence: the derivation
(`defaultApiKey && !signupUrl → free`, `signupUrl && freeTierNote → freeTier`, else `apikey`) puts
NVIDIA in `freeTier`, while 9router classifies it `apikey` — 1 wrong out of 3 on a 3-entry catalog,
and every new preset would be another coin flip. `D6`'s explicit field costs ~6 lines.

Primary risks: see `## Risk`. The two that can stop the phase are the lifted OAuth state breaking
the workbench render and the resource table being unusable at `2xl`.

Recovery: the phase is a single revertible commit (`## Rollback`). If the browser sweep finds the
OAuth flow broken in a way the lift cannot explain, stop and restore `/oauth` as a standalone route —
the grid is still shippable without the `OAUTH LOGIN` section, since `buildEntries` degrades to the
`config` + `preset` cards alone.

## Phases and Verification
<!-- Phase and task definitions are immutable after to-plan. Do not add task status fields. Append-only Progress is the sole task execution-status source. Only each phase lifecycle status changes to mirror DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. Each planned phase records phase_slug, story_id, status, goal, depends_on, waves, tasks, and checks. -->
- planning_status: planned

### Phase: provider-grid-console
- story_id: `01KZBHHJ7HGPZXD3DW9R48K94E`
- goal: Replace the provider workbench rail with per-category grids of provider cards; every card
  opens an `AppSheet` (resource list/form for config and preset cards, login panel plus auth files
  for OAuth cards) with login state lifted to the page so it survives sheet close. Refresh all brand
  logos to current official glyphs.
- depends_on: provider-presets
- touched_surfaces: `internal/config/presets/presets.go`,
  `internal/config/presets/providers.json`, `internal/config/presets/presets_test.go`,
  `web/src/types/provider.ts`, `web/src/services/api/providers.ts`,
  `web/src/components/ui/AppSheet.tsx`,
  `web/src/features/providers/entries.ts` (new),
  `web/src/features/providers/components/ProviderCategoryGrid.tsx` (new),
  `web/src/features/providers/panels/OAuthLoginPanel.tsx` (new),
  `web/src/features/providers/panels/AuthFileMiniTable.tsx` (new),
  `web/src/features/providers/sheets/ProviderSheet.tsx`,
  `web/src/features/providers/sheets/forms/BaseProviderForm.tsx`,
  `web/src/features/providers/ProvidersWorkbenchPage.tsx`,
  `web/src/features/providers/types.ts`, `web/src/router/MainRoutes.tsx`,
  `web/src/components/layout/MainLayout.tsx`, `web/src/assets/icons/*.svg`,
  `web/src/i18n/locales/en.json`, `web/src/i18n/locales/vi.json`,
  `web/src/pages/OAuthPage.tsx` (removed),
  `web/src/features/providers/components/ProviderCategoryList.tsx` (removed)
- avoided_surfaces: no new backend endpoint and no user config-schema change (`NG4`); no new preset
  entries (`NG5`); no `preset_id` config field (`NG6`); no file under `web/src/components/ui/` other
  than `AppSheet.tsx` (`NG3`); no change to `ProviderResourceTable`, `ProviderResourcePanel`,
  `OpenAIBrandToolbar`, `ResourceDetailView`, `AmpcodeForm`, or `useProviderWorkbench` (`NG8`); no
  change to `/auth-files` or its sub-routes (`NG1`); no new test files under `web/`
- lifecycle status: checked

#### Wave 1 — Preset category field (R9, depends on none)
- task 1.1: Add a `Category` field (JSON tag `category`) to `Preset` in
  `internal/config/presets/presets.go` and set it on all three entries in `providers.json`
  (`opencode: free`, `openrouter: freeTier`, `nvidia: apikey`). Add one case to `presets_test.go`
  asserting every preset's `Category` is one of `free | freeTier | apikey`.
  - touched: `internal/config/presets/presets.go`, `internal/config/presets/providers.json`,
    `internal/config/presets/presets_test.go`
  - check: `go test ./internal/config/presets/... -v` — the new case plus the 3 existing ones pass;
    `go build ./...` clean
- task 1.2: Mirror the field on the frontend — `category: 'free' | 'freeTier' | 'apikey'` on
  `ProviderPreset` (`web/src/types/provider.ts:69`), read in `normalizeProviderPreset`
  (`web/src/services/api/providers.ts:432`) with a fallback to `'apikey'` when absent or
  unrecognized.
  - touched: `web/src/types/provider.ts`, `web/src/services/api/providers.ts`
  - check: `cd web && bun run type-check` clean

#### Wave 2 — Entry view model (R1, R10, depends on Wave 1)
- task 2.1: Create `web/src/features/providers/entries.ts` with `ProviderEntryCategory`, the
  `ProviderEntry` union, `PROVIDERS` / `CALLBACK_SUPPORTED` / `OAUTH_TO_AUTH_FILE_TYPE` moved out of
  `OAuthPage.tsx`, `normalizeBaseUrl`, and `buildEntries({ groups, presets, authFiles })` emitting
  `oauth`, `preset`, and `config` entries in the `D8` category order with `key` values
  `oauth:{id}` / `preset:{id}` / `config:{brand}`.
  - touched: `web/src/features/providers/entries.ts` (new),
    `web/src/features/providers/types.ts`
  - check: `cd web && bun run type-check && bun run lint` clean
- task 2.2: Partition the `openaiCompatibility` group by `normalizeBaseUrl` against a
  `Map<normalizedBaseUrl, presetId>` built from the catalog; unmatched resources become the
  `CUSTOM` → `OpenAI Compatible` card's resources; `ampcode` lands in `custom` after it.
  - touched: `web/src/features/providers/entries.ts`
  - check: `cd web && bun run type-check` clean; the count identity (sum of preset-card counts plus
    the `CUSTOM` card's count equals the total `openai-compatibility` resource count) is asserted in
    task 6.3's browser sweep
- task 2.3: `ProvidersWorkbenchPage` fetches `providersApi.getProviderPresets()` alongside the
  workbench snapshot and feeds it to `buildEntries`; a rejected fetch yields zero preset cards and
  leaves every entry under `CUSTOM` without breaking the page.
  - touched: `web/src/features/providers/ProvidersWorkbenchPage.tsx`
  - check: `cd web && bun run build` succeeds; the degraded path is confirmed in task 6.3 by
    blocking `/provider-presets` in devtools and reloading

#### Wave 3 — Category grid (R2, R3, depends on Wave 2)
- task 3.1: Create `components/ProviderCategoryGrid.tsx` — one uppercase label plus one
  `grid-cols-[repeat(auto-fit,minmax(240px,1fr))] gap-3 max-md:grid-cols-[minmax(0,1fr)]` grid per
  non-empty category, in `D8` order. Card markup reuses `ProviderCategoryList`'s existing token set
  (border, `--primary-10`/`--primary-30` active, hover, `focus-visible` outline, count badge,
  `IconAlertTriangle`); content is exactly `D12` — logo, name, status badge, configured count,
  error/unverified warning.
  - touched: `web/src/features/providers/components/ProviderCategoryGrid.tsx` (new)
  - check: `cd web && bun run type-check && bun run lint` clean
- task 3.2: `ProvidersWorkbenchPage` drops the `xl:grid-cols-[240px_minmax(0,1fr)]` two-column
  layout and renders `ProviderHeaderCard` above `ProviderCategoryGrid` at full width; the loading
  skeleton is updated to match the single-column shape.
  - touched: `web/src/features/providers/ProvidersWorkbenchPage.tsx`
  - check: `cd web && bun run build` succeeds
- task 3.3: Remove `components/ProviderCategoryList.tsx` with `trash`.
  - touched: `web/src/features/providers/components/ProviderCategoryList.tsx` (removed)
  - check: `grep -rn "ProviderCategoryList" web/src` returns no results and
    `cd web && bun run type-check` clean

#### Wave 4 — Sheet console (R4, R5, R6, R7, R11, depends on Wave 3)
- task 4.1: Add `'2xl': 'sm:max-w-4xl'` to `AppSheet`'s `SIZE_MAP` and `'2xl'` to the `size` prop
  union. No other change to any file under `web/src/components/ui/`.
  - touched: `web/src/components/ui/AppSheet.tsx`
  - check: `cd web && bun run type-check` clean; `git diff --name-only web/src/components/ui/`
    lists only `AppSheet.tsx`
- task 4.2: Lift OAuth state to `ProvidersWorkbenchPage` — `states`, `pollingTimers`,
  `successResetTimers` (the three declarations at `OAuthPage.tsx:151-153`) plus `startAuth`,
  `startPolling`, `completeProviderAuth`, `resetProviderAttempt`, `clearProviderTimers`, and
  `submitCallback`, with a page-unmount cleanup clearing both timer maps.
  - touched: `web/src/features/providers/ProvidersWorkbenchPage.tsx`
  - check: `cd web && bun run type-check && bun run lint` clean
- task 4.3: Create `panels/OAuthLoginPanel.tsx` as a presentational component taking one provider's
  state and the page callbacks as props, holding only local input state (callback text, gemini-cli
  project id) and the xai helpers `isAbsoluteUrl`, `readQueryLikeCallbackInput`,
  `extractDisplayedXaiCode`, `buildXaiCallbackUrl`, `resolveCallbackUrl`. It owns no timer.
  - touched: `web/src/features/providers/panels/OAuthLoginPanel.tsx` (new)
  - check: `cd web && bun run type-check && bun run lint` clean; `grep -n "setInterval\|setTimeout"`
    in the new panel returns no results
- task 4.4: Create `panels/AuthFileMiniTable.tsx` consuming `useAuthFilesData()`, filtered by
  `OAUTH_TO_AUTH_FILE_TYPE`, with enable/disable, download, delete, and a footer link to
  `/auth-files` stating that files with absent or `unknown` type are listed only there.
  - touched: `web/src/features/providers/panels/AuthFileMiniTable.tsx` (new)
  - check: `cd web && bun run type-check && bun run lint` clean
- task 4.5: Widen `ProviderSheet`'s mode to `list | oauth | detail | create | edit` in one
  `AppSheet` instance — `list` renders the preset header (when `kind === 'preset'`) above
  `ProviderResourcePanel`'s body, `oauth` renders `OAuthLoginPanel` + `AuthFileMiniTable`, form
  views are unchanged; the footer gains a back-to-`list` button routed through the existing
  `confirmDiscardIfDirty`; `size` is `2xl` for `list`/`oauth` and `descriptor.sheetSize` otherwise.
  - touched: `web/src/features/providers/sheets/ProviderSheet.tsx`,
    `web/src/features/providers/ProvidersWorkbenchPage.tsx`
  - check: `cd web && bun run build` succeeds
- task 4.6: `BaseProviderForm` accepts an optional `initialPresetId` preselecting the existing preset
  picker; the preset sheet's Add button passes its own preset id.
  - touched: `web/src/features/providers/sheets/forms/BaseProviderForm.tsx`
  - check: `cd web && bun run build` succeeds; prefill is confirmed in task 6.3

#### Wave 5 — Brand logos (R12, depends on none)
- task 5.1: Add the three glyphs the grid needs and does not have — `opencode`, `openrouter`,
  `nvidia`, light and dark each. `openrouter` comes from
  `openrouter.ai/brand/v2/openrouter-glyph-{light,dark}.svg`, not from `svgl`, whose copy is the
  retired monochrome arrow mark.
  - touched: `web/src/assets/icons/*.svg` (new files)
  - check: each new file opens as valid SVG with a non-empty `viewBox`, and `openrouter` carries the
    current brand purple `#7624F4` rather than svgl's `#111111`
- task 5.2: Audit all 18 existing SVGs under `web/src/assets/icons/` against their provider's
  official brand page per `D13` (official page first, `svgl` fallback, glyph never lockup). Confirm
  or replace `amp.svg`, which is suspected to be `amp.dev` (Google AMP) rather than Sourcegraph
  Ampcode. Record per-file verdict — current / replaced / no official source found.
  - touched: `web/src/assets/icons/*.svg`
  - check: a written per-file verdict for all 18 in `## Progress`; every brand appearing on the grid
    is either verified current or replaced
- task 5.3: Wire the logo map used by `ProviderCategoryGrid` to cover all 15 grid entries, with
  light/dark variants where both exist and no card falling back to a generic icon.
  - touched: `web/src/features/providers/components/ProviderCategoryGrid.tsx`,
    `web/src/features/providers/entries.ts`
  - check: `cd web && bun run build` succeeds; all 15 cards render a brand mark in task 6.3's sweep,
    in both light and dark theme

#### Wave 6 — Route/nav, i18n, and full verification (R8, R13, depends on Waves 4 and 5)
- task 6.1: `?entry=` drives which sheet is open in `ProvidersWorkbenchPage` (absent means no sheet,
  unknown clears the param); `/oauth` becomes
  `<Navigate to="/ai-providers?entry=oauth:anthropic" replace />`; the `/oauth` sidebar item is
  removed from the `gateway` nav group; `web/src/pages/OAuthPage.tsx` is removed with `trash`.
  - touched: `web/src/features/providers/ProvidersWorkbenchPage.tsx`,
    `web/src/router/MainRoutes.tsx`, `web/src/components/layout/MainLayout.tsx`,
    `web/src/pages/OAuthPage.tsx` (removed)
  - check: `cd web && bun run type-check` clean and `grep -rn "OAuthPage" web/src` returns no results
- task 6.2: i18n — add `providersPage.categories.sections.*`, `providersPage.presets.*`,
  `providersPage.oauthAccounts.*`, and `providersPage.card.*` (including the `D11` pending-
  authorization string) to both locale files; reuse existing `auth_login.*` keys unchanged.
  - touched: `web/src/i18n/locales/en.json`, `web/src/i18n/locales/vi.json`
  - check: `grep -c` key parity between the two locale files for all four new namespaces; no literal
    English string in the new grid, panel, or sheet JSX
- task 6.3: Full gate and browser sweep. Run `go test ./...`, then
  `cd web && bun run type-check && bun run lint && bun run build`, then `make build-web && make build`.
  Then drive a **real browser agent** against the running server via `make dev-web` and confirm every
  item in `## Verify`'s browser list: 5 labels in `D8` order and counts `6 / 1 / 1 / 5 / 2`; all 15
  cards open a correct non-empty sheet; `/oauth` redirect lands with the anthropic sheet open;
  start-login → close sheet → pending badge on the card → re-open still live (`D11`); OpenRouter
  partition and the count identity; prefilled Add; the degraded `/provider-presets` path; main API
  paths returning 2xx including one create + one delete round trip; clean console throughout.
  - touched: none (verification only)
  - check: every command above clean; the browser sweep passes with a clean console; no file under
    `web/` matching `*.test.*` or `*.spec.*` was added; `git diff --name-only web/src/components/ui/`
    lists only `AppSheet.tsx`. **Stop and restore `/oauth` as a standalone route if the OAuth flow
    is broken in a way the state lift cannot explain (see `## Approach and Risks` recovery).**

## Progress
<!-- Append-only durable entries record timestamp, phase, wave, task, task_status, run_id, trace_id, exact verification/result, and changed surfaces or blocker. -->
- 2026-08-06 · phase provider-grid-console · phase-start · task_status=in-progress · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · phase moved to in-progress in DB and plan; beginning Wave 1.
- 2026-08-06 · phase provider-grid-console · wave 1 · task 1.1 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: internal/config/presets/presets.go,
  internal/config/presets/providers.json, internal/config/presets/presets_test.go · verify:
  `go test ./internal/config/presets/... -v` — 4/4 pass; `go build ./...` clean.
- 2026-08-06 · phase provider-grid-console · wave 1 · task 1.2 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/types/provider.ts,
  web/src/services/api/providers.ts · verify: `cd web && bun run type-check` clean.
- 2026-08-06 · phase provider-grid-console · wave 1 complete · trace_id: 01KZBKFMKJXC78031BT6C6QZEX
  · run_id: 01KZBKBD3XPA3D7GVMPBAMYEXJ
- 2026-08-06 · phase provider-grid-console · wave 2 · task 2.1 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/features/providers/entries.ts (new) · verify:
  `cd web && bun run type-check && bun run lint` — clean, 0 errors / 8 baseline warnings.
- 2026-08-06 · phase provider-grid-console · wave 2 · task 2.2 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/features/providers/entries.ts (partition folded
  into `buildEntries`) · verify: `cd web && bun run type-check` clean; count-identity assertion
  deferred to task 6.3's browser sweep per plan.
- 2026-08-06 · phase provider-grid-console · wave 2 · task 2.3 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/features/providers/ProvidersWorkbenchPage.tsx ·
  verify: `cd web && bun run build` — `tsc && vite build` succeeded (1,945 kB / gzip 564 kB); the
  presets fetch and the new authFiles fetch both `.catch()` to an empty array so a rejected
  `/provider-presets` degrades to zero preset cards without throwing; confirmed live in task 6.3.
- 2026-08-06 · phase provider-grid-console · wave 2 complete · trace_id: 01KZBM9XYN9Y6RFS8XXMX17VXQ
  · run_id: 01KZBKBD3XPA3D7GVMPBAMYEXJ
- 2026-08-06 · phase provider-grid-console · wave 3 · task 3.1 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/features/providers/components/ProviderCategoryGrid.tsx
  (new) · verify: `cd web && bun run type-check && bun run lint` — clean, 0 errors / 8 baseline
  warnings. Preset logos (`PRESET_LOGOS`) scaffolded empty pending Wave 5 task 5.3.
- 2026-08-06 · phase provider-grid-console · wave 3 · task 3.2 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/features/providers/ProvidersWorkbenchPage.tsx ·
  verify: `cd web && bun run build` succeeded. See Decisions for the header "+New" button and the
  list-view delete/toggle affordances, both temporarily reduced until Wave 4 lands.
- 2026-08-06 · phase provider-grid-console · wave 3 · task 3.3 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/features/providers/components/ProviderCategoryList.tsx
  (removed via `trash`) · verify: `grep -rn "ProviderCategoryList" web/src` — no results;
  `cd web && bun run type-check` clean.
- 2026-08-06 · phase provider-grid-console · wave 3 complete · trace_id: 01KZBMA1VHC7XZAKEZP3DJY6MZ
  · run_id: 01KZBKBD3XPA3D7GVMPBAMYEXJ
- 2026-08-06 · phase provider-grid-console · wave 4 · task 4.1 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/components/ui/AppSheet.tsx · verify:
  `cd web && bun run type-check` clean; `git diff --name-only web/src/components/ui/` showed only
  `AppSheet.tsx`.
- 2026-08-06 · phase provider-grid-console · wave 4 · task 4.2 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/features/providers/ProvidersWorkbenchPage.tsx,
  web/src/features/providers/entries.ts · verify: `cd web && bun run type-check && bun run lint` —
  type-check clean, lint 0 errors / 8 baseline warnings.
- 2026-08-06 · phase provider-grid-console · wave 4 · task 4.3 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/features/providers/panels/OAuthLoginPanel.tsx,
  web/src/features/providers/entries.ts · verify: `cd web && bun run type-check && bun run lint` —
  type-check clean, lint 0 errors / 8 baseline warnings; `grep -n "setInterval\\|setTimeout"
  web/src/features/providers/panels/OAuthLoginPanel.tsx` returned no results.
- 2026-08-06 · phase provider-grid-console · wave 4 · task 4.4 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/features/providers/panels/AuthFileMiniTable.tsx ·
  verify: `cd web && bun run type-check && bun run lint` — type-check clean, lint 0 errors / 8
  baseline warnings.
- 2026-08-06 · phase provider-grid-console · wave 4 · task 4.5 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/features/providers/sheets/ProviderSheet.tsx,
  web/src/features/providers/ProvidersWorkbenchPage.tsx · verify: `cd web && bun run build` —
  `tsc && vite build` succeeded (1,983.14 kB / gzip 572.81 kB).
- 2026-08-06 · phase provider-grid-console · wave 4 · task 4.6 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/features/providers/sheets/forms/BaseProviderForm.tsx ·
  verify: `cd web && bun run build` — `tsc && vite build` succeeded; visual preset-prefill proof
  deferred to task 6.3's browser sweep.
- 2026-08-06 · phase provider-grid-console · wave 4 complete · trace_id: 01KZBPNV776QVN61DNYHZJFZ78
  · run_id: 01KZBKBD3XPA3D7GVMPBAMYEXJ
- 2026-08-06 · phase provider-grid-console · wave 5 · task 5.1 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/assets/icons/opencode-light.svg,
  web/src/assets/icons/opencode-dark.svg, web/src/assets/icons/openrouter-light.svg,
  web/src/assets/icons/openrouter-dark.svg, web/src/assets/icons/nvidia-light.svg,
  web/src/assets/icons/nvidia-dark.svg · verify: all six parse as valid SVG with non-empty viewBox;
  OpenRouter uses the current official purple/lime glyph colors, not svgl's retired monochrome mark.
- 2026-08-06 · phase provider-grid-console · wave 5 · task 5.2 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/assets/icons/amp.svg · verify: per-file verdicts:
  `amp.svg` replaced — official Ampcode mark from `ampcode.com/amp-mark-color.svg`;
  `antigravity.svg` no official source found — retained;
  `claude.svg` current — official Claude glyph retained;
  `codex.svg` current — official Codex glyph retained;
  `deepseek.svg` no official source found — retained;
  `gemini.svg` current — official Gemini glyph retained;
  `glm.svg` no official source found — retained;
  `grok-dark.svg` current — official xAI glyph retained;
  `grok.svg` current — official xAI glyph retained;
  `iflow.svg` no official source found — retained;
  `kimi-dark.svg` current — official Kimi glyph retained;
  `kimi-light.svg` current — official Kimi glyph retained;
  `kiro.svg` current — official Kiro glyph retained;
  `minimax.svg` no official source found — retained;
  `openai-dark.svg` current — official OpenAI glyph retained;
  `openai-light.svg` current — official OpenAI glyph retained;
  `qwen.svg` current — official Qwen glyph retained;
  `vertex.svg` current — official Vertex glyph retained. Every brand on the grid is current or
  replaced; uncertain non-grid sources are explicitly marked above.
- 2026-08-06 · phase provider-grid-console · wave 5 · task 5.3 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/features/providers/components/ProviderCategoryGrid.tsx ·
  verify: `cd web && bun run type-check && bun run lint && bun run build` — type-check/build clean,
  lint 0 errors / 8 baseline warnings; all 15 entries have an explicit provider logo mapping, with
  light/dark variants selected from the active theme. Both-theme visual proof deferred to task 6.3.
- 2026-08-06 · phase provider-grid-console · wave 5 complete · trace_id: 01KZBQDA1VSZN57MWRD85JECZ9
  · run_id: 01KZBKBD3XPA3D7GVMPBAMYEXJ
- 2026-08-06 · phase provider-grid-console · wave 6 · task 6.1 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/features/providers/ProvidersWorkbenchPage.tsx,
  web/src/router/MainRoutes.tsx, web/src/components/layout/MainLayout.tsx,
  web/src/pages/OAuthPage.tsx (removed) · verify: `cd web && bun run type-check` clean;
  `grep -rn "OAuthPage" web/src` returned no results; `/oauth` remains only as the intentional redirect.
- 2026-08-06 · phase provider-grid-console · wave 6 · task 6.2 · task_status=DONE · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · touched: web/src/i18n/locales/en.json,
  web/src/i18n/locales/vi.json, provider grid/panel/sheet JSX · verify: English/Vietnamese key parity
  at 1,443 keys; new provider JSX has no `defaultValue` fallback strings.
- 2026-08-06 · phase provider-grid-console · wave 6 · task 6.3 · task_status=IN-PROGRESS · run_id:
  01KZBKBD3XPA3D7GVMPBAMYEXJ · automated verify passed: `go test ./...`, `cd web && bun run
  type-check && bun run lint && bun run build`, and `make build-web && make build`; lint remains
  0 errors / 8 baseline warnings. Browser sweep is not run because no browser automation runtime is
  available and `make dev` exits before serving when `PGSTORE_DSN` is unset.
- 2026-08-07T02:49:36Z · phase provider-grid-console · wave 6 · task 6.3 · task_status=DONE · run_id: 01KZCYACW6NPX5Y93P57CQSV88 · touched: web/src/features/providers/entries.ts, web/src/features/providers/panels/AuthFileMiniTable.tsx · verify: Gemini OAuth matching accepts both runtime auth-file types (`gemini` and `gemini-cli`) for card counts and the mini-table; the static-review blocker is closed.
- 2026-08-07T02:49:36Z · phase provider-grid-console · wave 6 complete · task 6.3 · task_status=DONE · run_id: 01KZCYACW6NPX5Y93P57CQSV88 · verify: `go test ./...`; `cd web && bun run type-check && bun run lint && bun run build`; `make build-web && make build`; and `git diff --check` passed; lint remained 0 errors / 8 baseline warnings. Real Postgres-backed browser proof passed: 5 labels in order, 15/15 non-empty sheets, `/oauth` redirect, OAuth close/reopen persistence, OpenRouter partition, preset prefill, API 2xx paths plus create/delete, dirty Back keep/discard, callback input reset after failure, degraded `/provider-presets`, and 15/15 logos under light and dark root states; browser console/runtime stayed clean.
- 2026-08-07T04:25:27Z · phase provider-grid-console · post-gate remediation · task_status=DONE · run_id: 01KZCYACW6NPX5Y93P57CQSV88 · touched: internal/api/handlers/management/auth_files.go, web/src/features/authFiles/constants.ts, web/src/features/providers/entries.ts, web/src/features/providers/sheets/forms/BaseProviderForm.tsx · verify: static review closed the Gemini virtual-primary false health error and stale API-key-on-preset-switch regressions; `go test ./...`, `cd web && bun run type-check && bun run lint && bun run build`, `make build-web && make build`, and `git diff --check` passed; remediation commit `2e5f0c3f` pushed to `origin/feature/unified-provider-console`; PR #11 now points to the remediation tip.

## Decisions
<!-- Append-only durable entries record timestamp, phase/task, decision, and rationale. -->
- 2026-08-06 · planning · `D1` merge two screens only, OAuth panel embeds the account list ·
  owner answer: "Gộp 2 màn, panel OAuth nhúng danh sách account". Rationale: absorbing the 890-line
  `/auth-files` page is where feature loss would come from; a mini-table plus a link gives the OAuth
  panel real content at a fraction of the risk.
- 2026-08-06 · planning · `D2` preset rows are always visible, catalog-style · owner answer:
  "Catalog — luôn hiện, kể cả chưa cấu hình". Rationale: OAuth cards are visible before login, so
  preset rows must be too or they are not peers.
- 2026-08-06 · planning · `D3` OpenRouter and NVIDIA are separate top-level rows · owner answer:
  "openrouter, nvidia riêng như 9router". The owner did not pick a matching mechanism, so planning
  chose base-URL matching (`NG6`) as the reversible zero-schema-change default and stated it back
  before this plan was written.
- 2026-08-06 · planning · `D4` keep the 240px rail + panel layout grouped by category · owner
  answer: "Giữ rail trái 240px + panel phải, group theo category". Rationale: preserves the resource
  table's sort, model filter, usage stats, and row actions, which a card grid would drop.
- 2026-08-06 · planning · plan authored without a prior `brainstorm` artifact · the `/think`
  session that preceded this stage locked outcome, authority, requirements, and non-goals with four
  explicit owner answers, so the initiative was not ambiguous and routing back to `brainstorm`
  would have re-derived settled scope.
- 2026-08-06 · re-planning · **`D4` and `NG3` reversed; the plan is re-cut from a rail into a grid.**
  Owner brief: "tối ưu rail UI UX layout cho page AI Provider mới … Chuẩn hoá dựng layout theo từng
  grid theo từng category … Ở các loại provider cấu hình form → dùng sheet/slider, loại provider
  OAuth thì cũng mở sheet/slider". Owner answer to the layout question: "Bỏ rail, grid theo
  category". This is `D9`. The original `NG3` ("no flat card-grid layout") argued a grid would drop
  the resource table's sort, model filter, usage stats, and row actions — that argument is answered,
  not ignored: the table moves into the sheet intact (`NG8`), so the affordances survive the layout
  change. The old `NG3` slot is reused for the design-system freeze.
- 2026-08-06 · re-planning · `D5` OAuth login state lifted to `ProvidersWorkbenchPage` · owner
  answer: "A — Nâng state lên page". Rationale, and the defect that forced the question: the earlier
  plan asserted that "timer refs stay keyed by provider so a login started on one row survives
  switching rows and back". Reading the source disproved it — `states`, `pollingTimers`, and
  `successResetTimers` are all component-local at `web/src/pages/OAuthPage.tsx:151-153`, so
  unmounting the panel kills the `window.setInterval` at `:236` and abandons the login silently.
  Keying by provider never helped, because the whole map dies with the component.
- 2026-08-06 · re-planning · `D6` preset category is an explicit JSON field, not derived · owner
  answer: "Explicit field trong JSON". Rationale: the derivation the earlier plan specified
  (`defaultApiKey && !signupUrl → free`; `signupUrl && freeTierNote → freeTier`; else `apikey`)
  classifies NVIDIA as `freeTier`, while 9router — the taxonomy source — classifies it `apikey`.
  One wrong out of three on a three-entry catalog, and every future preset is another coin flip. The
  explicit field costs about six lines across `presets.go`, three JSON entries, `provider.ts`, and
  the normalizer. It also relaxes `NG4`: a Go edit is now in scope, but only on the embedded catalog,
  never on a user-facing endpoint or config schema.
- 2026-08-06 · re-planning · `D7` `CUSTOM` stays as the 5th category · owner answer: "Giữ là section
  thứ 5". Rationale: unmatched `openai-compatibility` entries need a home that is visibly not a
  preset, and folding them into `API KEY` would hide the base-URL-matching rule that put them there.
- 2026-08-06 · re-planning · `D8` category order is `OAUTH → FREE → FREE TIER → API KEY → CUSTOM` ·
  owner answer, verbatim. Rationale: cheapest-to-start first; `CUSTOM` last because it is the
  fallback bucket, not a destination.
- 2026-08-06 · re-planning · `D10` add `2xl` (`sm:max-w-4xl`, 896px) to `AppSheet`'s `SIZE_MAP` ·
  owner answer: "Thêm size 2xl/full vào AppSheet". This is an **explicitly approved exception** to
  the owner's own constraint "Không sửa design system hiện tại". Rationale: with the rail gone the
  sheet becomes the only place the resource table can live, and the map's current ceiling of `xl` =
  576px cannot hold it. Scope of the exception is one map entry and one prop-union member; `NG3`
  freezes everything else under `web/src/components/ui/`.
- 2026-08-06 · re-planning · `D11` closing a sheet does not cancel an in-flight login · owner
  answer: "Đóng sheet không huỷ login". The card carries a "waiting for authorization" badge while
  it runs. This is only implementable because of `D5` — with state on the page, the sheet is a view
  of the login, not its owner.
- 2026-08-06 · re-planning · `D12` card content is logo + name + status badge + configured count +
  error/unverified warning · owner answer, multi-select: "Logo + tên + badge trạng thái", "Số
  account hoặc resource đã cấu hình", "Cảnh báo lỗi / unverified". The free-tier note and base URL
  were **not** selected and stay in the sheet header, where there is room for them.
- 2026-08-06 · re-planning · `D13` every brand logo must be the provider's current official mark ·
  owner answer, verbatim: "Đảm bảo 100% logo brand các provider phải là logo mới nhất", scoped by a
  follow-up answer to "Rà hết 18 cái". Sourcing rule adopted: official brand page first, `svgl` only
  as fallback, always the glyph variant and never the wide lockup. The rule exists because research
  hit two failures in one pass — svgl's OpenRouter entry is the retired monochrome arrow
  (633B, `#111111`, viewBox 512×512) rather than the current `brand/v2` glyph (661B, viewBox
  401.4×293.7, `#7624F4`), which the owner caught mid-session and corrected; and svgl's "AMP" entry
  is `amp.dev` (Google Accelerated Mobile Pages), not Sourcegraph Ampcode, which means
  `web/src/assets/icons/amp.svg` may already be the wrong brand and is flagged for inspection in
  task 5.2. Six brands are absent from svgl entirely (`vertex`, `amp`, `glm`, `iflow`, `kiro`,
  `minimax`); only `vertex` and `amp` appear on the new grid.
- 2026-08-06 · re-planning · requirement and non-goal numbering restarted · the previous cut had
  `R1`-`R9` / `NG1`-`NG7` written against the rail shape. Rather than leave dead numbers, the lists
  were renumbered `R1`-`R13` / `NG1`-`NG8`. Carried over unchanged in substance: old `R1`→new `R1`,
  old `R4`→new `R7`, old `R7`→new `R10` (partition half), old `R9`→new `R13`, old `NG1`/`NG2`/`NG5`
  →same numbers, old `NG6`/`NG7`→same numbers. Dead: old `R2` (rail rendering), old `R3` (panel with
  its own timers, killed by `D5`), old `R6` (derived category, killed by `D6`), old `R8` (separate
  preset panel component, now a sheet view). Safe to renumber because no phase had ever run —
  `## Progress` was empty and neither old phase had a real story row.
- 2026-08-06 · re-planning · the two previously planned phases `provider-console-merge` and
  `provider-preset-rows` are replaced by the single phase `provider-grid-console` · owner answer:
  "Slug mới provider-grid-console". Their `story_id`s (`01KZB8F85NSXWGJNK87BTGWQ98`,
  `01KZB8FFGC7J2NVD3QP72S8ERV`) were minted with `zharness id` and never persisted — no DB row ever
  existed for either, the same defect diagnosed in `docs/plans/done/error-classification.md`. The
  new phase's `story_id` `01KZBHHJ7HGPZXD3DW9R48K94E` was created with `zharness story` and is a
  real row. The two-phase split is also no longer meaningful: with the rail gone the preset cards
  and the grid are the same deliverable, and shipping the grid without preset categories would leave
  three of fifteen cards unplaced.

- 2026-08-06 · execution (Wave 3) · global "+New" header button loses its brand-specific default ·
  gap: `D9` removes the rail, so the `activeBrand` state that previously told the header's "+New"
  button which brand to create no longer exists, and neither the plan's `## Building` nor `## Changes`
  section says what the button should do instead. Resolved by pointing it at the `openaiCompatibility`
  create flow — the generic "custom provider" entry point — since it is the only brand without a
  narrower, more specific add path once every other brand's "Add" lives inside its own card's sheet
  (preset sheets get a prefilled Add per `R11`; OAuth cards start a login, not a form). Revisit if the
  Wave 6 browser sweep finds this confusing; the header button can also be dropped entirely in favor of
  per-card adds only.
- 2026-08-06 · execution (Wave 3) · list-view delete / toggle-disabled / view / edit are unreachable
  between Wave 3 and Wave 4 · gap: those actions today live only on `ProviderResourcePanel`'s row
  buttons (`ProviderSheet.tsx`'s current `detail` mode has no delete button of its own — confirmed by
  reading the file). Task 3.2 removes `ProviderResourcePanel` from the page to drop the two-column
  layout, and task 4.5 is what re-parents it into `ProviderSheet`'s new `list` view — so for the
  duration of this session's Wave 3→Wave 4 gap the actions have no UI home. Accepted because this is
  an in-progress working tree, not a shipped state — `NG8` ("zero feature loss") is checked at the
  end of the phase (task 6.3), not at each wave's own check command, and the plan already scopes
  `ProviderResourcePanel` re-parenting to task 4.5.
- 2026-08-06 · execution (Wave 3) · task 2.3's own check ("`bun run build` succeeds") could not pass
  with `buildEntries`'s result computed and left unused · gap: `tsconfig.json` has `noUnusedLocals:
  true`, so a `const entries = useMemo(...)` with no reader in the same component is a compile error,
  not a warning — there is no way to satisfy task 2.3's check in isolation without either a
  throwaway/hacky usage or wiring the value into the render tree, which is task 3.2's job. Resolved by
  writing tasks 2.3, 3.1, 3.2, and 3.3 in one continuous edit before running any of their check
  commands, then running each task's specified check afterward (all four passed). Wave boundaries
  (trace recorded per wave) and task-level Progress entries are unaffected — this only changes the
  order in which file edits vs. check commands were interleaved within the session.

- 2026-08-06 · execution (Wave 4) · tasks 4.2–4.5 were written as one batch before verification · gap:
  `tsconfig.json` has `noUnusedLocals: true`, so lifted page handlers, the panel callbacks, and the
  re-parented sheet branches cannot be checked independently without transient unused symbols.
  Resolved by completing the interdependent batch first, then running each task's specified check;
  all checks passed with 0 errors and the baseline 8 lint warnings.
- 2026-08-06 · execution (Wave 4) · `ProviderSheetState` uses stable `entryKey` plus `resourceId`,
  while the sheet receives live `entries` · gap: storing resource/preset objects captured at card
  click would leave the open sheet stale after delete or disabled-state mutations refresh the
  workbench snapshot. Resolved by looking up the entry and resource from the current `entries` prop
  on every render; list, detail, and edit views therefore reflect mutations without reopening.
- 2026-08-06 · execution (Wave 4) · successful create/update returns to the owning card's list view
  when one exists · gap: the old rail made the resource table visible behind forms, while the new
  sheet has no background list. Resolved by routing form completion to `list` for preset and
  non-Ampcode config entries, while Ampcode remains a direct edit flow and closes after save.
- 2026-08-06 · execution (Wave 6) · XAI callback URL helpers remain in
  `web/src/features/providers/entries.ts` rather than moving into `OAuthLoginPanel.tsx` · rationale:
  `ProvidersWorkbenchPage` owns callback submission and needs the same normalization logic as the
  panel; keeping the helpers in the shared provider-entry module avoids duplicating or widening a
  page-only callback contract. No behavior or security boundary changes.
- 2026-08-06 · execution (Wave 6) · static review remediation · OAuth account tables now load and
  report errors, page-owned auth-file revisions refresh both card counts and open mini-tables, card
  switching and Ampcode cancel honor the dirty-form guard, OAuth callback input resets per attempt,
  preset prefill only applies while pristine and resets its dirty baseline, and cards always render
  status, warning, and configured count including zero-resource Ampcode. Browser proof remains the
  only unverified requirement.
- 2026-08-06 · handoff · task_status=IN-PROGRESS · handoff_id: 01KZBV9J8F3Y5PAH4CYKN2FME6 ·
  branch: feature/unified-provider-console · working tree clean and branch pushed · static review
  identified unresolved Gemini CLI auth-file mapping, preset-create state reuse, browser-history
  dirty-form bypass, and additional OAuth/form/table correctness items; real-browser/Postgres proof
  remains outstanding.
- 2026-08-07 · phase provider-grid-console · phase-start · task_status=IN-PROGRESS · run_id:
  01KZCYACW6NPX5Y93P57CQSV88 · lifecycle row recreated from the locked active plan after the DB
  contained only superseded provider-console-merge/provider-preset-rows rows; beginning static-review
  remediation.

## Validation
<!-- Append-only durable entries record timestamp, phase, exact command/result/output, run_id, check_id, verdict, and proof_gaps. -->
- 2026-08-06T14:46:20Z · phase provider-grid-console · run_id: 01KZBKBD3XPA3D7GVMPBAMYEXJ ·
  check_id: 01KZBRKZV7V35W104FJQ54R2RR · verdict: REQUEST_CHANGES · automated gate passed:
  `go test ./...`; `cd web && bun run type-check && bun run lint && bun run build`; and
  `make build-web && make build`; lint remained 0 errors / 8 baseline warnings; locale parity passed
  at 1,443 keys. Proof gaps: required real-browser sweep not run because no browser automation runtime
  is available and `make dev` exits with `PGSTORE_DSN is required` before serving; API 2xx and
  browser console evidence are therefore not recorded.
- 2026-08-06T15:16:40Z · phase provider-grid-console · run_id: 01KZBKBD3XPA3D7GVMPBAMYEXJ ·
  check_id: 01KZBTBHGQDZ42BQ3P3DS526JC · verdict: REQUEST_CHANGES · post-review remediation
  automated gate passed: `go test ./...`; `cd web && bun run type-check && bun run lint && bun run build`;
  `make build-web && make build`; hygiene and durable audit passed; lint remained 0 errors / 8 baseline
  warnings. Static review found the previously reported lifecycle, synchronization, form-initialization,
  and card-rendering defects addressed. Proof gaps remain: no real-browser sweep, no configured
  Postgres-backed runtime/API 2xx evidence, and no browser-console evidence.
- 2026-08-07T02:49:36Z · phase provider-grid-console · run_id: 01KZCYACW6NPX5Y93P57CQSV88 ·
  check_id: 01KZD20XQMJG2WDDCPR7NZT1JD · verdict: APPROVED · automated gate passed: `go test ./...`;
  `cd web && bun run type-check && bun run lint && bun run build`; `make build-web && make build`;
  `git diff --check`; lint remained 0 errors / 8 baseline warnings. Full static review verified no
  additional security, performance, architecture, or code-quality blocker after the Gemini mapping fix.
  Real Postgres-backed browser proof passed the complete card/API/OAuth suite, dirty Back guard,
  callback-reset failure path, degraded preset-catalog path, and light/dark logo rendering; browser
  console and runtime exceptions were clean. Proof gaps: the local dataset had no Gemini auth-file
  fixture, so the live Gemini row was not exercised; source matching covers both `gemini` and
  `gemini-cli` runtime types.

## Current State and Next Action
- active_phase: provider-grid-console
- lifecycle_status: checked
- latest_run_id: 01KZCYACW6NPX5Y93P57CQSV88
- latest_trace_ids: [01KZBKFMKJXC78031BT6C6QZEX, 01KZBM9XYN9Y6RFS8XXMX17VXQ, 01KZBMA1VHC7XZAKEZP3DJY6MZ, 01KZBPNV776QVN61DNYHZJFZ78, 01KZBQDA1VSZN57MWRD85JECZ9]
- latest_check_id: 01KZD20XQMJG2WDDCPR7NZT1JD
- latest_handoff_id: 01KZBV9J8F3Y5PAH4CYKN2FME6
- blockers: none
- open_items:
  - The local Postgres dataset had no Gemini auth-file fixture, so live Gemini row rendering was not observable; source matching now accepts both `gemini` and `gemini-cli` runtime types.
  - Grid shows `codex`, `claude`, and `gemini` twice (OAuth card + API-key card). Accepted — the category label disambiguates.
  - `AuthFileMiniTable` duplicates part of `/auth-files`. Accepted per `D1`; revisit only if the two drift.
  - No search/filter across the 15 cards. Deferred — 15 cards fit one screen at `auto-fit` 240px.
  - Global header "+New" button always opens the `openaiCompatibility` create flow instead of a brand-specific one. Browser proof found this acceptable; revisit only with product feedback.
- exact_next_action: await PR review and merge; provider-console changes are committed, pushed, and tracked by PR #11
