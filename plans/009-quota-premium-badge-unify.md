# Plan 009: Unify premium plan badge to simple amber style

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat c0fff0f..HEAD -- web/src/components/quota/quotaStyles.ts web/src/features/authFiles/components/AuthFileQuotaSection.tsx`
> If either file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `c0fff0f`, 2026-06-18

## Why this matters

The "premium" plan badge (shown for Codex Pro and Gemini Ultra tiers) has two
different implementations:

- **Quota Management page** (`quotaStyles.ts:premiumPlanValue`): a complex
  multi-shadow radial gradient with hardcoded `rgba` and hex colors. Looks
  visually heavy, breaks in dark mode (the baked-in white gradients invert
  poorly), and is ~200 characters of Tailwind that is hard to maintain.
- **Auth File detail panel** (`AuthFileQuotaSection.tsx:quotaStyleMap.premiumPlanValue`):
  a simple `bg-amber-500/15 border border-amber-500/30 text-amber-600` badge
  that uses Tailwind's opacity scale. Adapts naturally to dark mode, is easy
  to read, and matches Shadcn badge conventions.

Both surfaces show the same "premium tier" information. Making the Quota
Management page use the simpler style removes the dark-mode regression risk and
makes the codebase consistent.

## Current state

**Files in scope:**

1. `web/src/components/quota/quotaStyles.ts` — the `premiumPlanValue` key is
   at line 79–80. Only this key changes.

2. `web/src/features/authFiles/components/AuthFileQuotaSection.tsx` — the
   `quotaStyleMap.premiumPlanValue` at line 52–53 is the **reference** (target
   style). It does **not** change.

**Excerpt A — `web/src/components/quota/quotaStyles.ts:79–80` (CHANGE THIS):**
```ts
  premiumPlanValue:
    'relative inline-flex items-center font-bold text-[12px] px-2 py-0.5 rounded-full overflow-visible isolate [background:radial-gradient(circle_at_18%_24%,rgba(255,255,255,0.96)_0%,rgba(255,255,255,0.72)_18%,rgba(255,255,255,0)_42%),linear-gradient(135deg,#fff9e3_0%,#ffe07f_52%,#e0aa14_100%)] border border-[rgba(217,165,22,0.72)] shadow-[0_1px_3px_rgba(133,92,0,0.16),0_0_0_1px_rgba(255,255,255,0.22)_inset,0_0_16px_rgba(255,214,98,0.28)] text-[#6b4b00] [text-shadow:0_1px_0_rgba(255,255,255,0.55)] capitalize',
```

**Excerpt B — `web/src/features/authFiles/components/AuthFileQuotaSection.tsx:52–53` (DO NOT CHANGE — reference only):**
```ts
  premiumPlanValue:
    'inline-flex items-center font-bold text-[12px] px-2 py-[2px] bg-amber-500/15 border border-amber-500/30 text-amber-600 capitalize',
```

The target value to use in `quotaStyles.ts` is:
```ts
  premiumPlanValue:
    'inline-flex items-center font-bold text-[12px] px-2 py-[2px] bg-amber-500/15 border border-amber-500/30 text-amber-600 capitalize',
```

## Commands you will need

All commands run from the `web/` directory.

| Purpose    | Command              | Expected on success                   |
|------------|----------------------|---------------------------------------|
| Type-check | `bun run type-check` | exit 0, no errors                     |
| Lint       | `bun run lint`       | exit 0, or pre-existing warnings only |

## Scope

**In scope** (the only file you should modify):
- `web/src/components/quota/quotaStyles.ts`

**Out of scope** (do NOT touch):
- `web/src/features/authFiles/components/AuthFileQuotaSection.tsx` — reference only; do not modify.
- `web/src/components/quota/quotaConfigs.ts` — uses `styles.premiumPlanValue` but does not hardcode the class; the style map change flows through automatically.
- Any other file.

## Git workflow

- Branch: `advisor/009-quota-premium-badge`
- Single commit: `fix(web): simplify premium plan badge to amber-500 token`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Replace premiumPlanValue in quotaStyles.ts

Open `web/src/components/quota/quotaStyles.ts`.

Locate lines 79–80 (`premiumPlanValue`). The entire current value is the long
gradient string shown in "Excerpt A" above.

Replace those two lines with:
```ts
  premiumPlanValue:
    'inline-flex items-center font-bold text-[12px] px-2 py-[2px] bg-amber-500/15 border border-amber-500/30 text-amber-600 capitalize',
```

**Verify**:
```
grep -n "premiumPlanValue" web/src/components/quota/quotaStyles.ts
```
Expected output — must NOT contain `radial-gradient` or `rgba`:
```
80:  premiumPlanValue:
81:    'inline-flex items-center font-bold text-[12px] px-2 py-[2px] bg-amber-500/15 border border-amber-500/30 text-amber-600 capitalize',
```

### Step 2: Type-check

From `web/`:
```
bun run type-check
```
Expected: exit 0. (String-only change in the style map.)

### Step 3: Lint

From `web/`:
```
bun run lint
```
Expected: exit 0 or same pre-existing warnings.

## Test plan

No automated tests cover Tailwind class strings. Manual visual gate:

1. Run `bun dev` from `web/`.
2. Load the Quota Management page and refresh quota for a Codex Pro account.
3. Confirm the plan badge reads "Pro" (or "Pro Lite") and is styled as an amber
   badge with a subtle border — not a gold gradient.
4. Toggle the app between light and dark mode; confirm the badge remains legible
   in both (amber-500/15 + amber-600 adapts correctly via Tailwind opacity).

## Done criteria

- [ ] `bun run type-check` exits 0 (from `web/`)
- [ ] `bun run lint` exits 0 (from `web/`)
- [ ] `grep "premiumPlanValue" web/src/components/quota/quotaStyles.ts` shows `bg-amber-500/15` and does NOT show `radial-gradient`
- [ ] Only `web/src/components/quota/quotaStyles.ts` is modified (`git diff --name-only`)
- [ ] `plans/README.md` status row updated to DONE

## STOP conditions

Stop and report back if:

- The `premiumPlanValue` key at line 79–80 does not match Excerpt A (codebase drifted).
- Type-check or lint produce new errors after this change.
- The fix appears to require touching an out-of-scope file.

## Maintenance notes

- The `PREMIUM_CODEX_PLAN_TYPES` set in `web/src/components/quota/quotaConfigs.ts`
  controls which plan types receive `premiumPlanValue` styling. If new premium
  tiers are added, that set must be updated — this plan does not change that logic.
- `PREMIUM_GEMINI_CLI_TIER_IDS` in the same file does the same for Gemini CLI.
- If dark-mode–specific amber variants are later needed (e.g. lighter text in
  dark mode), use Tailwind's `dark:` prefix on `text-amber-600` rather than
  reverting to the gradient approach.
