import { describe, expect, it } from 'vitest';
import { CODEX_CONFIG } from '@/components/quota/quotaConfigs';
import { normalizeCodexResetCreditsPayload } from '@/utils/quota';

describe('normalizeCodexResetCreditsPayload', () => {
  it('normalizes available Codex credits and filters unusable entries', () => {
    const result = normalizeCodexResetCreditsPayload({
      available_count: '2',
      credits: [
        {
          id: 'credit-1',
          reset_type: 'codex_rate_limits',
          status: 'available',
          granted_at: '2026-07-01T00:00:00Z',
          expires_at: '2026-08-01T00:00:00Z',
        },
        {
          id: 'credit-2',
          resetType: 'codex_rate_limits',
          status: 'available',
          grantedAt: '2026-07-02T00:00:00Z',
          expiresAt: '2026-08-02T00:00:00Z',
        },
        {
          id: 'used-credit',
          reset_type: 'codex_rate_limits',
          status: 'consumed',
          expires_at: '2026-08-03T00:00:00Z',
        },
        {
          id: 'other-reset',
          reset_type: 'other',
          status: 'available',
          expires_at: '2026-08-04T00:00:00Z',
        },
      ],
    });

    expect(result).toEqual({
      availableCount: 2,
      credits: [
        {
          id: 'credit-1',
          status: 'available',
          grantedAt: '2026-07-01T00:00:00Z',
          expiresAt: '2026-08-01T00:00:00Z',
        },
        {
          id: 'credit-2',
          status: 'available',
          grantedAt: '2026-07-02T00:00:00Z',
          expiresAt: '2026-08-02T00:00:00Z',
        },
      ],
      invalidPayload: false,
    });
  });

  it('accepts JSON strings and camelCase counts', () => {
    expect(
      normalizeCodexResetCreditsPayload(JSON.stringify({ availableCount: 1, credits: [] }))
    ).toEqual({ availableCount: 1, credits: [], invalidPayload: false });
  });

  it('marks malformed or unexpected payloads invalid', () => {
    expect(normalizeCodexResetCreditsPayload('not-json').invalidPayload).toBe(true);
    expect(normalizeCodexResetCreditsPayload({ message: 'ok' }).invalidPayload).toBe(true);
    expect(normalizeCodexResetCreditsPayload(null).invalidPayload).toBe(true);
  });
});

describe('CODEX_CONFIG.canResetQuota', () => {
  it('only allows reset when the provider reports an available credit', () => {
    expect(
      CODEX_CONFIG.canResetQuota?.({
        status: 'success',
        windows: [],
        rateLimitResetCreditsAvailableCount: 1,
      })
    ).toBe(true);
    expect(
      CODEX_CONFIG.canResetQuota?.({
        status: 'success',
        windows: [],
        rateLimitResetCreditsAvailableCount: 0,
      })
    ).toBe(false);
    expect(CODEX_CONFIG.canResetQuota?.({ status: 'success', windows: [] })).toBe(false);
  });
});
