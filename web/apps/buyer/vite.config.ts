// apps/buyer/vite.config.ts
import svg from '@poppanator/sveltekit-svg';
import { sveltekit } from '@sveltejs/kit/vite';
import { SvelteKitPWA } from '@vite-pwa/sveltekit';
import { searchForWorkspaceRoot } from 'vite';
import { defineConfig } from 'vitest/config';
import { BACKGROUND_COLOR, THEME_COLOR } from '@kalakriti/tokens/brand';
import { sizeReport } from '../../scripts/vite/size-report.js';
import { WORKSPACE_PACKAGES, vendorChunks } from '../../scripts/vite/workspace.js';

/**
 * Initial-JS budget, gzip. Enforced on every build; `SIZE_BUDGET_ENFORCE=0`
 * downgrades it to a report. See the artisan app's
 * vite.config.ts for the same convention.
 */
const BUYER_JS_BUDGET_KB = 200;

/** Cache-busts the precached shell on every build. */
const BUILD_ID = Date.now().toString(36);

export default defineConfig({
  plugins: [
    sveltekit(),

    svg({
      // The asset packs' SVGs are already SVGO-optimised by
      // scripts/svgo.config.mjs and every one of them is authored
      // against currentColor and --k-* custom properties. Re-running
      // the default SVGO preset here would strip the viewBox and
      // inline currentColor as black, so this pass is deliberately
      // narrow: it only removes the XML prolog the generators emit.
      svgoOptions: {
        multipass: false,
        plugins: [{ name: 'removeXMLProcInst' }],
      },
    }),

    SvelteKitPWA({
      // autoUpdate is right here and wrong in artisan: a buyer browsing a
      // catalogue loses nothing to a silent worker swap, and stale product
      // and price data is a real problem.
      registerType: 'autoUpdate',
      strategies: 'generateSW',
      // Registered from the layout instead: the plugin's auto-injected script
      // tag lands in prerendered HTML, and this app prerenders nothing.
      injectRegister: null,
      scope: '/',
      base: '/',

      manifest: {
        id: '/',
        name: 'Kalakriti — Handmade, with proof',
        short_name: 'Kalakriti',
        description:
          'Buy directly from Indian artisans, with verified provenance for every piece.',
        start_url: '/',
        scope: '/',
        display: 'standalone',
        lang: 'en-IN',
        dir: 'ltr',
        background_color: BACKGROUND_COLOR,
        theme_color: THEME_COLOR,
        icons: [
          {
            src: '/icons/app-icon-192.png',
            sizes: '192x192',
            type: 'image/png',
            purpose: 'any',
          },
          {
            src: '/icons/app-icon-512.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'any',
          },
          {
            src: '/icons/app-icon-maskable-512.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'maskable',
          },
        ],
      },

      workbox: {
        // adapter-static writes the fallback page after this plugin has already
        // generated the precache manifest, so index.html is not in the glob and
        // the NavigationRoute below would be bound to a URL workbox has never
        // cached -- which throws at worker startup and leaves the app with no
        // offline navigation at all. Declaring it explicitly makes workbox
        // fetch and cache it on install. It is keyed on '/' rather than
        // '/index.html' because that is the URL the shell is actually served
        // at -- a static file server maps / to index.html and 404s the
        // explicit name, so precaching '/index.html' fails install and the
        // browser silently throws the whole registration away.
        // The revision is the build id, so a redeploy replaces the cached
        // shell rather than serving a stale one.
        additionalManifestEntries: [{ url: '/', revision: BUILD_ID }],
        // Shell only. The buyer is on a real connection and the catalogue is
        // large and changes constantly; precaching product media would burn
        // storage on data that is stale before it is read.
        globPatterns: ['**/*.{js,css,html,webmanifest}'],
        globIgnores: ['**/size-report.json', '**/*.map'],
        navigateFallback: '/',
        navigateFallbackDenylist: [/^\/api\//],
        cleanupOutdatedCaches: true,
        clientsClaim: true,
        skipWaiting: true,
      },

      devOptions: {
        enabled: process.env.PWA_DEV === '1',
        type: 'module',
        suppressWarnings: true,
      },
      kit: {
        trailingSlash: 'never',
        // adapter-static runs AFTER this plugin, so the fallback page does not
        // exist yet when the precache manifest is written. Naming it here is
        // what gets it into the precache -- without this the shell is not
        // cached and the app cannot boot with no network, however many JS
        // chunks the worker has stored.
        adapterFallback: 'index.html',
      },
    }),

    sizeReport({
      app: 'buyer',
      budgetKb: BUYER_JS_BUDGET_KB,
      // Enforced by default: a budget that only fails when someone remembers to
      // set an env var is a report, not a budget. SIZE_BUDGET_ENFORCE=0 is the
      // escape hatch for a local build you knowingly want to finish anyway.
      enforce: process.env.SIZE_BUDGET_ENFORCE !== '0',
    }),
  ],

  build: {
    target: 'es2020',
    cssTarget: 'chrome80',
    sourcemap: true,
    reportCompressedSize: false,
    rollupOptions: { output: { manualChunks: vendorChunks } },
  },

  optimizeDeps: { exclude: WORKSPACE_PACKAGES },

  server: {
    port: 5174,
    strictPort: false,
    fs: {
      allow: [searchForWorkspaceRoot(process.cwd())],
    },
    proxy: {
      '/api': {
        target: 'http://localhost:8000',
        changeOrigin: true,
      },
      // routes/verify/[code]/+page.svelte fetches /v/{code}/verify.json
      // directly (not through the typed API client, since it's a public,
      // unauthenticated bff route with its own JSON shape) -- without this,
      // pnpm dev:buyer never reaches the bff for it: the request hits vite's
      // own dev server, which has no route for /v/*, and returns its dev
      // error page instead of proxying through like /api already does.
      '/v': {
        target: 'http://localhost:8000',
        changeOrigin: true,
      },
    },
  },
  preview: { port: 4174, strictPort: true },

  /*
   * Under vitest, resolve Svelte through its BROWSER export condition.
   * Without this, svelte resolves to its server build and every component
   * render throws `mount(...) is not available on the server`. Guarded on
   * VITEST so the real build is untouched -- SvelteKit still runs an SSR
   * pass over the app during `vite build` and must keep the server
   * condition there.
   */
  resolve: process.env.VITEST ? { conditions: ['browser'] } : {},

  test: { environment: 'jsdom', include: ['src/**/*.{test,spec}.{js,ts}'] },
});
