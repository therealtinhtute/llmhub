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

  it('masked region always contains at least 1 star', () => {
    expect(maskApiKey('xy')).toContain('*');
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

  it('formats 1.5 KB', () => {
    expect(formatFileSize(1536)).toBe('1.50 KB');
  });

  it('keeps sub-1024 values in bytes', () => {
    expect(formatFileSize(512)).toBe('512.00 B');
  });
});

describe('formatUnixTimestamp', () => {
  it('returns empty string for null, undefined, or empty string', () => {
    expect(formatUnixTimestamp(null)).toBe('');
    expect(formatUnixTimestamp(undefined)).toBe('');
    expect(formatUnixTimestamp('')).toBe('');
  });

  it('returns empty string for non-finite numbers', () => {
    expect(formatUnixTimestamp(NaN)).toBe('');
    expect(formatUnixTimestamp(Infinity)).toBe('');
  });

  it('returns a non-empty string for a valid 10-digit Unix seconds timestamp', () => {
    // 2024-01-01T00:00:00Z
    const result = formatUnixTimestamp(1704067200);
    expect(typeof result).toBe('string');
    expect(result.length).toBeGreaterThan(0);
  });

  it('returns a non-empty string for a valid 13-digit Unix milliseconds timestamp', () => {
    const result = formatUnixTimestamp(1704067200000);
    expect(typeof result).toBe('string');
    expect(result.length).toBeGreaterThan(0);
  });

  it('handles an ISO 8601 string', () => {
    const result = formatUnixTimestamp('2024-01-01T00:00:00Z');
    expect(typeof result).toBe('string');
    expect(result.length).toBeGreaterThan(0);
  });
});
