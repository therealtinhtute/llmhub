import { describe, expect, test } from 'bun:test';
import { buildAntigravityQuotaGroups } from '../src/utils/quota/builders';
import {
  ANTIGRAVITY_FALLBACK_GROUP_LABEL,
  buildAntigravitySummaryGroupsFromModels,
  hasAntigravityGeminiGroup,
} from '../src/utils/quota/antigravityModels';
import { parseAntigravityPayload } from '../src/utils/quota/parsers';
import type { AntigravityQuotaSummaryPayload } from '../src/types/quota';

type SummaryGroup = {
  displayName?: string;
  buckets?: Array<{ bucketId?: string; remainingFraction?: number | string }>;
};

const readGroups = (payload: Record<string, unknown> | null): SummaryGroup[] =>
  (Array.isArray(payload?.groups) ? payload.groups : []) as SummaryGroup[];

describe('buildAntigravitySummaryGroupsFromModels', () => {
  test('maps known model IDs to their existing groups', () => {
    const groups = buildAntigravitySummaryGroupsFromModels({
      'claude-sonnet-4-6': {
        displayName: 'Claude Sonnet',
        quotaInfo: { remainingFraction: 0.9 },
      },
      'gemini-3-flash': {
        displayName: 'Gemini 3 Flash',
        quotaInfo: { remainingFraction: 0.75 },
      },
    });

    expect(groups.map((group) => group.displayName)).toEqual(['Claude/GPT', 'Gemini 3 Flash']);
  });

  test('collects unmatched gemini-* models into the fallback group', () => {
    const groups = buildAntigravitySummaryGroupsFromModels({
      'gemini-3-flash': {
        displayName: 'Gemini 3 Flash',
        quotaInfo: { remainingFraction: 0.75 },
      },
      'gemini-3.8-flash-high': {
        displayName: 'Gemini 3.8 Flash',
        quotaInfo: { remainingFraction: 0.5 },
      },
      'gemini-4-argon-pro': {
        displayName: 'Gemini 4 Argon',
        quotaInfo: { remainingFraction: 0.2 },
      },
    });

    expect(groups.map((group) => group.displayName)).toEqual([
      'Gemini 3 Flash',
      ANTIGRAVITY_FALLBACK_GROUP_LABEL,
    ]);
    const fallback = groups.find((group) => group.displayName === ANTIGRAVITY_FALLBACK_GROUP_LABEL);
    expect(fallback?.buckets?.map((bucket) => bucket.bucketId)).toEqual([
      'gemini-3.8-flash-high',
      'gemini-4-argon-pro',
    ]);
  });

  test('keeps an exhausted model instead of dropping it', () => {
    const groups = buildAntigravitySummaryGroupsFromModels({
      'gemini-3.9-flash': {
        displayName: 'Gemini 3.9 Flash',
        quotaInfo: { resetTime: '2026-10-20T00:00:00Z' },
      },
    });

    const fallback = groups.find((group) => group.displayName === ANTIGRAVITY_FALLBACK_GROUP_LABEL);
    expect(fallback?.buckets?.[0]?.remainingFraction).toBe(0);
  });
});

describe('parseAntigravityPayload', () => {
  test('synthesizes groups from a fetchAvailableModels payload', () => {
    const parsed = parseAntigravityPayload({
      models: {
        'gemini-3.8-flash-high': {
          displayName: 'Gemini 3.8 Flash',
          quotaInfo: { remainingFraction: 0.4 },
        },
      },
    });

    const groups = readGroups(parsed);
    expect(groups.map((group) => group.displayName)).toEqual([ANTIGRAVITY_FALLBACK_GROUP_LABEL]);
  });

  test('rejects a summary without any Gemini group so the loader tries the next host', () => {
    const parsed = parseAntigravityPayload({
      groups: [
        {
          displayName: 'Claude and GPT models',
          buckets: [{ bucketId: '3p-5h', remainingFraction: 1 }],
        },
      ],
    });

    expect(parsed).toBeNull();
  });

  test('renders an unknown gemini-* model through the group builder', () => {
    const parsed = parseAntigravityPayload({
      models: {
        'gemini-3.8-flash-high': {
          displayName: 'Gemini 3.8 Flash',
          quotaInfo: { remainingFraction: 0.5, resetTime: '2026-10-20T00:00:00Z' },
        },
      },
    }) as AntigravityQuotaSummaryPayload | null;

    const groups = buildAntigravityQuotaGroups(parsed ?? {});
    const fallback = groups.find((group) => group.label === ANTIGRAVITY_FALLBACK_GROUP_LABEL);

    expect(fallback?.id).toBe('gemini-other');
    expect(fallback?.buckets[0]?.id).toBe('gemini-3.8-flash-high');
    expect(fallback?.buckets[0]?.remainingFraction).toBe(0.5);
  });

  test('keeps a complete summary payload unchanged', () => {
    const payload = {
      groups: [
        {
          displayName: 'Gemini models',
          buckets: [{ bucketId: 'gemini-5h', remainingFraction: 1 }],
        },
        {
          displayName: 'Claude and GPT models',
          buckets: [{ bucketId: '3p-5h', remainingFraction: 1 }],
        },
      ],
    };

    expect(parseAntigravityPayload(payload)).toEqual(payload);
  });
});

describe('hasAntigravityGeminiGroup', () => {
  test('detects Gemini groups through display names and bucket IDs', () => {
    expect(hasAntigravityGeminiGroup([{ displayName: 'Gemini 4 Argon' }])).toBe(true);
    expect(
      hasAntigravityGeminiGroup([{ displayName: 'Claude and GPT models', buckets: [{ bucketId: 'gemini-weekly' }] }])
    ).toBe(true);
    expect(
      hasAntigravityGeminiGroup([
        { displayName: 'Claude and GPT models', buckets: [{ bucketId: '3p-weekly' }] },
      ])
    ).toBe(false);
  });
});
