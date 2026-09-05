// packages/i18n/src/format.ts
//
// Formatting that must never be done ad hoc in a component.
//
// MONEY IS INT64 PAISE, EVERYWHERE. Two rules follow and neither is optional:
//   1. A raw paise integer is never rendered. 45000 is not a price.
//   2. No float math. `paise / 100` is a float division and will produce
//      1234.5599999999999 for values a rupee-and-paise display would show
//      wrong. The rupee and paise parts are split with integer arithmetic and
//      recombined as text.
//
// Indian digit grouping (1,23,456 -- lakh/crore, not thousands) is what
// `en-IN` and `hi-IN` give from Intl, which is why the locale table carries a
// numberLocale rather than reusing the language tag.

import { LOCALES, type LocaleCode } from './locales';

/** Money as it crosses the API boundary: int64 paise. */
export type Paise = number;

const RUPEE = '₹';

/**
 * Split paise into integer rupees and the 0-99 remainder without float math.
 * Exact for any |paise| below Number.MAX_SAFE_INTEGER, which covers every
 * value an int64 paise field can carry in a price or a statement line.
 */
function splitPaise(paise: Paise): { negative: boolean; rupees: number; remainder: number } {
  if (!Number.isInteger(paise)) {
    throw new TypeError(`money must be an integer number of paise, received ${paise}`);
  }
  const negative = paise < 0;
  const abs = Math.abs(paise);
  return { negative, rupees: Math.trunc(abs / 100), remainder: abs % 100 };
}

export interface MoneyOptions {
  /**
   * Show the paise remainder. Default is 'auto': shown only when non-zero,
   * because a catalogue full of ".00" reads as machine output, while a price
   * that silently drops 50 paise is a bug the artisan will notice first.
   */
  paise?: 'auto' | 'always' | 'never';
  /** Prefix the rupee sign. Off for table cells that carry the unit in the header. */
  symbol?: boolean;
}

export function formatMoney(
  paise: Paise,
  locale: LocaleCode,
  options: MoneyOptions = {},
): string {
  const { paise: paiseMode = 'auto', symbol = true } = options;
  const { negative, rupees, remainder } = splitPaise(paise);
  const grouped = new Intl.NumberFormat(LOCALES[locale].numberLocale, {
    useGrouping: true,
    maximumFractionDigits: 0,
  }).format(rupees);

  const showRemainder = paiseMode === 'always' || (paiseMode === 'auto' && remainder !== 0);
  const tail = showRemainder ? `.${String(remainder).padStart(2, '0')}` : '';

  return `${negative ? '-' : ''}${symbol ? RUPEE : ''}${grouped}${tail}`;
}

/**
 * A price band, as PriceAdvice returns one. Rendered as a single range so the
 * two ends cannot be read as two separate prices.
 */
export function formatMoneyRange(minPaise: Paise, maxPaise: Paise, locale: LocaleCode): string {
  return `${formatMoney(minPaise, locale)}–${formatMoney(maxPaise, locale, { symbol: false })}`;
}

export function formatNumber(value: number, locale: LocaleCode): string {
  return new Intl.NumberFormat(LOCALES[locale].numberLocale).format(value);
}

export function formatDate(
  value: Date | string | number,
  locale: LocaleCode,
  options: Intl.DateTimeFormatOptions = { dateStyle: 'medium' },
): string {
  return new Intl.DateTimeFormat(LOCALES[locale].numberLocale, options).format(
    value instanceof Date ? value : new Date(value),
  );
}

const RELATIVE_STEPS: ReadonlyArray<[Intl.RelativeTimeFormatUnit, number]> = [
  ['second', 60],
  ['minute', 60],
  ['hour', 24],
  ['day', 7],
  ['week', 4.348],
  ['month', 12],
  ['year', Number.POSITIVE_INFINITY],
];

/**
 * "2 minutes ago". Used on outbox rows, where the artisan needs to know how
 * stale a queued item is more than they need its wall-clock time.
 */
export function formatRelativeTime(
  value: Date | string | number,
  locale: LocaleCode,
  now: Date = new Date(),
): string {
  const then = value instanceof Date ? value : new Date(value);
  let delta = (then.getTime() - now.getTime()) / 1000;
  const formatter = new Intl.RelativeTimeFormat(LOCALES[locale].numberLocale, {
    numeric: 'auto',
  });

  for (const [unit, span] of RELATIVE_STEPS) {
    if (Math.abs(delta) < span) {
      return formatter.format(Math.round(delta), unit);
    }
    delta /= span;
  }
  return formatter.format(Math.round(delta), 'year');
}

export function formatList(items: readonly string[], locale: LocaleCode): string {
  return new Intl.ListFormat(LOCALES[locale].numberLocale, {
    style: 'long',
    type: 'conjunction',
  }).format(items);
}

/** 0.87 -> "87%". ML confidence is always shown with a label, never alone. */
export function formatPercent(fraction: number, locale: LocaleCode): string {
  return new Intl.NumberFormat(LOCALES[locale].numberLocale, {
    style: 'percent',
    maximumFractionDigits: 0,
  }).format(fraction);
}
