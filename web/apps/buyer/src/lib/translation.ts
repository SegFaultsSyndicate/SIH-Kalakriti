// apps/buyer/src/lib/translation.ts
//
// Language selection utilities to accurately match translations according to
// the active user locale (ISO code e.g. 'en', 'hi') or full English/Hindi names.

export interface TranslationLike {
  language?: string;
  title?: string;
  description?: string;
  highlights?: string[] | null;
  machine_generated?: boolean;
}

/**
 * Returns the title matching the user's chosen locale.
 * Priority:
 * 1. Exact or canonical match for current locale (e.g. 'en' / 'english' or 'hi' / 'hindi')
 * 2. Fallback to English if current locale is not available
 * 3. Fallback to first non-empty translation
 */
export function getListingTitle(translations?: TranslationLike[] | null, localeCode = 'en'): string {
  if (!translations || translations.length === 0) return '';
  const lang = (localeCode || 'en').toLowerCase();

  // 1. Direct match for active locale
  const match = translations.find((tr) => {
    const l = (tr.language ?? '').toLowerCase();
    if (l === lang) return true;
    if (lang === 'en' && (l === 'english' || l === 'en-in' || l === 'en-us')) return true;
    if (lang === 'hi' && (l === 'hindi' || l === 'hi-in')) return true;
    return false;
  });
  if (match?.title) return match.title;

  // 2. If user wanted something else and it was not found, fallback to English
  const enMatch = translations.find((tr) => {
    const l = (tr.language ?? '').toLowerCase();
    return l === 'en' || l === 'english' || l === 'en-in';
  });
  if (enMatch?.title) return enMatch.title;

  // 3. Fallback to first available title
  for (const tr of translations) {
    if (tr.title) return tr.title;
  }
  return '';
}

/**
 * Returns the description matching the user's chosen locale.
 */
export function getListingDescription(translations?: TranslationLike[] | null, localeCode = 'en'): string {
  if (!translations || translations.length === 0) return '';
  const lang = (localeCode || 'en').toLowerCase();

  const match = translations.find((tr) => {
    const l = (tr.language ?? '').toLowerCase();
    if (l === lang) return true;
    if (lang === 'en' && (l === 'english' || l === 'en-in' || l === 'en-us')) return true;
    if (lang === 'hi' && (l === 'hindi' || l === 'hi-in')) return true;
    return false;
  });
  if (match?.description) return match.description;

  const enMatch = translations.find((tr) => {
    const l = (tr.language ?? '').toLowerCase();
    return l === 'en' || l === 'english' || l === 'en-in';
  });
  if (enMatch?.description) return enMatch.description;

  for (const tr of translations) {
    if (tr.description) return tr.description;
  }
  return '';
}
