// packages/i18n/src/format.test.ts
import { describe, expect, it } from 'vitest';
import { formatMoney, formatMoneyRange } from './format';

describe('formatMoney', () => {
  it('never renders a raw paise integer', () => {
    expect(formatMoney(45000, 'en')).toBe('₹450');
  });

  it('groups by lakh, not by thousand', () => {
    expect(formatMoney(1234567800, 'en')).toBe('₹1,23,45,678');
  });

  it('keeps the paise remainder exact where float division would not', () => {
    // 123456 / 100 is 1234.56 in float, but 45699 / 100 is 456.99000000000001.
    expect(formatMoney(45699, 'en')).toBe('₹456.99');
    expect(formatMoney(1, 'en')).toBe('₹0.01');
    expect(formatMoney(10, 'en')).toBe('₹0.10');
  });

  it('hides a zero remainder by default and shows it on request', () => {
    expect(formatMoney(45000, 'en')).toBe('₹450');
    expect(formatMoney(45000, 'en', { paise: 'always' })).toBe('₹450.00');
    expect(formatMoney(45699, 'en', { paise: 'never' })).toBe('₹456');
  });

  it('handles negative amounts (refunds, statement debits)', () => {
    expect(formatMoney(-45699, 'en')).toBe('-₹456.99');
  });

  it('rejects a non-integer, which can only mean float math upstream', () => {
    expect(() => formatMoney(456.99, 'en')).toThrow(TypeError);
  });

  it('formats a price band as one range', () => {
    expect(formatMoneyRange(120000, 180000, 'en')).toBe('₹1,200–1,800');
  });
});
