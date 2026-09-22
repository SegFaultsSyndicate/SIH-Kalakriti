import { en, type Messages } from './en';
import { LOCALE_CODES, type LocaleCode } from '../locales';

/**
 * The scheduled languages are supported even before a translated catalogue is
 * available.  Keeping this map explicit gives consumers (and the accessibility
 * statement) one source of truth for catalogue coverage without duplicating the
 * English message object twenty times.
 */
export const FALLBACK_CATALOGUES: Readonly<Record<LocaleCode, Partial<Messages>>> =
  Object.fromEntries(LOCALE_CODES.map((code) => [code, code === 'en' ? en : {}])) as Record<
    LocaleCode,
    Partial<Messages>
  >;

export const FALLBACK_LOCALE_CHAIN: readonly LocaleCode[] = ['hi', 'en'];
