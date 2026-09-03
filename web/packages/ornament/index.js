/*
 * @kalakriti/ornament — Batch 3
 *
 * Structural ornament: things that divide, edge and frame. Nothing here is a
 * sticker; if an asset from this package ends up floating in a corner as
 * decoration, it is being used wrong.
 *
 * Import the stylesheet once, at the app root:
 *   import '@kalakriti/ornament/ornament.css';
 *
 * The raw tiles in ./src are also usable directly:
 *   - as Svelte components via vite-plugin-svelte-svg
 *   - as SVG <pattern> definitions by inlining ./src/patterns.svg once per
 *     document and then referencing fill="url(#k-jaali-hex-24)"
 *
 * The CSS utilities and the <pattern> sprite both preserve currentColor.
 * A background-image or border-image does not — see ornament.css.
 */

export { default as Divider } from './Divider.svelte';
export { default as Rule } from './Rule.svelte';
export { default as KolamCorner } from './KolamCorner.svelte';
export { default as CardEdge } from './CardEdge.svelte';

/** Every divider variant, with its tile size. Useful for docs and tests. */
export const DIVIDERS = [
  { variant: 'jaali-hex', density: 'fine', tile: [14, 16], tradition: 'jaali (hexagonal)' },
  { variant: 'jaali-hex', density: 'medium', tile: [21, 24], tradition: 'jaali (hexagonal)' },
  { variant: 'jaali-hex', density: 'bold', tile: [28, 32], tradition: 'jaali (hexagonal)' },
  { variant: 'jaali-octstar', density: 'fine', tile: [16, 16], tradition: 'jaali (octagonal-star)' },
  { variant: 'jaali-octstar', density: 'medium', tile: [24, 24], tradition: 'jaali (octagonal-star)' },
  { variant: 'jaali-octstar', density: 'bold', tile: [32, 32], tradition: 'jaali (octagonal-star)' },
  { variant: 'jaali-interlace', density: 'fine', tile: [16, 16], tradition: 'jaali (interlaced-square)' },
  { variant: 'jaali-interlace', density: 'medium', tile: [24, 24], tradition: 'jaali (interlaced-square)' },
  { variant: 'jaali-interlace', density: 'bold', tile: [32, 32], tradition: 'jaali (interlaced-square)' },
  { variant: 'blockprint-running', density: 'medium', tile: [20, 20], tradition: 'block print (Ajrakh/Bagru)' },
  { variant: 'blockprint-band', density: 'medium', tile: [32, 32], tradition: 'block print (Ajrakh)' },
];
