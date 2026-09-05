// apps/admin/svelte.config.js
//
// Desktop dashboard for ministry officials and cluster development officers.
// No service worker: it is used on managed desktops on an office connection,
// where offline support buys nothing and a stale cache of policy data is an
// actual hazard. See apps/artisan/svelte.config.js for why these are three
// builds rather than three route groups.

import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
export default {
  preprocess: vitePreprocess(),

  kit: {
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

    serviceWorker: { register: false },
  },
};
