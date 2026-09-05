// packages/ui/src/a11y.svelte.ts
//
// The accessibility store: text scale, contrast mode, reduced motion.
// Persisted to the Dexie `prefs` table (@kalakriti/offline) -- exactly the
// device-local-settings table that batch built for this. Locale persistence
// stays on localStorage instead (see packages/i18n/src/locale.svelte.ts) to
// avoid a circular package dependency; this store has no such conflict,
// since @kalakriti/ui already depends on neither @kalakriti/offline nor is
// depended on by it.
//
// Applying the choice is one job: set the attribute tokens/scale.css and
// tokens/palette.css already key off (data-text-scale, data-theme,
// data-reduce-motion) on <html>. This file owns none of the actual CSS
// values -- those live in @kalakriti/tokens -- only the state and its
// persistence.

import { getPref, setPref } from '@kalakriti/offline';

export type TextScale = 'normal' | 'large' | 'xlarge' | 'xxlarge';
export type Contrast = 'normal' | 'high';

const PREF_KEYS = {
  textScale: 'a11y.textScale',
  contrast: 'a11y.contrast',
  reduceMotion: 'a11y.reduceMotion',
} as const;

function systemPrefersReducedMotion(): boolean {
  return (
    typeof matchMedia !== 'undefined' && matchMedia('(prefers-reduced-motion: reduce)').matches
  );
}

class A11yState {
  textScale = $state<TextScale>('normal');
  contrast = $state<Contrast>('normal');
  /** null = follow the OS setting. */
  reduceMotionOverride = $state<boolean | null>(null);
  #systemReducedMotion = $state(systemPrefersReducedMotion());

  reduceMotion: boolean = $derived(this.reduceMotionOverride ?? this.#systemReducedMotion);

  #apply(): void {
    if (typeof document === 'undefined') return;
    const root = document.documentElement;

    if (this.textScale === 'normal') delete root.dataset.textScale;
    else root.dataset.textScale = this.textScale;

    if (this.contrast === 'high') root.dataset.theme = 'high-contrast';
    else delete root.dataset.theme;

    if (this.reduceMotionOverride === null) delete root.dataset.reduceMotion;
    else root.dataset.reduceMotion = String(this.reduceMotionOverride);
  }

  /** Load persisted choices and apply them. Call once, from the root layout. */
  async init(): Promise<void> {
    const [textScale, contrast, reduceMotionOverride] = await Promise.all([
      getPref<TextScale>(PREF_KEYS.textScale),
      getPref<Contrast>(PREF_KEYS.contrast),
      getPref<boolean>(PREF_KEYS.reduceMotion),
    ]);
    if (textScale) this.textScale = textScale;
    if (contrast) this.contrast = contrast;
    if (reduceMotionOverride !== undefined) this.reduceMotionOverride = reduceMotionOverride;
    this.#apply();
  }

  /** Track the OS reduced-motion setting live. Returns a disposer. */
  start(): () => void {
    if (typeof matchMedia === 'undefined') return () => {};
    const query = matchMedia('(prefers-reduced-motion: reduce)');
    const update = () => {
      this.#systemReducedMotion = query.matches;
    };
    query.addEventListener('change', update);
    return () => query.removeEventListener('change', update);
  }

  async setTextScale(value: TextScale): Promise<void> {
    this.textScale = value;
    this.#apply();
    await setPref(PREF_KEYS.textScale, value);
  }

  async setContrast(value: Contrast): Promise<void> {
    this.contrast = value;
    this.#apply();
    await setPref(PREF_KEYS.contrast, value);
  }

  async setReduceMotionOverride(value: boolean | null): Promise<void> {
    this.reduceMotionOverride = value;
    this.#apply();
    await setPref(PREF_KEYS.reduceMotion, value);
  }
}

/** One per app, same reasoning as @kalakriti/i18n's `locale`: no SSR, no leak risk. */
export const a11y = new A11yState();

export const TEXT_SCALE_STEPS: readonly TextScale[] = ['normal', 'large', 'xlarge', 'xxlarge'];
