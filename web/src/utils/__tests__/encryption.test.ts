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

  it('passes through a plain string with no enc prefix', () => {
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
  it('encryptData / decryptData round-trip', () => {
    const val = 'alias-test';
    expect(decryptData(encryptData(val))).toBe(val);
  });

  it('isEncrypted agrees with isObfuscated', () => {
    const enc = obfuscateData('x');
    expect(isEncrypted(enc)).toBe(isObfuscated(enc));
  });
});
