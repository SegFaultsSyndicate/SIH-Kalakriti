/*
 * @kalakriti/icons — Batch 4
 *
 * Two ways to use an icon, and they cost differently:
 *
 *   import { Icon } from '@kalakriti/icons';        <Icon name="search" />
 *   import Search from '@kalakriti/icons/src/search.svg';   direct, tree-shakes
 *
 * <Icon name> is the convenience path and pulls in the full ~90-icon set
 * (icons.js imports every component so the name lookup can dispatch on any
 * of them). A direct .svg import tree-shakes to just that icon. Batch 8
 * proves the actual bundle number; this package just keeps the two paths
 * separate so a direct import doesn't silently regress.
 *
 * Import icons.css once, at the app root, for sizing, the stroke-weight
 * variants, the spinner animation and the upload-zone states.
 */

export { default as Icon } from './Icon.svelte';
export { default as Spinner } from './Spinner.svelte';
export { default as UploadZone } from './UploadZone.svelte';
export { ICONS } from './manifest.js';
