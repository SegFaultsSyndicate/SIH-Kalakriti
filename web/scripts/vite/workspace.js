// web/scripts/vite/workspace.js
//
// Shared vite wiring for all three apps.
//
// Workspace packages ship TypeScript and .svelte source, not a build. pnpm
// symlinks them into node_modules, which is enough for vite to mistake them
// for ordinary dependencies and hand them to esbuild for pre-bundling -- which
// does not understand runes and would silently produce a broken $state. They
// have to be excluded from optimizeDeps and kept un-externalised, in every
// app, or one app works and the next one fails mysteriously.

import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../../', import.meta.url));

/** Every workspace package an app may import. */
export const WORKSPACE_PACKAGES = [
  '@kalakriti/api',
  '@kalakriti/i18n',
  '@kalakriti/icons',
  '@kalakriti/identity',
  '@kalakriti/illustrations',
  '@kalakriti/motion',
  '@kalakriti/offline',
  '@kalakriti/ornament',
  '@kalakriti/patterns',
  '@kalakriti/tokens',
  '@kalakriti/ui',
  '@kalakriti/voice',
];

/**
 * Short aliases, mirroring tsconfig.json's `paths`. Note the absence of
 * `$lib`: SvelteKit reserves it for the app's own src/lib, and pointing it at
 * a shared package breaks per-app type generation.
 */
export const workspaceAliases = {
  $ui: `${root}packages/ui/src`,
  $api: `${root}packages/api/src`,
  $i18n: `${root}packages/i18n/src`,
  $offline: `${root}packages/offline/src`,
  $tokens: `${root}packages/tokens/src`,
  $icons: `${root}packages/icons`,
  $patterns: `${root}packages/patterns`,
  $ornament: `${root}packages/ornament`,
  $illustrations: `${root}packages/illustrations`,
  $motion: `${root}packages/motion`,
  $identity: `${root}packages/identity`,
};

/**
 * Split vendor code out of the app chunk.
 *
 * Whether this helps is an empirical question and the answer differs per app:
 * one shared vendor chunk caches well across route navigations but is loaded
 * in full up front, which can be worse for a first visit than SvelteKit's
 * default per-route splitting. Each app decides; the function is here so the
 * three cannot drift apart accidentally. Measured results are in README.md.
 */
export function vendorChunks(id) {
  // `NO_MANUAL_CHUNKS=1 pnpm build` falls back to rollup's own splitting, so
  // the claim in README.md that manual chunking helps can be re-checked in one
  // command instead of by editing three configs.
  if (process.env.NO_MANUAL_CHUNKS === '1') return undefined;
  if (!id.includes('node_modules')) return undefined;
  // Svelte's runtime is on every route's critical path; keeping it apart from
  // the rest of vendor means a dependency change does not invalidate it.
  if (id.includes('/svelte/') || id.includes('\\svelte\\')) return 'svelte';
  if (id.includes('dexie')) return 'dexie';
  return 'vendor';
}
