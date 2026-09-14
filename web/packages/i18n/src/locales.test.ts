// packages/i18n/src/locales.test.ts
import { beforeAll, describe, expect, it } from 'vitest';
import {
  LOCALES,
  LOCALE_CODES,
  SUPPORTED_LOCALES,
  isLocaleCode,
  resolveLocale,
  type LocaleCode,
} from './locales';
import { en } from './messages/en';
import { auditCatalogue } from './catalogue-audit';

const NON_EN: LocaleCode[] = LOCALE_CODES.filter((code) => code !== 'en');
const catalogues = new Map<LocaleCode, Record<string, string>>();

beforeAll(async () => {
  const loaded = await Promise.all(
    NON_EN.map((code) => import(`./messages/${code}.ts`) as Promise<Record<string, unknown>>),
  );
  NON_EN.forEach((code, i) => catalogues.set(code, loaded[i][code] as Record<string, string>));
});

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

  it('never declares complete/machine coverage for a catalogue the audit finds broken', () => {
    // Regression guard for the exact bug this test used to have: it asserted
    // a hardcoded list of "complete" locales without ever reading a
    // catalogue file, so 9 locales that were ~90% pasted Hindi stayed
    // labelled 'complete' indefinitely. See catalogue-audit.ts and
    // I18N_PLAN.md's Phase 1 for the real measurement; a locale may only
    // claim 'complete' or 'machine' once auditCatalogue reports zero issues
    // against it.
    expect(LOCALES.en.coverage).toBe('complete');
    const hi = catalogues.get('hi') ?? {};
    for (const code of NON_EN) {
      const issues = auditCatalogue(code, catalogues.get(code)!, en, hi);
      if (issues.length === 0) {
        expect(['complete', 'machine']).toContain(LOCALES[code].coverage);
      } else {
        expect(LOCALES[code].coverage).toBe('fallback');
      }
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
