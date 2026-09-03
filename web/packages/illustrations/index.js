/*
 * @kalakriti/illustrations — Batch 5
 *
 * import '@kalakriti/illustrations/illustrations.css'; once, at the app root
 * — it carries the accent-colour tokens, the hero-backdrop mask, and the
 * process-strip layout.
 *
 * <Illustration name> is the convenience path (pulls in the full ~30-scene
 * set); a direct `import Scene from '@kalakriti/illustrations/src/<name>.svg'`
 * tree-shakes to just that one. Same trade-off as @kalakriti/icons, for the
 * same reason — see that package's README.
 */

export { default as Illustration } from './Illustration.svelte';
export { default as ProcessSequence } from './ProcessSequence.svelte';
export { default as HeroBackdrop } from './HeroBackdrop.svelte';
export { ILLUSTRATIONS } from './manifest.js';
