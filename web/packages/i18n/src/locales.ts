// packages/i18n/src/locales.ts
//
// The locale set is fixed at build time. Each translated locale ships as its
// own dynamically-imported chunk (see locale.svelte.ts's CATALOGUE_LOADERS),
// so a phone downloads only the language it actually selects -- that saving
// holds regardless of the PWA service worker's own precache behaviour. (A
// generated Workbox worker does exist -- via vite-plugin-pwa, configured in
// each app's vite.config.ts, not a hand-written file -- but its build-time
// `globPatterns` glob the whole build output with no runtime notion of "the
// selected language"; the vite.config.ts comment claiming it precaches only
// the active locale's chunk has not been verified against an actual build
// and should not be trusted without checking the generated sw.js.)
//
// Scripts are recorded alongside the tag so the font layer can bind a face per
// script without a second lookup table, and `dir` is carried explicitly rather
// than inferred: Urdu is the RTL case this product will hit, and inferring
// direction from the tag is exactly the kind of guess that breaks it.
//
// 20 of the 22 languages of the Eighth Schedule are listed here (Manipuri and
// Santali were dropped -- weakest script/font support of the 22, and neither
// had a real translated catalogue), per the batch spec's requirement that
// every supported language resolve to *something*.
// `coverage` records which ones actually have a translated catalogue --
// see messages/ -- versus which fall back through Hindi to English. Adding
// a real translation later is a messages/<code>.ts file plus one loader
// entry in locale.svelte.ts; this table does not change.

export interface LocaleMeta {
  /** BCP 47 tag, used verbatim for <html lang> and for Intl constructors. */
  readonly tag: string;
  /** The language's own name, always shown in that language. */
  readonly endonym: string;
  /** English name, for the admin dashboard's language columns. */
  readonly englishName: string;
  readonly dir: 'ltr' | 'rtl';
  /** ISO 15924 script code, for font binding. */
  readonly script: string;
  /** Locale used for number, currency and date formatting. */
  readonly numberLocale: string;
  /**
   * 'complete': messages/<code>.ts exists and is a human-authored source (en)
   * or full translation. 'machine': messages/<code>.ts exists but was
   * produced by translation rather than a human reviewer -- correct enough to
   * ship, but not yet verified the way 'complete' catalogues are. 'fallback':
   * messages/<code>.ts may exist, but catalogue-audit.ts finds it incomplete
   * (missing keys, untranslated pasted-Hindi values, or wrong script) --
   * lookups fall through the hi -> en chain for whatever it's missing.
   * catalogue-audit.ts is what actually gates 'complete'/'machine' vs
   * 'fallback' (see locales.test.ts); this field must never claim better
   * than the audit measures. Intl-driven formatting (numbers, dates) still
   * works for every locale regardless of this value -- it only describes the
   * message catalogue.
   */
  readonly coverage: 'complete' | 'machine' | 'fallback';
}

export const LOCALES = {
  en: {
    tag: 'en-IN',
    endonym: 'English',
    englishName: 'English',
    dir: 'ltr',
    script: 'Latn',
    numberLocale: 'en-IN',
    coverage: 'complete',
  },
  hi: {
    tag: 'hi-IN',
    endonym: 'हिन्दी',
    englishName: 'Hindi',
    dir: 'ltr',
    script: 'Deva',
    numberLocale: 'hi-IN',
    coverage: 'fallback',
  },
  as: {
    tag: 'as-IN',
    endonym: 'অসমীয়া',
    englishName: 'Assamese',
    dir: 'ltr',
    script: 'Beng',
    numberLocale: 'as-IN',
    coverage: 'fallback',
  },
  bn: {
    tag: 'bn-IN',
    endonym: 'বাংলা',
    englishName: 'Bengali',
    dir: 'ltr',
    script: 'Beng',
    numberLocale: 'bn-IN',
    coverage: 'fallback',
  },
  brx: {
    tag: 'brx-IN',
    endonym: 'बड़ो',
    englishName: 'Bodo',
    dir: 'ltr',
    script: 'Deva',
    numberLocale: 'brx-IN',
    coverage: 'fallback',
  },
  doi: {
    tag: 'doi-IN',
    endonym: 'डोगरी',
    englishName: 'Dogri',
    dir: 'ltr',
    script: 'Deva',
    numberLocale: 'doi-IN',
    coverage: 'fallback',
  },
  gu: {
    tag: 'gu-IN',
    endonym: 'ગુજરાતી',
    englishName: 'Gujarati',
    dir: 'ltr',
    script: 'Gujr',
    numberLocale: 'gu-IN',
    coverage: 'fallback',
  },
  kn: {
    tag: 'kn-IN',
    endonym: 'ಕನ್ನಡ',
    englishName: 'Kannada',
    dir: 'ltr',
    script: 'Knda',
    numberLocale: 'kn-IN',
    coverage: 'fallback',
  },
  ks: {
    tag: 'ks-IN',
    endonym: 'کٲشُر',
    englishName: 'Kashmiri',
    dir: 'rtl',
    script: 'Arab',
    numberLocale: 'ks-IN',
    coverage: 'fallback',
  },
  kok: {
    tag: 'kok-IN',
    endonym: 'कोंकणी',
    englishName: 'Konkani',
    dir: 'ltr',
    script: 'Deva',
    numberLocale: 'kok-IN',
    coverage: 'fallback',
  },
  mai: {
    tag: 'mai-IN',
    endonym: 'मैथिली',
    englishName: 'Maithili',
    dir: 'ltr',
    script: 'Deva',
    numberLocale: 'mai-IN',
    coverage: 'fallback',
  },
  ml: {
    tag: 'ml-IN',
    endonym: 'മലയാളം',
    englishName: 'Malayalam',
    dir: 'ltr',
    script: 'Mlym',
    numberLocale: 'ml-IN',
    coverage: 'fallback',
  },
  mr: {
    tag: 'mr-IN',
    endonym: 'मराठी',
    englishName: 'Marathi',
    dir: 'ltr',
    script: 'Deva',
    numberLocale: 'mr-IN',
    coverage: 'fallback',
  },
  ne: {
    tag: 'ne-IN',
    endonym: 'नेपाली',
    englishName: 'Nepali',
    dir: 'ltr',
    script: 'Deva',
    numberLocale: 'ne-IN',
    coverage: 'fallback',
  },
  or: {
    tag: 'or-IN',
    endonym: 'ଓଡ଼ିଆ',
    englishName: 'Odia',
    dir: 'ltr',
    script: 'Orya',
    numberLocale: 'or-IN',
    coverage: 'fallback',
  },
  pa: {
    tag: 'pa-IN',
    endonym: 'ਪੰਜਾਬੀ',
    englishName: 'Punjabi',
    dir: 'ltr',
    script: 'Guru',
    numberLocale: 'pa-IN',
    coverage: 'fallback',
  },
  sa: {
    tag: 'sa-IN',
    endonym: 'संस्कृतम्',
    englishName: 'Sanskrit',
    dir: 'ltr',
    script: 'Deva',
    numberLocale: 'sa-IN',
    coverage: 'fallback',
  },
  sd: {
    tag: 'sd-IN',
    endonym: 'سنڌي',
    englishName: 'Sindhi',
    dir: 'rtl',
    script: 'Arab',
    numberLocale: 'sd-IN',
    coverage: 'fallback',
  },
  ta: {
    tag: 'ta-IN',
    endonym: 'தமிழ்',
    englishName: 'Tamil',
    dir: 'ltr',
    script: 'Taml',
    numberLocale: 'ta-IN',
    coverage: 'fallback',
  },
  te: {
    tag: 'te-IN',
    endonym: 'తెలుగు',
    englishName: 'Telugu',
    dir: 'ltr',
    script: 'Telu',
    numberLocale: 'te-IN',
    coverage: 'fallback',
  },
  ur: {
    tag: 'ur-IN',
    endonym: 'اردو',
    englishName: 'Urdu',
    dir: 'rtl',
    script: 'Arab',
    numberLocale: 'ur-IN',
    coverage: 'fallback',
  },
} as const satisfies Record<string, LocaleMeta>;

export type LocaleCode = keyof typeof LOCALES;

export const LOCALE_CODES = Object.keys(LOCALES) as LocaleCode[];

/** The 20 supported scheduled languages (English is the source locale). */
export const SUPPORTED_LOCALES = LOCALE_CODES.filter((code) => code !== 'en');

export const DEFAULT_LOCALE: LocaleCode = 'hi';

export function isLocaleCode(value: string): value is LocaleCode {
  return Object.prototype.hasOwnProperty.call(LOCALES, value);
}

/**
 * Whether a `listing_translation.language` value matches a UI locale. The
 * real backend stores the trimmed proto enum name (e.g. "HINDI"); a listing
 * drafted offline before syncing may instead carry the mock pipeline's
 * lowercase locale code (e.g. "hi", see apps/artisan/src/lib/ml-mock.ts) --
 * this matches either, so display code works regardless of which one wrote
 * the row.
 */
export function matchesLocale(language: string, locale: LocaleCode): boolean {
  return language === locale || language === LOCALES[locale].englishName.toUpperCase();
}

/**
 * Best locale for a set of Accept-Language-style preferences.
 * Matches on the primary subtag only: `hi-IN`, `hi-Deva-IN` and `hi` all
 * resolve to `hi`, which is what a language picker actually means.
 */
export function resolveLocale(preferred: readonly string[]): LocaleCode {
  for (const raw of preferred) {
    const primary = raw.toLowerCase().split('-')[0];
    if (primary && isLocaleCode(primary)) return primary;
  }
  return DEFAULT_LOCALE;
}
