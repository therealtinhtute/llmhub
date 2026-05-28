# HANDOFF — Design Token v2 Migration

Date: 2026-05-28
Branch: master (no upstream divergence — 1 commit ahead of origin, all migration work uncommitted)
Continuity mode: harness
Active phase: ALL COMPLETE — ready to commit

---

## Completed This Session

All 4 phases executed and verified:

**Phase 1 — token-foundation** ✅
- Rewrote `web/src/index.css` from 282 lines → ~420 lines
- Added 209 CSS custom properties in oklch format
- Warm cream palette (#faf9f4 base) replacing cool blue-purple
- Extended token system: semantic backgrounds, text hierarchy, status colors, graph tokens, spacing/typography/dimension/layout scales
- Dual mapping: shadcn standard vars (`--background`, `--primary`) + new semantic vars (`--bg-primary`, `--accent`)
- `@theme` block rewritten with 17 extended color mappings
- Shadow simplified to 3 levels, radius to direct px values

**Phase 2 — dark-mode-removal** ✅
- `.dark {}` block deleted from index.css
- `@custom-variant dark` removed
- `dark:` utility classes removed from 21 component/page/feature files
- `useThemeStore.ts` simplified — always resolves to light
- `authFiles/constants.ts` + `quota/constants.ts`: dark color map entries removed, type updated
- `QuotaCard.tsx`: simplified `resolvedTheme` branch removed
- 3 remaining `dark:` references are icon type defs `{ light: string; dark: string }` — intentional, not CSS

**Phase 3 — component-adoption** ✅
- `DiffModal.tsx`: 25 hardcoded hex (#3fb950, #f85149, #388bfd) → `var(--success)`, `var(--error)`, `var(--accent)`
- `sidebar.tsx`: SIDEBAR_WIDTH `16rem` → `260px`, SIDEBAR_WIDTH_ICON `3rem` → `60px`
- `quotaStyles.ts`: kept decorative gold badge gradient as-is
- `ModelMappingDiagram.tsx`: kept decorative category color array as-is

**Phase 4 — verification** ✅
- `bun run build` PASS (dist/index.html 2,143 kB)
- `tsc --noEmit` PASS (zero errors)
- `make build` PASS (Go binary compiled with embedded frontend)
- `grep -c "^\.dark" index.css` → 0
- `grep -c "@custom-variant dark" index.css` → 0
- Agent-browser visual inspection: warm cream bg, blue accent, no regressions

---

## State

All 4 phases complete. **29 files changed, 286 insertions, 216 deletions.** Nothing committed yet.

---

## Blockers

None.

---

## Next Steps

→ **START HERE**: Commit all migration changes as a single conventional commit:
```
cd /home/tinhpt/Lab/llmhub
git add web/src/ 
git commit -m "feat(web): Design Token v2 — warm palette, extended semantic system, dark mode removal"
```

2. Optionally push to origin/master
3. Optionally update design-token-new.json reference or document the migration in CLAUDE.md

---

## Key Decisions Made

- **oklch format** for all color values (not hex)
- **Dual variable mapping**: shadcn vars preserved + new semantic vars added (no shadcn breakage)
- **Light-only**: dark mode fully removed (no dark token definition exists yet)
- **Decorative colors kept as-is**: quotaStyles gold badge, ModelMappingDiagram category colors — not tokenized
- **Sidebar width**: `260px` / `60px` from design tokens (was `16rem` / `3rem`)
- **3 files with intentional `dark:` icon refs**: OAuthPage, SystemPage, authFiles/constants — these are image variant types, not CSS
