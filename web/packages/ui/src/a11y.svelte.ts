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
export type LineSpacing = 'normal' | 'relaxed';

const PREF_KEYS = {
  textScale: 'a11y.textScale',
  contrast: 'a11y.contrast',
  reduceMotion: 'a11y.reduceMotion',
  readableFont: 'a11y.readableFont',
  lineSpacing: 'a11y.lineSpacing',
  highlightLinks: 'a11y.highlightLinks',
  monochrome: 'a11y.monochrome',
  bigCursor: 'a11y.bigCursor',
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
  readableFont = $state<boolean>(false);
  lineSpacing = $state<LineSpacing>('normal');
  highlightLinks = $state<boolean>(false);
  monochrome = $state<boolean>(false);
  bigCursor = $state<boolean>(false);
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

    if (this.readableFont) root.dataset.font = 'readable';
    else delete root.dataset.font;

    if (this.lineSpacing === 'relaxed') root.dataset.lineSpacing = 'relaxed';
    else delete root.dataset.lineSpacing;

    if (this.highlightLinks) root.dataset.highlightLinks = 'true';
    else delete root.dataset.highlightLinks;

    if (this.monochrome) root.dataset.monochrome = 'true';
    else delete root.dataset.monochrome;

    if (this.bigCursor) root.dataset.bigCursor = 'true';
    else delete root.dataset.bigCursor;
  }

  /** Load persisted choices and apply them. Call once, from the root layout. */
  async init(): Promise<void> {
    const [
      textScale,
      contrast,
      reduceMotionOverride,
      readableFont,
      lineSpacing,
      highlightLinks,
      monochrome,
      bigCursor,
    ] = await Promise.all([
      getPref<TextScale>(PREF_KEYS.textScale),
      getPref<Contrast>(PREF_KEYS.contrast),
      getPref<boolean>(PREF_KEYS.reduceMotion),
      getPref<boolean>(PREF_KEYS.readableFont),
      getPref<LineSpacing>(PREF_KEYS.lineSpacing),
      getPref<boolean>(PREF_KEYS.highlightLinks),
      getPref<boolean>(PREF_KEYS.monochrome),
      getPref<boolean>(PREF_KEYS.bigCursor),
    ]);
    if (textScale) this.textScale = textScale;
    if (contrast) this.contrast = contrast;
    if (reduceMotionOverride !== undefined) this.reduceMotionOverride = reduceMotionOverride;
    if (readableFont !== undefined) this.readableFont = Boolean(readableFont);
    if (lineSpacing) this.lineSpacing = lineSpacing;
    if (highlightLinks !== undefined) this.highlightLinks = Boolean(highlightLinks);
    if (monochrome !== undefined) this.monochrome = Boolean(monochrome);
    if (bigCursor !== undefined) this.bigCursor = Boolean(bigCursor);
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

  async stepTextScale(): Promise<void> {
    const steps: TextScale[] = ['normal', 'large', 'xlarge', 'xxlarge'];
    const idx = steps.indexOf(this.textScale);
    const next = steps[(idx + 1) % steps.length];
    await this.setTextScale(next);
  }

  async setContrast(value: Contrast): Promise<void> {
    this.contrast = value;
    this.#apply();
    await setPref(PREF_KEYS.contrast, value);
  }

  async toggleContrast(): Promise<void> {
    await this.setContrast(this.contrast === 'high' ? 'normal' : 'high');
  }

  async setReduceMotionOverride(value: boolean | null): Promise<void> {
    this.reduceMotionOverride = value;
    this.#apply();
    await setPref(PREF_KEYS.reduceMotion, value);
  }

  async setReadableFont(value: boolean): Promise<void> {
    this.readableFont = value;
    this.#apply();
    await setPref(PREF_KEYS.readableFont, value);
  }

  async setLineSpacing(value: LineSpacing): Promise<void> {
    this.lineSpacing = value;
    this.#apply();
    await setPref(PREF_KEYS.lineSpacing, value);
  }

  async setHighlightLinks(value: boolean): Promise<void> {
    this.highlightLinks = value;
    this.#apply();
    await setPref(PREF_KEYS.highlightLinks, value);
  }

  async setMonochrome(value: boolean): Promise<void> {
    this.monochrome = value;
    this.#apply();
    await setPref(PREF_KEYS.monochrome, value);
  }

  async setBigCursor(value: boolean): Promise<void> {
    this.bigCursor = value;
    this.#apply();
    await setPref(PREF_KEYS.bigCursor, value);
  }

  async resetAll(): Promise<void> {
    this.textScale = 'normal';
    this.contrast = 'normal';
    this.reduceMotionOverride = null;
    this.readableFont = false;
    this.lineSpacing = 'normal';
    this.highlightLinks = false;
    this.monochrome = false;
    this.bigCursor = false;
    this.#apply();
    await Promise.all([
      setPref(PREF_KEYS.textScale, 'normal'),
      setPref(PREF_KEYS.contrast, 'normal'),
      setPref(PREF_KEYS.reduceMotion, null),
      setPref(PREF_KEYS.readableFont, false),
      setPref(PREF_KEYS.lineSpacing, 'normal'),
      setPref(PREF_KEYS.highlightLinks, false),
      setPref(PREF_KEYS.monochrome, false),
      setPref(PREF_KEYS.bigCursor, false),
    ]);
  }
}

/** One per app, same reasoning as @kalakriti/i18n's `locale`: no SSR, no leak risk. */
export const a11y = new A11yState();

export const TEXT_SCALE_STEPS: readonly TextScale[] = ['normal', 'large', 'xlarge', 'xxlarge'];
