# Plan 002: Establish Vitest test infrastructure for web utility modules

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md`.
>
> **Drift check (run first)**:
> `git diff --stat 5f81cee..HEAD -- web/package.json web/src/utils/encryption.ts web/src/utils/recentRequests.ts web/src/utils/validation.ts web/src/utils/format.ts`
> If any in-scope utility changed since this plan was written, compare the
> "Current state" excerpts before proceeding; on a mismatch treat it as a
> STOP condition.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW
- **Depends on**: none (independent of plan 001)
- **Category**: tests
- **Planned at**: commit `5f81cee`, 2026-06-11

## Why this matters

The web panel has zero automated tests. Pure utility functions in `src/utils/`
— including the obfuscation layer that stores the management key, a
request-counting module used in every quota display, and input-validation
helpers — can regress silently. This plan installs Vitest, wires it into the
existing bun/vite workspace, and writes an initial test suite for the four
most stable utility modules. It is intentionally scoped to utilities only;
React component tests are deferred so this delivers fast, reliable baseline
coverage without the complexity of a full React testing setup.

## Current state

- `web/package.json` — no `test` script, no vitest dependency.
- `web/vite.config.ts` — production build config; uses `viteSingleFile` plugin
  which would interfere with test collection if vitest reused it. A separate
  `vitest.config.ts` avoids this coupling.
- `web/tsconfig.json` — `"include": ["src"]` already covers any `__tests__`
  subdirectories inside `src/`.

Target utilities (all in `web/src/utils/`):

| File | What it does | Test concern |
|------|-------------|--------------|
| `encryption.ts` | XOR-obfuscates values for localStorage | Uses `window`, `navigator`, `TextEncoder/Decoder`, `btoa/atob` — needs jsdom |
| `recentRequests.ts` | Normalises/merges API-call bucket data | Uses `Date.now()` in one function; rest is pure |
| `validation.ts` | ASCII charset check for API keys | Pure regex |
| `format.ts` | Masks keys, formats file sizes, formats timestamps | Pure arithmetic + `toLocaleString` |

The `@/` path alias (`src/`) is configured in `vite.config.ts:resolve.alias`.
The vitest config must mirror it so imports inside utilities resolve.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Install deps | `cd web && bun add -d vitest jsdom` | exit 0 |
| Run tests | `cd web && bun run test:run` | all pass, 0 failures |
| Type-check | `cd web && bun run type-check` | exit 0 |
| Lint | `cd web && bun run lint` | exit 0 |

## Scope

**In scope** (the only files you should create or modify):
- `web/vitest.config.ts` (create)
- `web/package.json` — add scripts and devDependencies only
- `web/src/utils/__tests__/encryption.test.ts` (create)
- `web/src/utils/__tests__/recentRequests.test.ts` (create)
- `web/src/utils/__tests__/validation.test.ts` (create)
- `web/src/utils/__tests__/format.test.ts` (create)

**Out of scope** (do NOT touch):
- `web/vite.config.ts` — production build must not change.
- Any `src/` file that is not a new test file — do not modify utility source.
- React component tests — deferred.
- `web/tsconfig.json` — current `"include": ["src"]` already covers test files.

## Git workflow

- Branch: `advisor/002-web-test-infra`
- Commit per logical unit: e.g. `test(web): add vitest config and utility test suite`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Install Vitest and jsdom

```bash
cd web
bun add -d vitest jsdom
```

**Verify**: `grep '"vitest"' web/package.json` → prints a version line under `devDependencies`.

### Step 2: Create `web/vitest.config.ts`

Create this file exactly:

```ts
import { defineConfig } from 'vitest/config';
import path from 'path';

export default defineConfig({
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    environment: 'jsdom',
    globals: false,
  },
});
```

Rationale:
- `resolve.alias` mirrors `vite.config.ts` so `@/utils/...` imports resolve in tests.
- `environment: 'jsdom'` provides `window`, `navigator`, `TextEncoder`, `btoa`, etc. for `encryption.ts`.
- `globals: false` — tests import `describe`/`it`/`expect` from `vitest` explicitly; avoids polluting global types without a separate `tsconfig` override.

**Verify**: `ls web/vitest.config.ts` → file exists.

### Step 3: Add test scripts to `web/package.json`

Add to the `"scripts"` object:
```json
"test": "vitest",
"test:run": "vitest run"
```

Do not remove or modify any existing script.

**Verify**: `grep '"test:run"' web/package.json` → prints the new line.

### Step 4: Write `src/utils/__tests__/validation.test.ts`

```ts
import { describe, it, expect } from 'vitest';
import { isValidApiKeyCharset } from '@/utils/validation';

describe('isValidApiKeyCharset', () => {
  it('returns false for empty string', () => {
    expect(isValidApiKeyCharset('')).toBe(false);
  });

  it('returns true for printable ASCII without spaces', () => {
    expect(isValidApiKeyCharset('sk-abc123!@#')).toBe(true);
    expect(isValidApiKeyCharset('ABC123xyz')).toBe(true);
  });

  it('returns false for a plain space (0x20)', () => {
    expect(isValidApiKeyCharset('has space')).toBe(false);
  });

  it('returns false for a newline character', () => {
    expect(isValidApiKeyCharset('has\nnewline')).toBe(false);
  });

  it('returns false for non-ASCII characters', () => {
    expect(isValidApiKeyCharset('café')).toBe(false);
    expect(isValidApiKeyCharset('你好')).toBe(false);
  });

  it('accepts tilde (0x7E), the highest valid character', () => {
    expect(isValidApiKeyCharset('~')).toBe(true);
  });

  it('accepts exclamation mark (0x21), the lowest valid character', () => {
    expect(isValidApiKeyCharset('!')).toBe(true);
  });
});
```

**Verify**: `cd web && bun run test:run src/utils/__tests__/validation.test.ts` → 7 tests pass.

### Step 5: Write `src/utils/__tests__/encryption.test.ts`

```ts
import { describe, it, expect } from 'vitest';
import {
  obfuscateData,
  deobfuscateData,
  isObfuscated,
  encryptData,
  decryptData,
  isEncrypted,
} from '@/utils/encryption';

const ENC_PREFIX = 'enc::v1::';

describe('obfuscateData', () => {
  it('adds the enc::v1:: prefix', () => {
    expect(obfuscateData('hello')).toMatch(new RegExp(`^${ENC_PREFIX}`));
  });

  it('returns empty string unchanged', () => {
    expect(obfuscateData('')).toBe('');
  });

  it('produces a different string than the input', () => {
    const input = 'my-management-key';
    const result = obfuscateData(input);
    expect(result).not.toBe(input);
  });
});

describe('deobfuscateData', () => {
  it('round-trips: deobfuscate(obfuscate(x)) === x', () => {
    const cases = ['hello', 'sk-abc123', 'http://localhost:9090', '{}', 'a'];
    for (const input of cases) {
      expect(deobfuscateData(obfuscateData(input))).toBe(input);
    }
  });

  it('passes through a plain string that has no prefix', () => {
    expect(deobfuscateData('plaintext')).toBe('plaintext');
  });

  it('passes through an empty string', () => {
    expect(deobfuscateData('')).toBe('');
  });
});

describe('isObfuscated', () => {
  it('returns true for an obfuscated value', () => {
    expect(isObfuscated(obfuscateData('x'))).toBe(true);
  });

  it('returns false for a plain string', () => {
    expect(isObfuscated('plain')).toBe(false);
  });

  it('returns false for empty string', () => {
    expect(isObfuscated('')).toBe(false);
  });
});

describe('backward-compat aliases', () => {
  it('encryptData and decryptData are the same functions', () => {
    const val = 'alias-test';
    expect(decryptData(encryptData(val))).toBe(val);
  });

  it('isEncrypted matches isObfuscated', () => {
    const enc = obfuscateData('x');
    expect(isEncrypted(enc)).toBe(isObfuscated(enc));
  });
});
```

**Verify**: `cd web && bun run test:run src/utils/__tests__/encryption.test.ts` → all tests pass.

### Step 6: Write `src/utils/__tests__/recentRequests.test.ts`

```ts
import { describe, it, expect } from 'vitest';
import {
  normalizeUsageTotal,
  normalizeRecentRequestBuckets,
  mergeRecentRequestBucketGroups,
  sumRecentRequests,
  statusBarDataFromRecentRequests,
  buildRecentRequestCompositeKey,
  normalizeRecentRequestAuthIndex,
  normalizeRecentRequestUsageEntry,
} from '@/utils/recentRequests';

describe('normalizeUsageTotal', () => {
  it('returns finite numbers as-is', () => {
    expect(normalizeUsageTotal(5)).toBe(5);
    expect(normalizeUsageTotal(0)).toBe(0);
  });

  it('coerces numeric strings', () => {
    expect(normalizeUsageTotal('42')).toBe(42);
    expect(normalizeUsageTotal('  7  ')).toBe(7);
  });

  it('returns 0 for non-numeric / infinite values', () => {
    expect(normalizeUsageTotal(Infinity)).toBe(0);
    expect(normalizeUsageTotal('abc')).toBe(0);
    expect(normalizeUsageTotal(null)).toBe(0);
    expect(normalizeUsageTotal(undefined)).toBe(0);
    expect(normalizeUsageTotal('')).toBe(0);
  });
});

describe('normalizeRecentRequestBuckets', () => {
  it('returns empty array for non-array input', () => {
    expect(normalizeRecentRequestBuckets(null)).toEqual([]);
    expect(normalizeRecentRequestBuckets('str')).toEqual([]);
    expect(normalizeRecentRequestBuckets(42)).toEqual([]);
  });

  it('normalises string fields to numbers', () => {
    const result = normalizeRecentRequestBuckets([{ success: '5', failed: '3' }]);
    expect(result).toEqual([{ success: 5, failed: 3 }]);
  });

  it('preserves the time field when present', () => {
    const result = normalizeRecentRequestBuckets([
      { success: 1, failed: 0, time: '2024-01-01T00:00:00Z' },
    ]);
    expect(result[0].time).toBe('2024-01-01T00:00:00Z');
  });

  it('caps output at 20 entries (RECENT_REQUEST_BLOCK_COUNT)', () => {
    const input = Array.from({ length: 25 }, (_, i) => ({ success: i, failed: 0 }));
    expect(normalizeRecentRequestBuckets(input)).toHaveLength(20);
  });
});

describe('mergeRecentRequestBucketGroups', () => {
  it('returns empty array when all groups are empty', () => {
    expect(mergeRecentRequestBucketGroups([[], []])).toEqual([]);
  });

  it('sums overlapping single buckets', () => {
    const a = [{ success: 3, failed: 1 }];
    const b = [{ success: 2, failed: 2 }];
    const merged = mergeRecentRequestBucketGroups([a, b]);
    expect(merged).toEqual([{ success: 5, failed: 3 }]);
  });

  it('passes through a single group unchanged', () => {
    const group = [{ success: 1, failed: 0 }, { success: 2, failed: 1 }];
    expect(mergeRecentRequestBucketGroups([group])).toEqual(group);
  });
});

describe('sumRecentRequests', () => {
  it('totals success and failure across buckets', () => {
    const buckets = [
      { success: 3, failed: 2 },
      { success: 1, failed: 0 },
    ];
    expect(sumRecentRequests(buckets)).toEqual({ success: 4, failure: 2 });
  });

  it('returns zeros for empty input', () => {
    expect(sumRecentRequests([])).toEqual({ success: 0, failure: 0 });
  });
});

describe('statusBarDataFromRecentRequests', () => {
  it('returns 20 idle blocks for empty input', () => {
    const result = statusBarDataFromRecentRequests([]);
    expect(result.blocks).toHaveLength(20);
    expect(result.blocks.every((b) => b === 'idle')).toBe(true);
    expect(result.successRate).toBe(100);
    expect(result.totalSuccess).toBe(0);
    expect(result.totalFailure).toBe(0);
  });

  it('classifies success-only bucket', () => {
    const result = statusBarDataFromRecentRequests([{ success: 5, failed: 0 }]);
    expect(result.blocks.at(-1)).toBe('success');
  });

  it('classifies failure-only bucket', () => {
    const result = statusBarDataFromRecentRequests([{ success: 0, failed: 3 }]);
    expect(result.blocks.at(-1)).toBe('failure');
  });

  it('classifies mixed bucket', () => {
    const result = statusBarDataFromRecentRequests([{ success: 2, failed: 2 }]);
    expect(result.blocks.at(-1)).toBe('mixed');
    expect(result.successRate).toBe(50);
  });

  it('blockDetails has 20 entries', () => {
    const result = statusBarDataFromRecentRequests([{ success: 1, failed: 0 }]);
    expect(result.blockDetails).toHaveLength(20);
  });
});

describe('buildRecentRequestCompositeKey', () => {
  it('joins base URL and api key with pipe', () => {
    expect(buildRecentRequestCompositeKey('http://localhost', 'key')).toBe(
      'http://localhost|key'
    );
  });

  it('handles null/undefined gracefully', () => {
    expect(buildRecentRequestCompositeKey(null, undefined)).toBe('|');
  });
});

describe('normalizeRecentRequestAuthIndex', () => {
  it('returns string for a numeric index', () => {
    expect(normalizeRecentRequestAuthIndex(3)).toBe('3');
  });

  it('returns trimmed string', () => {
    expect(normalizeRecentRequestAuthIndex('  abc  ')).toBe('abc');
  });

  it('returns null for empty string', () => {
    expect(normalizeRecentRequestAuthIndex('')).toBeNull();
    expect(normalizeRecentRequestAuthIndex('   ')).toBeNull();
  });

  it('returns null for non-string non-number', () => {
    expect(normalizeRecentRequestAuthIndex(null)).toBeNull();
    expect(normalizeRecentRequestAuthIndex({})).toBeNull();
  });
});

describe('normalizeRecentRequestUsageEntry', () => {
  it('returns zero entry for non-object input', () => {
    expect(normalizeRecentRequestUsageEntry(null)).toEqual({
      success: 0,
      failed: 0,
      recentRequests: [],
    });
  });

  it('normalises success and failed from object', () => {
    const result = normalizeRecentRequestUsageEntry({ success: '10', failed: 2 });
    expect(result.success).toBe(10);
    expect(result.failed).toBe(2);
  });

  it('accepts both recent_requests and recentRequests keys', () => {
    const a = normalizeRecentRequestUsageEntry({
      success: 1,
      failed: 0,
      recent_requests: [{ success: 1, failed: 0 }],
    });
    const b = normalizeRecentRequestUsageEntry({
      success: 1,
      failed: 0,
      recentRequests: [{ success: 1, failed: 0 }],
    });
    expect(a.recentRequests).toHaveLength(1);
    expect(b.recentRequests).toHaveLength(1);
  });
});
```

**Verify**: `cd web && bun run test:run src/utils/__tests__/recentRequests.test.ts` → all tests pass.

### Step 7: Write `src/utils/__tests__/format.test.ts`

```ts
import { describe, it, expect } from 'vitest';
import { maskApiKey, formatFileSize, formatUnixTimestamp } from '@/utils/format';

describe('maskApiKey', () => {
  it('returns empty string for empty input', () => {
    expect(maskApiKey('')).toBe('');
  });

  it('returns empty string for whitespace-only input', () => {
    expect(maskApiKey('   ')).toBe('');
  });

  it('preserves 2 visible chars at start and end for keys >= 4 chars', () => {
    const result = maskApiKey('sk-abcdef');
    expect(result.startsWith('sk')).toBe(true);
    expect(result.endsWith('ef')).toBe(true);
    expect(result).toContain('*');
  });

  it('uses 1 visible char at each end for short keys (< 4 chars)', () => {
    const result = maskApiKey('ab');
    expect(result.startsWith('a')).toBe(true);
    expect(result.endsWith('b')).toBe(true);
  });

  it('masked region is always at least 1 star', () => {
    const result = maskApiKey('xy');
    expect(result).toContain('*');
  });
});

describe('formatFileSize', () => {
  it('formats 0 bytes', () => {
    expect(formatFileSize(0)).toBe('0 B');
  });

  it('formats exactly 1 KB', () => {
    expect(formatFileSize(1024)).toBe('1.00 KB');
  });

  it('formats exactly 1 MB', () => {
    expect(formatFileSize(1024 * 1024)).toBe('1.00 MB');
  });

  it('formats exactly 1 GB', () => {
    expect(formatFileSize(1024 ** 3)).toBe('1.00 GB');
  });

  it('formats fractional KB', () => {
    expect(formatFileSize(512)).toBe('0.50 KB');
  });
});

describe('formatUnixTimestamp', () => {
  it('returns empty string for null, undefined, or empty', () => {
    expect(formatUnixTimestamp(null)).toBe('');
    expect(formatUnixTimestamp(undefined)).toBe('');
    expect(formatUnixTimestamp('')).toBe('');
  });

  it('returns empty string for non-finite numbers', () => {
    expect(formatUnixTimestamp(NaN)).toBe('');
    expect(formatUnixTimestamp(Infinity)).toBe('');
  });

  it('returns a non-empty string for a valid Unix seconds timestamp (10-digit)', () => {
    // 2024-01-01T00:00:00Z in seconds
    const result = formatUnixTimestamp(1704067200);
    expect(typeof result).toBe('string');
    expect(result.length).toBeGreaterThan(0);
  });

  it('returns a non-empty string for a valid Unix milliseconds timestamp (13-digit)', () => {
    const result = formatUnixTimestamp(1704067200000);
    expect(typeof result).toBe('string');
    expect(result.length).toBeGreaterThan(0);
  });

  it('handles an ISO string', () => {
    const result = formatUnixTimestamp('2024-01-01T00:00:00Z');
    expect(typeof result).toBe('string');
    expect(result.length).toBeGreaterThan(0);
  });
});
```

**Verify**: `cd web && bun run test:run src/utils/__tests__/format.test.ts` → all tests pass.

### Step 8: Run the full suite

```bash
cd web && bun run test:run
```

**Verify**: all tests pass, exit code 0.

### Step 9: Type-check and lint

**Verify**: `cd web && bun run type-check` → exits 0.
**Verify**: `cd web && bun run lint` → exits 0 errors (pre-existing warnings are acceptable).

## Test plan

Covered in steps 4–7. The test files serve as both specification and regression guard:
- `validation.test.ts` — 7 cases covering boundary chars and unicode
- `encryption.test.ts` — 11 cases covering round-trip, empty input, aliases
- `recentRequests.test.ts` — 24 cases covering all exported functions
- `format.test.ts` — 13 cases covering masking, size formatting, timestamp parsing

No prior test file to model after (first tests in this repo).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep '"vitest"' web/package.json` prints a version under `devDependencies`
- [ ] `ls web/vitest.config.ts` exits 0
- [ ] `grep '"test:run"' web/package.json` prints the new script line
- [ ] `cd web && bun run test:run` exits 0; prints "X tests passed"
- [ ] `cd web && bun run type-check` exits 0
- [ ] `cd web && bun run lint` exits 0 errors
- [ ] `git status` shows only the in-scope files modified/created
- [ ] `plans/README.md` status row updated to DONE

## STOP conditions

Stop and report back (do not improvise) if:

- Any utility source file has drifted from the plan's understanding (drift check above).
- `bun run test:run` reports failures that cannot be fixed without modifying the utility source (that would be out of scope — report instead).
- `bun run type-check` introduces new errors (not the 8 pre-existing warnings in `sidebar.tsx`).
- Installing vitest conflicts with an existing package and resolution is not obvious.

## Maintenance notes

- When new utility functions are added to `src/utils/`, the convention is to add
  a `__tests__` entry alongside them in the same directory.
- `encryption.ts` has a module-level `cachedKeyBytes` that persists across tests
  within a single run. Tests that depend on specific key material (e.g. testing
  the `window.location` fallback) would need `vi.resetModules()` to clear it.
  The current tests deliberately avoid this by testing round-trip only.
- `formatUnixTimestamp` uses `toLocaleString()` without a fixed locale, so exact
  output varies by environment. Tests assert structure (non-empty string), not
  exact formatted text. If locale-sensitive formatting becomes critical, pass a
  fixed locale (e.g. `'en-US'`) in assertions.
- `statusBarDataFromRecentRequests` uses `Date.now()` for block timestamps. The
  tests here do not assert exact timestamps; if timestamp precision matters in
  future, use `vi.useFakeTimers()` to freeze time.
