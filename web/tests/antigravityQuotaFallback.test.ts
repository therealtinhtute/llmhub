import { describe, expect, mock, test } from 'bun:test';
import type { TFunction } from 'i18next';
import { buildAntigravityQuotaGroups } from '../src/utils/quota/builders';
import {
  ANTIGRAVITY_FALLBACK_GROUP_LABEL,
  buildAntigravitySummaryGroupsFromModels,
  hasAntigravityGeminiGroup,
} from '../src/utils/quota/antigravityModels';
import {
  parseAntigravityPartialSummary,
  parseAntigravityPayload,
} from '../src/utils/quota/parsers';
import {
  ANTIGRAVITY_BUCKET_LABEL_KEYS,
  ANTIGRAVITY_QUOTA_URLS,
} from '../src/utils/quota/constants';
import en from '../src/i18n/locales/en.json';
import vi from '../src/i18n/locales/vi.json';
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

describe('parseAntigravityPartialSummary', () => {
  test('returns a Claude/GPT-only summary for the loader to keep as fallback', () => {
    const partial = {
      groups: [
        {
          displayName: 'Claude and GPT models',
          buckets: [{ bucketId: '3p-5h', remainingFraction: 1 }],
        },
      ],
    };

    expect(parseAntigravityPartialSummary(partial)).toEqual(partial);
  });

  test('ignores complete summaries, empty groups and model payloads', () => {
    expect(
      parseAntigravityPartialSummary({
        groups: [
          { displayName: 'Gemini models', buckets: [{ bucketId: 'gemini-5h', remainingFraction: 1 }] },
        ],
      })
    ).toBeNull();
    expect(parseAntigravityPartialSummary({ groups: [] })).toBeNull();
    expect(parseAntigravityPartialSummary({})).toBeNull();
    expect(
      parseAntigravityPartialSummary({
        models: {
          'gemini-3.8-flash-high': { quotaInfo: { remainingFraction: 0.4 } },
        },
      })
    ).toBeNull();
  });
});

describe('ANTIGRAVITY_QUOTA_URLS order', () => {
  test('polls the daily host before production, matching the inference executor', () => {
    expect(ANTIGRAVITY_QUOTA_URLS[0]).toBe(
      'https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary'
    );
    const summaryHosts = ANTIGRAVITY_QUOTA_URLS.filter((url) =>
      url.includes('retrieveUserQuotaSummary')
    ).map((url) => new URL(url).host);
    expect(summaryHosts).toEqual([
      'daily-cloudcode-pa.googleapis.com',
      'daily-cloudcode-pa.sandbox.googleapis.com',
      'cloudcode-pa.googleapis.com',
    ]);

    const modelHosts = ANTIGRAVITY_QUOTA_URLS.filter((url) =>
      url.includes('fetchAvailableModels')
    ).map((url) => new URL(url).host);
    expect(modelHosts).toEqual([
      'daily-cloudcode-pa.googleapis.com',
      'daily-cloudcode-pa.sandbox.googleapis.com',
      'cloudcode-pa.googleapis.com',
    ]);
    // fetchAvailableModels stays the last resort after every summary host.
    expect(ANTIGRAVITY_QUOTA_URLS.findIndex((url) => url.includes('fetchAvailableModels'))).toBe(
      summaryHosts.length
    );
  });
});

describe('ANTIGRAVITY_BUCKET_LABEL_KEYS', () => {
  test('resolves upstream window values to keys present in en and vi', () => {
    const enQuota = en.antigravity_quota as Record<string, string>;
    const viQuota = vi.antigravity_quota as Record<string, string>;

    for (const window of ['5h', '5 hour limit', 'daily limit', 'weekly', 'weekly limit']) {
      const key = ANTIGRAVITY_BUCKET_LABEL_KEYS[window];
      expect(key).toBeDefined();
      expect(enQuota[key]).toBeTruthy();
      expect(viQuota[key]).toBeTruthy();
    }

    expect(ANTIGRAVITY_BUCKET_LABEL_KEYS['5h']).toBe('five_hour_limit');
    expect(ANTIGRAVITY_BUCKET_LABEL_KEYS.weekly).toBe('weekly_limit');
  });

  test('renders upstream windows through the same lookup the quota card uses', () => {
    const lookup = (locale: unknown, key: string): unknown =>
      key
        .split('.')
        .reduce<unknown>(
          (node, part) =>
            node && typeof node === 'object'
              ? (node as Record<string, unknown>)[part]
              : undefined,
          locale
        );

    const label = (locale: unknown, window: string) =>
      lookup(locale, `antigravity_quota.${ANTIGRAVITY_BUCKET_LABEL_KEYS[window.toLowerCase()]}`);

    expect(label(en, '5h')).toBe('5-hour limit');
    expect(label(en, 'weekly')).toBe('Weekly limit');
    expect(label(vi, '5h')).toBe('Giới hạn 5 giờ');
    expect(label(vi, 'weekly')).toBe('Giới hạn tuần');
  });
});

type FakeApiResult = {
  statusCode: number;
  header: Record<string, string[]>;
  bodyText: string;
  body: unknown;
};

const requestedUrls: string[] = [];
let respondToRequest: (url: string) => FakeApiResult = () => ({
  statusCode: 500,
  header: {},
  bodyText: '',
  body: null,
});

mock.module('@/services/api', () => ({
  apiCallApi: {
    request: async ({ url }: { url: string }) => {
      requestedUrls.push(url);
      return respondToRequest(url);
    },
  },
  antigravitySubscriptionApi: { get: async () => null },
  authFilesApi: { downloadText: async () => '' },
  getApiCallErrorMessage: (result: FakeApiResult) => `HTTP ${result.statusCode}`,
}));

const { ANTIGRAVITY_CONFIG } = await import('@/components/quota/quotaConfigs');

const summaryPayload = (label: string, remainingFraction: number) => ({
  groups: [
    {
      displayName: label,
      buckets: [{ bucketId: '3p-5h', window: '5h', remainingFraction }],
    },
  ],
});

const fetchQuota = (t: (key: string) => string = (key) => key) =>
  ANTIGRAVITY_CONFIG.fetchQuota(
    {
      name: 'antigravity.json',
      type: 'antigravity',
      auth_index: '1',
      project_id: 'project-1',
    },
    t as unknown as TFunction
  );

describe('fetchAntigravityQuota partial-summary fallback', () => {
  test('renders the first partial summary when every host is partial and the model fallback fails', async () => {
    requestedUrls.length = 0;
    respondToRequest = (url) => {
      if (url.includes('fetchAvailableModels')) {
        return { statusCode: 500, header: {}, bodyText: 'boom', body: null };
      }
      const isFirstSummary = url === ANTIGRAVITY_QUOTA_URLS[0];
      const payload = summaryPayload(
        'Claude and GPT models',
        isFirstSummary ? 1 : 0.5
      );
      return { statusCode: 200, header: {}, bodyText: JSON.stringify(payload), body: payload };
    };

    const data = await fetchQuota();

    expect(requestedUrls).toEqual(ANTIGRAVITY_QUOTA_URLS);
    expect(data.groups.map((group) => group.label)).toEqual(['Claude and GPT models']);
    // The first partial summary wins over later hosts.
    expect(data.groups[0]?.buckets[0]?.remainingFraction).toBe(1);
  });

  test('keeps the first partial summary when the model fallback yields nothing', async () => {
    requestedUrls.length = 0;
    respondToRequest = (url) => {
      if (url.includes('fetchAvailableModels')) {
        const payload = { models: {} };
        return { statusCode: 200, header: {}, bodyText: JSON.stringify(payload), body: payload };
      }
      const payload = summaryPayload('Claude and GPT models', 0.8);
      return { statusCode: 200, header: {}, bodyText: JSON.stringify(payload), body: payload };
    };

    const data = await fetchQuota();

    expect(requestedUrls).toEqual(ANTIGRAVITY_QUOTA_URLS);
    expect(data.groups.map((group) => group.label)).toEqual(['Claude and GPT models']);
  });

  test('still returns the complete summary from the first host without extra requests', async () => {
    requestedUrls.length = 0;
    respondToRequest = () => {
      const payload = {
        groups: [
          { displayName: 'Gemini models', buckets: [{ bucketId: 'gemini-5h', remainingFraction: 0.9 }] },
          { displayName: 'Claude and GPT models', buckets: [{ bucketId: '3p-5h', remainingFraction: 0.7 }] },
        ],
      };
      return { statusCode: 200, header: {}, bodyText: JSON.stringify(payload), body: payload };
    };

    const data = await fetchQuota();

    expect(requestedUrls).toEqual([ANTIGRAVITY_QUOTA_URLS[0]]);
    expect(new URL(requestedUrls[0]!).host).toBe('daily-cloudcode-pa.googleapis.com');
    expect(data.groups.map((group) => group.label)).toEqual([
      'Gemini models',
      'Claude and GPT models',
    ]);
  });

  test('still throws when every host fails', async () => {
    requestedUrls.length = 0;
    respondToRequest = () => ({ statusCode: 403, header: {}, bodyText: 'forbidden', body: null });

    await expect(fetchQuota()).rejects.toThrow();
    expect(requestedUrls).toEqual(ANTIGRAVITY_QUOTA_URLS);
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
