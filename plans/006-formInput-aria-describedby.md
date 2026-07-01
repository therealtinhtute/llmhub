# Plan 006: Add aria-describedby to FormInput so screen readers announce error and hint text

> **Executor instructions**: Follow this plan step by step. Run every
> verification command before moving on. If anything in "STOP conditions" occurs,
> stop and report — do not improvise. Update the status row in `plans/README.md`
> when done.
>
> **Drift check (run first)**:
> `git diff --stat 5f81cee..HEAD -- web/src/components/ui/FormInput.tsx`
> If the file changed since this plan was written, compare the
> "Current state" excerpt against live code before proceeding.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: accessibility (A11Y)
- **Planned at**: commit `5f81cee`, 2026-06-12

## Why this matters

`FormInput` renders error and hint text in `<p>` elements next to the input. The
input already sets `aria-invalid={error ? true : undefined}`, which tells screen
readers that the field is invalid. But without `aria-describedby` pointing at the
error `<p>`, the screen reader must guess the relationship or skip the error text
entirely — the standard is to announce it when the field receives focus.

The fix is small: generate stable `id` attributes for the error and hint paragraphs
using the existing `id` prop, then add `aria-describedby` on the `ShadcnInput`. When
`id` is not provided by the caller, fall back to a hint that omits the association
(safe — `aria-describedby` with no valid `id` is harmless but may trigger lint
warnings; skipping it entirely is cleaner).

## Current state

**`web/src/components/ui/FormInput.tsx`** (full file, 38 lines):

```tsx
import type { InputHTMLAttributes, ReactNode } from 'react';
import { Input as ShadcnInput } from './Input';
import { cn } from '@/lib/utils';

interface FormInputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  hint?: string;
  error?: string;
  rightElement?: ReactNode;
}

function FormInput({ label, hint, error, rightElement, className, id, ...rest }: FormInputProps) {
  return (
    <div className="flex flex-col gap-1">
      {label ? (
        <label htmlFor={id} className="text-sm font-medium text-foreground">
          {label}
        </label>
      ) : null}
      <div className={cn('relative', rightElement && 'flex items-center')}>
        <ShadcnInput
          id={id}
          className={cn(error && 'border-destructive', rightElement && 'pr-9', className)}
          aria-invalid={error ? true : undefined}
          {...rest}
        />
        {rightElement ? (
          <div className="absolute right-2 top-1/2 -translate-y-1/2">{rightElement}</div>
        ) : null}
      </div>
      {error ? <p className="text-xs text-destructive">{error}</p> : null}
      {hint && !error ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
    </div>
  );
}

export { FormInput };
```

The input at line 21–26 has `aria-invalid` but no `aria-describedby`. The error `<p>`
at line 31 and hint `<p>` at line 32 have no `id`.

## Scope

**In scope**: `web/src/components/ui/FormInput.tsx` only.
**Out of scope**: callers of `FormInput` — the fix is backward-compatible (adds
optional attributes, removes nothing, callers need no changes). Do not touch
`Input.tsx` (the Shadcn primitive) or any page/feature that uses `FormInput`.

## Git workflow

- Branch: `advisor/006-form-input-aria-describedby`
- Commit: `fix(web): add aria-describedby to FormInput error and hint text`
- Do NOT push or open a PR unless instructed.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Type-check | `cd web && bun run type-check` | exit 0 |
| Tests | `cd web && bun run test:run` | 60 tests pass |
| Lint | `cd web && bun run lint` | exit 0 errors |

## Steps

### Step 1: Derive stable IDs for error and hint paragraphs

Add two derived `id` constants inside the `FormInput` function body, before the
`return`. Use the existing `id` prop as a prefix:

```ts
const errorId = id ? `${id}-error` : undefined;
const hintId = id ? `${id}-hint` : undefined;
```

`id` is already destructured from props at the function signature. These constants
are `undefined` when `id` is not provided — that is intentional (see "STOP
conditions" below for why this is safe).

### Step 2: Add aria-describedby to ShadcnInput

In the `ShadcnInput` element (lines 21–26), add an `aria-describedby` attribute.
The value should include the `errorId` when there is an error, or `hintId` when
there is a hint but no error — matching the same conditional logic used for the `<p>`
elements. When neither applies, the prop is `undefined` (browser ignores it):

```tsx
<ShadcnInput
  id={id}
  className={cn(error && 'border-destructive', rightElement && 'pr-9', className)}
  aria-invalid={error ? true : undefined}
  aria-describedby={error ? errorId : hint ? hintId : undefined}
  {...rest}
/>
```

### Step 3: Add id attributes to the error and hint paragraphs

Update the two `<p>` elements at the bottom of the JSX to use the derived IDs:

```tsx
{error ? <p id={errorId} className="text-xs text-destructive">{error}</p> : null}
{hint && !error ? <p id={hintId} className="text-xs text-muted-foreground">{hint}</p> : null}
```

### Step 4: Final file should look like this

```tsx
import type { InputHTMLAttributes, ReactNode } from 'react';
import { Input as ShadcnInput } from './Input';
import { cn } from '@/lib/utils';

interface FormInputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  hint?: string;
  error?: string;
  rightElement?: ReactNode;
}

function FormInput({ label, hint, error, rightElement, className, id, ...rest }: FormInputProps) {
  const errorId = id ? `${id}-error` : undefined;
  const hintId = id ? `${id}-hint` : undefined;
  return (
    <div className="flex flex-col gap-1">
      {label ? (
        <label htmlFor={id} className="text-sm font-medium text-foreground">
          {label}
        </label>
      ) : null}
      <div className={cn('relative', rightElement && 'flex items-center')}>
        <ShadcnInput
          id={id}
          className={cn(error && 'border-destructive', rightElement && 'pr-9', className)}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? errorId : hint ? hintId : undefined}
          {...rest}
        />
        {rightElement ? (
          <div className="absolute right-2 top-1/2 -translate-y-1/2">{rightElement}</div>
        ) : null}
      </div>
      {error ? <p id={errorId} className="text-xs text-destructive">{error}</p> : null}
      {hint && !error ? <p id={hintId} className="text-xs text-muted-foreground">{hint}</p> : null}
    </div>
  );
}

export { FormInput };
```

### Step 5: Run all gates

```bash
cd web && bun run type-check
cd web && bun run test:run
cd web && bun run lint
```

**Verify**: all exit 0.

## Test plan

`FormInput` is a presentational component with no store dependency. A unit test
can verify the wiring. If the test suite already has a `FormInput.test.tsx`, extend
it. If not, create `web/src/components/ui/__tests__/FormInput.test.tsx` following
the pattern of `web/src/utils/__tests__/format.test.ts`:

```ts
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { FormInput } from '../FormInput';
```

Wait — the test suite uses jsdom (configured in `vitest.config.ts`) but does NOT have
`@testing-library/react`. Do NOT install it just for this plan; write DOM-based tests
using `document.createElement` or skip the test and note it. **The type-check and lint
gates are the primary automated signal.**

Manual verification:
1. Inspect any form field on LoginPage in DevTools (element tab).
2. Confirm the `<input>` has `aria-describedby="<id>-error"` (or `-hint`).
3. Confirm the `<p>` below has the matching `id`.
4. Confirm that inputs without an `id` prop have no `aria-describedby` attribute.

## Done criteria

- [ ] `grep -n "aria-describedby" web/src/components/ui/FormInput.tsx` → prints 1 line (on ShadcnInput)
- [ ] `grep -n "errorId\|hintId" web/src/components/ui/FormInput.tsx` → prints 4 lines (2 derivations + 2 `<p id>` usages)
- [ ] `cd web && bun run type-check` exits 0
- [ ] `cd web && bun run test:run` exits 0; 60 tests pass
- [ ] `cd web && bun run lint` exits 0 errors
- [ ] `git status` shows only `FormInput.tsx` modified
- [ ] `plans/README.md` status row updated to DONE

## STOP conditions

- `bun run type-check` reports that `aria-describedby` with `undefined` value is a
  type error — if so, cast to `string | undefined` explicitly: the type is already
  `string | undefined` in React's `InputHTMLAttributes`, so this should not happen.
- If the `id` prop is typed as `string | undefined` in `InputHTMLAttributes`, the
  `id ? \`${id}-error\` : undefined` pattern is correct TypeScript — no cast needed.
- Do not use `React.useId()` as a fallback — it would require changing the component
  signature (React 18+ hook) and its output changes per-render, causing hydration
  issues in SSR scenarios (even though this app doesn't SSR today). Leaving the IDs
  `undefined` when no `id` is provided is the correct safe default.

## Maintenance notes

- When a new `FormInput` usage is added without an `id` prop, the `aria-describedby`
  association is silently omitted. This is acceptable for now — add a note to the PR
  suggesting that callers pass `id` for fields that show validation errors.
- If `@testing-library/react` is added to the project in the future, add a render
  test: render `<FormInput id="username" error="Required" />` and assert
  `getByRole('textbox').getAttribute('aria-describedby') === 'username-error'`.
