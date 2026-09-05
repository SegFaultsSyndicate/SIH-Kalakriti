// apps/artisan/svelte.config.js
//
// WHY THREE APPS AND NOT THREE ROUTE GROUPS
//
// The artisan app's user is on a 16-32GB Android phone over intermittent 2G,
// and the app is offline-first: its service worker precaches the entire shell.
// A single SvelteKit build with route groups would put admin's tables and
// charts and buyer's marketplace code in the same module graph, the same
// shared chunks, and the same precache manifest -- so every ministry-dashboard
// dependency would become bytes on an artisan's phone, and every admin deploy
// would invalidate the artisan's precache. Separate builds make that
// impossible by construction rather than by discipline.
//
// The cost is three configs and a shared packages/ directory, which is the
// cheap half of the trade.

import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
export default {
  preprocess: vitePreprocess(),

  kit: {
    // Static SPA. The Go BFF serves this build directory and rewrites unknown
    // paths to the fallback; there is no Node runtime in production, and SEO
    // pages are rendered by the BFF separately (Batch 11).
    adapter: adapter({
      pages: 'build',
      assets: 'build',
      fallback: 'index.html',
      precompress: true,
      strict: false,
    }),

    // Absolute asset paths. The default relative ones resolve against the
    // current URL, so the single fallback index.html served at /listings/new
    // would look for ./_app/ under /listings/ and 404. A SPA fallback needs
    // absolute paths.
    paths: { relative: false },

    alias: {
      // tsconfig paths satisfy the type-checker only; the bundler needs these.
      $ui: '../../packages/ui/src',
      $api: '../../packages/api/src',
      $i18n: '../../packages/i18n/src',
      $offline: '../../packages/offline/src',
      $tokens: '../../packages/tokens/src',
      $icons: '../../packages/icons',
      $patterns: '../../packages/patterns',
      $ornament: '../../packages/ornament',
      $illustrations: '../../packages/illustrations',
      $motion: '../../packages/motion',
      $identity: '../../packages/identity',
    },

    // @vite-pwa/sveltekit generates the worker; Kit's own must stay off or
    // both would register and fight over the same scope.
    serviceWorker: { register: false },

    version: {
      // Content-hashed so the update prompt can tell a real deploy from a
      // reload, without polling a version endpoint over 2G.
      pollInterval: 0,
    },
  },
};
