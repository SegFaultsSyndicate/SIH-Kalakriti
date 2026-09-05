// packages/i18n/src/locales.ts
//
// The locale set is fixed at build time because the artisan service worker
// precaches exactly one message bundle -- the one for the selected language --
// and it cannot precache a language it has never heard of.
//
// Scripts are recorded alongside the tag so the font layer can bind a face per
// script without a second lookup table, and `dir` is carried explicitly rather
// than inferred: Urdu is the RTL case this product will hit, and inferring
// direction from the tag is exactly the kind of guess that breaks it.
//
// All 22 languages of the Eighth Schedule are listed here, per the batch
// spec's requirement that every scheduled language resolve to *something*.
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
   * 'complete': messages/<code>.ts exists and is the source (en) or a full
   * translation (hi). 'fallback': no catalogue yet; resolves through the
   * hi -> en chain. Intl-driven formatting (numbers, dates) still works for
   * every locale regardless of this value -- it only describes the message
   * catalogue.
   */
  readonly coverage: 'complete' | 'fallback';
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
    coverage: 'complete',
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
    coverage: 'complete',
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
    coverage: 'complete',
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
  mni: {
    tag: 'mni-IN',
    endonym: 'ꯃꯤꯇꯩꯂꯣꯟ',
    englishName: 'Manipuri',
    dir: 'ltr',
    script: 'Mtei',
    numberLocale: 'mni-IN',
    coverage: 'fallback',
  },
  mr: {
    tag: 'mr-IN',
    endonym: 'मराठी',
    englishName: 'Marathi',
    dir: 'ltr',
    script: 'Deva',
    numberLocale: 'mr-IN',
    coverage: 'complete',
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
    coverage: 'complete',
  },
  pa: {
    tag: 'pa-IN',
    endonym: 'ਪੰਜਾਬੀ',
    englishName: 'Punjabi',
    dir: 'ltr',
    script: 'Guru',
    numberLocale: 'pa-IN',
    coverage: 'complete',
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
  sat: {
    tag: 'sat-IN',
    endonym: 'ᱥᱟᱱᱛᱟᱲᱤ',
    englishName: 'Santali',
    dir: 'ltr',
    script: 'Olck',
    numberLocale: 'sat-IN',
    coverage: 'fallback',
  },
  sd: {
    tag: 'sd-IN',
    endonym: 'سنڌي',
    englishName: 'Sindhi',
    dir: 'rtl',
    script: 'Arab',
    numberLocale: 'sd-IN',
    coverage: 'complete',
  },
  ta: {
    tag: 'ta-IN',
    endonym: 'தமிழ்',
    englishName: 'Tamil',
    dir: 'ltr',
    script: 'Taml',
    numberLocale: 'ta-IN',
    coverage: 'complete',
  },
  te: {
    tag: 'te-IN',
    endonym: 'తెలుగు',
    englishName: 'Telugu',
    dir: 'ltr',
    script: 'Telu',
    numberLocale: 'te-IN',
    coverage: 'complete',
  },
  ur: {
    tag: 'ur-IN',
    endonym: 'اردو',
    englishName: 'Urdu',
    dir: 'rtl',
    script: 'Arab',
    numberLocale: 'ur-IN',
    coverage: 'complete',
  },
} as const satisfies Record<string, LocaleMeta>;

export type LocaleCode = keyof typeof LOCALES;

export const LOCALE_CODES = Object.keys(LOCALES) as LocaleCode[];

/** The 22 constitutionally scheduled languages (English is the source locale). */
export const SUPPORTED_LOCALES = LOCALE_CODES.filter((code) => code !== 'en');

export const DEFAULT_LOCALE: LocaleCode = 'hi';

export function isLocaleCode(value: string): value is LocaleCode {
  return Object.prototype.hasOwnProperty.call(LOCALES, value);
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
