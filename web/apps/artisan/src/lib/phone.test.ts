// apps/artisan/src/lib/phone.test.ts
import { describe, expect, it } from 'vitest';
import { digitsFromTranscript, isValidIndianMobile, toE164 } from './phone';

describe('isValidIndianMobile', () => {
  it('accepts a 10-digit number starting 6-9', () => {
    expect(isValidIndianMobile('9876543210')).toBe(true);
    expect(isValidIndianMobile('6000000000')).toBe(true);
  });

  it('rejects wrong length', () => {
    expect(isValidIndianMobile('987654321')).toBe(false);
    expect(isValidIndianMobile('98765432100')).toBe(false);
  });

  it('rejects a first digit outside 6-9', () => {
    expect(isValidIndianMobile('5876543210')).toBe(false);
    expect(isValidIndianMobile('0876543210')).toBe(false);
  });
});

describe('toE164', () => {
  it('prefixes +91', () => {
    expect(toE164('9876543210')).toBe('+919876543210');
  });
});

describe('digitsFromTranscript', () => {
  it('strips non-digits and caps at 10', () => {
    expect(digitsFromTranscript('98765 43210')).toBe('9876543210');
    expect(digitsFromTranscript('+91 98765-43210')).toBe('9198765432');
  });

  it('returns empty for a transcript with no digits', () => {
    expect(digitsFromTranscript('hello')).toBe('');
  });
});
