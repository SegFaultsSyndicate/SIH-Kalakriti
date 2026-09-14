// packages/i18n/src/locales.test.ts
import { describe, expect, it } from 'vitest';
import { LOCALES, LOCALE_CODES, SUPPORTED_LOCALES, isLocaleCode, resolveLocale } from './locales';

describe('LOCALES', () => {
  it('lists 20 of the 22 Eighth Schedule languages, plus English', () => {
    // English is the administrative link language, not one of the scheduled
    // languages -- it's the 21st code, added for the demo and as the
    // catalogue's source of truth. Manipuri and Santali were dropped (weakest
    // script/font support of the 22, no translated catalogue).
    expect(LOCALE_CODES).toHaveLength(21);
    expect(LOCALE_CODES).toContain('en');
    expect(LOCALE_CODES).not.toContain('mni');
    expect(LOCALE_CODES).not.toContain('sat');
    expect(SUPPORTED_LOCALES).toHaveLength(20);
    expect(SUPPORTED_LOCALES).not.toContain('en');
  });

  it('gives every locale a resolvable Intl tag', () => {
    for (const code of LOCALE_CODES) {
      expect(() => new Intl.NumberFormat(LOCALES[code].numberLocale)).not.toThrow();
    }
  });

  it('reports complete coverage for translated languages; the rest report fallback', () => {
    const complete = new Set(['en', 'hi', 'bn', 'gu', 'mr', 'or', 'pa', 'sd', 'ta', 'te', 'ur']);
    for (const code of LOCALE_CODES) {
      const expected = complete.has(code) ? 'complete' : 'fallback';
      expect(LOCALES[code].coverage).toBe(expected);
    }
  });

  it('flags the three RTL scripts (Kashmiri, Sindhi, Urdu) and nothing else', () => {
    const rtl = LOCALE_CODES.filter((code) => LOCALES[code].dir === 'rtl').sort();
    expect(rtl).toEqual(['ks', 'sd', 'ur']);
  });
});

describe('resolveLocale', () => {
  it('resolves every one of the 20 codes without error', () => {
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

describe('tooltip translations', () => {
  it('has translated tooltips without placeholder suffixes in message bundles', async () => {
    const { hi } = await import('./messages/hi');
    const { kok } = await import('./messages/kok');
    const { bn } = await import('./messages/bn');
    const { ta } = await import('./messages/ta');
    const { mr } = await import('./messages/mr');

    expect(hi['tooltip.back']).toBe('पीछे जाएं');
    expect(hi['tooltip.search']).toBe('खोजें');
    expect(kok['tooltip.back']).toBe('फाटीं वचा');
    expect(kok['tooltip.search']).toBe('सोदा');
    expect(bn['tooltip.back']).toBe('ফিরে যান');
    expect(bn['tooltip.search']).toBe('অনুসন্ধান');
    expect(ta['tooltip.back']).toBe('பின்னே செல்க');
    expect(ta['tooltip.search']).toBe('தேடுக');
    expect(mr['tooltip.back']).toBe('मागे जा');
    expect(mr['tooltip.search']).toBe('शोधा');

    // Verify none of the tooltips contain the dummy "(hi)" or "(kok)" pattern
    for (const [key, val] of Object.entries(hi)) {
      if (key.startsWith('tooltip.')) {
        expect(val).not.toMatch(/\([a-z]{2,3}\)$/);
      }
    }
    for (const [key, val] of Object.entries(kok)) {
      if (key.startsWith('tooltip.')) {
        expect(val).not.toMatch(/\([a-z]{2,3}\)$/);
      }
    }
  });
});

