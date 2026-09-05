// packages/i18n/src/format-message.test.ts
import { describe, expect, it } from 'vitest';
import { interpolate, selectPluralKey } from './format-message';

describe('interpolate', () => {
  it('substitutes known placeholders', () => {
    expect(interpolate('{count} items', { count: 3 })).toBe('3 items');
  });

  it('leaves an unknown placeholder untouched', () => {
    expect(interpolate('{missing} items', {})).toBe('{missing} items');
  });

  it('is a no-op with no values', () => {
    expect(interpolate('plain text')).toBe('plain text');
  });
});

describe('selectPluralKey', () => {
  const always = () => true;
  const never = () => false;

  it('picks the "one" category for English count 1', () => {
    expect(selectPluralKey('x', 1, 'en-IN', always)).toBe('x.one');
  });

  it('picks the "other" category for English count 5', () => {
    expect(selectPluralKey('x', 5, 'en-IN', always)).toBe('x.other');
  });

  it('falls back to base.other when the selected category key does not exist', () => {
    expect(selectPluralKey('x', 1, 'en-IN', never)).toBe('x.other');
  });

  it('resolves a language with more than two plural categories (Arabic zero/one/two/few/many/other)', () => {
    expect(selectPluralKey('x', 0, 'ar', always)).toBe('x.zero');
    expect(selectPluralKey('x', 2, 'ar', always)).toBe('x.two');
    expect(selectPluralKey('x', 11, 'ar', always)).toBe('x.many');
  });
});
