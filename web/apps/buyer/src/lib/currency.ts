// apps/buyer/src/lib/currency.ts
//
// Catalogue prices arrive from the API as paise of INR -- the base currency.
// This conversion-rate object maps every selectable display currency to its
// worth in rupees; the base currency keeps the identity rate of 1. Rates are
// hardcoded, human-checked dev figures until a live FX feed replaces them.

import type { CurrencyCode } from '@kalakriti/i18n';

export const DEFAULT_CURRENCY: CurrencyCode = 'INR';

export const CURRENCY_RATES: Record<CurrencyCode, number> = {
  INR: 1,
  USD: 0.012,
  EUR: 0.011,
  GBP: 0.0095,
  AED: 0.044,
  JPY: 1.8,
};

/** Convert base-currency paise to the selected currency's own paise, integer-safe. */
export function convertPaise(paise: number, rate: number): number {
  return Math.round(paise * rate);
}