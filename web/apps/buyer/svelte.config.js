// apps/buyer/svelte.config.js
//
// Separate build from artisan and admin. See apps/artisan/svelte.config.js for
// the reasoning; the short version is that the artisan bundle must not carry
// marketplace or dashboard code, and shared chunks in a single build would
// guarantee that it did.

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
