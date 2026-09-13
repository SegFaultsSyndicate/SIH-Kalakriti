// apps/buyer/src/lib/currency.svelte.ts
//
// Catalogue prices arrive from the API as paise of INR -- the base currency.
// CURRENCY_RATES maps every selectable display currency to its worth in
// rupees; the base currency keeps the identity rate of 1. Rates are hardcoded,
// human-checked dev figures until a live FX feed replaces them.
//
// The selection is a rune-backed singleton, the same pattern @kalakriti/i18n's
// locale store uses: it is one long-lived piece of app state whose reads
// happen in the price line of every listing card, so $state gives fine-grained
// reactivity without a subscription per card. Persistence is localStorage, and
// like the locale store, module scope is correct because there is no SSR.

import { CURRENCY_META, type CurrencyCode, type CurrencyMeta } from '@kalakriti/i18n';

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

const STORAGE_KEY = 'kalakriti.currency';

function readStoredCurrency(): CurrencyCode | null {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored && stored in CURRENCY_RATES ? (stored as CurrencyCode) : null;
  } catch {
    // Private browsing, or storage disabled by policy. Not an error: the
    // picker still works, the choice just does not survive a reload.
    return null;
  }
}

class CurrencyState {
  #code = $state<CurrencyCode>(DEFAULT_CURRENCY);

  get code(): CurrencyCode {
    return this.#code;
  }

  get meta(): CurrencyMeta {
    return CURRENCY_META[this.#code];
  }

  /** Restore the stored choice once from the root layout; safe to call again. */
  init(): void {
    this.#code = readStoredCurrency() ?? DEFAULT_CURRENCY;
  }

  set(code: CurrencyCode): void {
    this.#code = code;
    try {
      localStorage.setItem(STORAGE_KEY, code);
    } catch {
      // See readStoredCurrency: a failure to persist is not a failure to switch.
    }
  }
}

/** One instance per app; see the module comment for why module scope is right. */
export const currency = new CurrencyState();