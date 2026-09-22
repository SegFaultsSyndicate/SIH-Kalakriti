// packages/i18n/src/index.ts
export {
  LOCALES,
  LOCALE_CODES,
  SUPPORTED_LOCALES,
  DEFAULT_LOCALE,
  isLocaleCode,
  resolveLocale,
  matchesLocale,
  type LocaleCode,
  type LocaleMeta,
} from './locales';
export { FALLBACK_CATALOGUES, FALLBACK_LOCALE_CHAIN } from './messages/fallback';
export { en, type MessageKey, type Messages } from './messages/en';
export {
  locale,
  t,
  tPlural,
  tooltip,
  hasExplicitLocale,
  type Translate,
  type MessageValues,
} from './locale.svelte';
export type { DntTerm } from './dnt-terms';
export {
  formatMoney,
  formatMoneyRange,
  formatNumber,
  formatDate,
  formatRelativeTime,
  formatList,
  formatPercent,
  CURRENCY_META,
  type Paise,
  type MoneyOptions,
  type CurrencyCode,
  type CurrencyMeta,
} from './format';
