// packages/tokens/src/brand.js
//
// The handful of token values that have to exist outside CSS.
//
// A web app manifest is JSON generated at build time; it cannot read a
// CSS custom property. Without this file the three vite configs would
// each hardcode a hex, and the day the palette moves, two of them would
// be wrong. Values below are literal because they ARE the source for
// the manifest -- they mirror palette.css and are asserted equal to it
// by scripts/check-brand-tokens.mjs, which runs in `pnpm check`.

/** khadi-50 -- the light page ground. Splash background behind the icon. */
export const BACKGROUND_COLOR = '#FCFAF6';

/** terracotta-700 -- primary accent. Colours the OS status/title bar. */
export const THEME_COLOR = '#963D14';

/** ink-950 -- dark page ground, for the dark-scheme manifest entry. */
export const BACKGROUND_COLOR_DARK = '#0F0A07';

/** The token each value above is taken from, for the equality check. */
export const BRAND_TOKEN_SOURCES = {
  BACKGROUND_COLOR: '--k-khadi-50',
  THEME_COLOR: '--k-terracotta-700',
  BACKGROUND_COLOR_DARK: '--k-ink-950',
};
