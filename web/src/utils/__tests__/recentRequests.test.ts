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

  it('caps output at 20 entries', () => {
    const input = Array.from({ length: 25 }, (_, i) => ({ success: i, failed: 0 }));
    expect(normalizeRecentRequestBuckets(input)).toHaveLength(20);
  });
});

describe('mergeRecentRequestBucketGroups', () => {
  it('returns empty array when all groups are empty', () => {
    expect(mergeRecentRequestBucketGroups([[], []])).toEqual([]);
  });

  it('sums overlapping single buckets from two groups', () => {
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
    const { blocks } = statusBarDataFromRecentRequests([{ success: 5, failed: 0 }]);
    expect(blocks[blocks.length - 1]).toBe('success');
  });

  it('classifies failure-only bucket', () => {
    const { blocks } = statusBarDataFromRecentRequests([{ success: 0, failed: 3 }]);
    expect(blocks[blocks.length - 1]).toBe('failure');
  });

  it('classifies mixed bucket and computes successRate', () => {
    const result = statusBarDataFromRecentRequests([{ success: 2, failed: 2 }]);
    expect(result.blocks[result.blocks.length - 1]).toBe('mixed');
    expect(result.successRate).toBe(50);
  });

  it('blockDetails always has 20 entries', () => {
    const result = statusBarDataFromRecentRequests([{ success: 1, failed: 0 }]);
    expect(result.blockDetails).toHaveLength(20);
  });
});

describe('buildRecentRequestCompositeKey', () => {
  it('joins base URL and api key with a pipe', () => {
    expect(buildRecentRequestCompositeKey('http://localhost', 'key')).toBe(
      'http://localhost|key',
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

  it('returns trimmed string for string input', () => {
    expect(normalizeRecentRequestAuthIndex('  abc  ')).toBe('abc');
  });

  it('returns null for empty or whitespace-only string', () => {
    expect(normalizeRecentRequestAuthIndex('')).toBeNull();
    expect(normalizeRecentRequestAuthIndex('   ')).toBeNull();
  });

  it('returns null for non-string non-number types', () => {
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
