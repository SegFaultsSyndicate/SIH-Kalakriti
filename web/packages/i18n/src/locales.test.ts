// packages/i18n/src/locales.test.ts
import { describe, expect, it } from 'vitest';
import { LOCALES, LOCALE_CODES, isLocaleCode, resolveLocale } from './locales';

describe('LOCALES', () => {
  it('lists all 22 Eighth Schedule languages, plus English', () => {
    // English is the administrative link language, not one of the 22
    // constitutionally scheduled languages -- it's the 23rd code, added for
    // the demo and as the catalogue's source of truth.
    expect(LOCALE_CODES).toHaveLength(23);
    expect(LOCALE_CODES).toContain('en');
  });

  it('gives every locale a resolvable Intl tag', () => {
    for (const code of LOCALE_CODES) {
      expect(() => new Intl.NumberFormat(LOCALES[code].numberLocale)).not.toThrow();
    }
  });

  it('exactly en and hi report complete coverage; the rest report fallback', () => {
    for (const code of LOCALE_CODES) {
      const expected = code === 'en' || code === 'hi' ? 'complete' : 'fallback';
      expect(LOCALES[code].coverage).toBe(expected);
    }
  });

  it('flags the three RTL scripts (Kashmiri, Sindhi, Urdu) and nothing else', () => {
    const rtl = LOCALE_CODES.filter((code) => LOCALES[code].dir === 'rtl').sort();
    expect(rtl).toEqual(['ks', 'sd', 'ur']);
  });
});

describe('resolveLocale', () => {
  it('resolves every one of the 22 codes without error', () => {
    for (const code of LOCALE_CODES) {
      expect(resolveLocale([code])).toBe(code);
    }
  });

  it('matches on the primary subtag', () => {
    expect(resolveLocale(['hi-Deva-IN'])).toBe('hi');
  });

  it('falls back to the default for an unknown preference list', () => {
    expect(resolveLocale(['fr-FR', 'de-DE'])).toBe('hi');
  });
});

describe('isLocaleCode', () => {
  it('accepts every scheduled code and rejects junk', () => {
    for (const code of LOCALE_CODES) expect(isLocaleCode(code)).toBe(true);
    expect(isLocaleCode('xx')).toBe(false);
  });
});
