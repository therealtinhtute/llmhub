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
