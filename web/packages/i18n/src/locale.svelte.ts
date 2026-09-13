// packages/i18n/src/locale.svelte.ts
//
// The locale store. A rune-backed class rather than a Svelte store: this is
// one long-lived piece of app state, reads happen in a hundred components, and
// $state gives them fine-grained reactivity without a subscription each.
//
// Catalogues load dynamically. The artisan app runs on 2G, and shipping every
// language's strings to a phone that will only ever show one of them is the
// cheapest large win available at this stage. Each catalogue becomes its own
// chunk, and the service worker precaches only the active one.
//
// English is imported statically because it is the fallback of last resort: a
// key missing from a translation must resolve to readable text without a
// second network round trip. Hindi sits between the active locale and English
// in the chain (see #hiCatalogue below) because it is the only other complete
// catalogue -- for the 20 locales with no messages/<code>.ts yet, Hindi is a
// better fallback than English for most of their speakers.
//
// Persistence is localStorage, not the Dexie `prefs` table @kalakriti/offline
// exposes for exactly this kind of setting: @kalakriti/offline already
// depends on @kalakriti/i18n (its ConflictBanner uses `t`), so the reverse
// dependency would be circular. localStorage has no such issue and is
// synchronous, which matters here -- the locale must be known before first
// paint, not after an IndexedDB round trip.
//
// No i18n library (Paraglide/inlang, formatjs) is pulled in: the message set
// is small, plural handling needs only Intl.PluralRules (native, zero-KB),
// and per-locale code splitting is one line of dynamic import per catalogue.
// A library earns its weight when the catalogue or the plural/gender grammar
// outgrows what a native Intl object expresses -- not yet, here.

import { en, type MessageKey, type Messages } from './messages/en';
import { FALLBACK_CATALOGUES } from './messages/fallback';
import { interpolate, selectPluralKey, type MessageValues } from './format-message';
import {
  DEFAULT_LOCALE,
  LOCALES,
  isLocaleCode,
  resolveLocale,
  type LocaleCode,
  type LocaleMeta,
} from './locales';

declare global {
  interface ImportMeta {
    readonly env: ImportMetaEnv;
  }
}

const STORAGE_KEY = 'kalakriti.locale';
const DEV = Boolean(import.meta.env?.DEV);

export type { MessageValues } from './format-message';
export type Translate = (key: MessageKey, values?: MessageValues) => string;

/**
 * Only the locales with a real catalogue get a loader. Everything else falls
 * through the hi -> en chain below -- that IS "support all 22 in the
 * catalogue structure", per the batch spec: LOCALES lists all 22, this map
 * lists only the translated ones, and set() never throws for a code that's
 * missing from it.
 */
const CATALOGUE_LOADERS: Record<LocaleCode, () => Promise<Partial<Messages>>> = {
  en: async () => en,
  hi: async () => (await import('./messages/hi')).hi,
  bn: async () => (await import('./messages/bn')).bn,
  ta: async () => (await import('./messages/ta')).ta,
  te: async () => (await import('./messages/te')).te,
  gu: async () => (await import('./messages/gu')).gu,
  mr: async () => (await import('./messages/mr')).mr,
  sd: async () => (await import('./messages/sd')).sd,
  ur: async () => (await import('./messages/ur')).ur,
  pa: async () => (await import('./messages/pa')).pa,
  or: async () => (await import('./messages/or')).or,
  kn: async () => (await import('./messages/kn')).kn,
  ml: async () => (await import('./messages/ml')).ml,
  as: async () => (await import('./messages/as')).as,
  mai: async () => (await import('./messages/mai')).mai,
  kok: async () => (await import('./messages/kok')).kok,
  doi: async () => (await import('./messages/doi')).doi,
  ks: async () => (await import('./messages/ks')).ks,
  ne: async () => (await import('./messages/ne')).ne,
  sa: async () => (await import('./messages/sa')).sa,
  brx: async () => (await import('./messages/brx')).brx,
};

/**
 * Whether the artisan has ever explicitly chosen a language (as opposed to
 * `init()` having resolved one from the browser's Accept-Language list).
 * The onboarding gate in apps/artisan uses this to decide whether /language
 * must be shown before anything else -- a resolved default is not a choice.
 */
export function hasExplicitLocale(): boolean {
  return readStoredLocale() !== null;
}

function readStoredLocale(): LocaleCode | null {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored && isLocaleCode(stored) ? stored : null;
  } catch {
    // Private browsing, or storage disabled by policy. Not an error: the
    // language picker still works, the choice just does not survive a reload.
    return null;
  }
}

class LocaleState {
  #code = $state<LocaleCode>(DEFAULT_LOCALE);
  #catalogue = $state<Partial<Messages>>({});
  #hiCatalogue = $state<Partial<Messages> | null>(null);
  #hiCataloguePromise: Promise<Partial<Messages>> | null = null;
  #loading = $state(false);
  #version = $state(0);

  get code(): LocaleCode {
    return this.#code;
  }

  get meta(): LocaleMeta {
    return LOCALES[this.#code];
  }

  get loading(): boolean {
    return this.#loading;
  }

  /** Active catalogue -> Hindi -> English -> the raw key itself. */
  #lookup(key: MessageKey): string {
    const found = this.#catalogue[key] ?? this.#hiCatalogue?.[key] ?? en[key];
    if (found === undefined) {
      if (DEV) console.warn(`[i18n] missing key "${key}" for locale "${this.#code}"`);
      return key;
    }
    return found;
  }

  /**
   * Reactive translate. Read it inside a component and the component
   * re-renders when the language changes -- there is no separate subscription
   * to remember to tear down.
   */
  get t(): Translate {
    // Referenced so the getter re-derives when code, version, or either catalogue changes.
    void this.#code;
    void this.#catalogue;
    void this.#hiCatalogue;
    void this.#version;
    return (key, values) => interpolate(this.#lookup(key), values);
  }

  /**
   * Plural-aware translate. `base.one` / `base.other` (etc, per CLDR
   * category) are looked up as ordinary keys through the same active -> hi ->
   * en chain as `t`; `base.other` is the required fallback if the selected
   * category isn't present, since every language has an "other" category.
   */
  tPlural(base: string, count: number, values?: MessageValues): string {
    const hasKey = (key: string) =>
      key in this.#catalogue || key in (this.#hiCatalogue ?? {}) || key in en;
    const key = selectPluralKey(base, count, this.meta.tag, hasKey) as MessageKey;
    return interpolate(this.#lookup(key), { count, ...values });
  }

  /**
   * Resolve the startup language: an explicit stored choice wins, otherwise
   * the browser's preference list, otherwise Hindi. Called once from the root
   * layout; safe to call again.
   */
  async init(): Promise<void> {
    const stored = readStoredLocale();
    const preferred = stored
      ? [stored]
      : typeof navigator !== 'undefined'
        ? [...navigator.languages]
        : [];
    await this.set(resolveLocale(preferred), { persist: false });
  }

  async set(code: LocaleCode, options: { persist?: boolean } = {}): Promise<void> {
    const { persist = true } = options;
    this.#loading = true;
    try {
      const loader = CATALOGUE_LOADERS[code] ?? (async () => FALLBACK_CATALOGUES[code] ?? {});
      this.#catalogue = await loader();
      this.#code = code;

      if (code !== 'en' && code !== 'hi') {
        // Load Hindi before resolving set(), so a caller never observes the
        // temporary English fallback for a scheduled language.
        this.#hiCataloguePromise ??= CATALOGUE_LOADERS.hi().then((hi) => {
          this.#hiCatalogue = hi;
          return hi;
        });
        await this.#hiCataloguePromise;
      }

      this.#version++;

      if (typeof document !== 'undefined') {
        // <html lang> and dir are what a screen reader switches voice on, and
        // what the CSS logical properties resolve against. Setting them here
        // keeps that from being every layout's job to remember.
        document.documentElement.lang = LOCALES[code].tag;
        document.documentElement.dir = LOCALES[code].dir;
      }
      if (persist) {
        try {
          localStorage.setItem(STORAGE_KEY, code);
        } catch {
          // See readStoredLocale: a failure to persist is not a failure to switch.
        }
      }
    } finally {
      this.#loading = false;
    }
  }
}

/**
 * One instance per app. Module scope is correct here precisely because there
 * is no SSR: nothing renders on a server, so there is no request whose state
 * could leak into another's.
 */
export const locale = new LocaleState();

/** Convenience for `const t = $derived(locale.t)` at the top of a component. */
export function t(key: MessageKey, values?: MessageValues): string {
  return locale.t(key, values);
}

/** Convenience for `const tp = $derived(locale.tPlural.bind(locale))`. */
export function tPlural(base: string, count: number, values?: MessageValues): string {
  return locale.tPlural(base, count, values);
}
