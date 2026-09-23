// apps/artisan/vite.config.ts
import svg from '@poppanator/sveltekit-svg';
import { sveltekit } from '@sveltejs/kit/vite';
import { SvelteKitPWA } from '@vite-pwa/sveltekit';
import { searchForWorkspaceRoot } from 'vite';
import { defineConfig } from 'vitest/config';
import { BACKGROUND_COLOR, THEME_COLOR } from '@kalakriti/tokens/brand';
import { respondUnavailableOnProxyError } from '../../scripts/vite/proxy-error.js';
import { sizeReport } from '../../scripts/vite/size-report.js';
import { WORKSPACE_PACKAGES, vendorChunks } from '../../scripts/vite/workspace.js';

/**
 * Initial-JS budget, gzip. Recorded in web/README.md with the measurement that
 * produced it. Enforced on every build; `SIZE_BUDGET_ENFORCE=0` downgrades it to
 * a build failure -- that is what CI runs.
 */
const ARTISAN_JS_BUDGET_KB = 120;

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
      // NEVER autoUpdate here. An auto-reload while an artisan is part-way
      // through photographing a piece throws away unsaved capture state. The
      // prompt is shown by PwaUpdatePrompt.svelte and is dismissible.
      registerType: 'prompt',
      strategies: 'generateSW',
      injectRegister: null,
      scope: '/',
      base: '/',

      manifest: {
        id: '/',
        name: 'Kalakriti — Artisan',
        short_name: 'Kalakriti',
        description:
          'Photograph your work, price it fairly, and reach buyers directly. Works without internet.',
        start_url: '/',
        scope: '/',
        display: 'standalone',
        orientation: 'portrait',
        lang: 'hi-IN',
        dir: 'ltr',
        background_color: BACKGROUND_COLOR,
        theme_color: THEME_COLOR,
        categories: ['business', 'shopping', 'productivity'],
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
            src: '/icons/app-icon-maskable-192.png',
            sizes: '192x192',
            type: 'image/png',
            purpose: 'maskable',
          },
          {
            src: '/icons/app-icon-maskable-512.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'maskable',
          },
        ],
        shortcuts: [
          {
            name: 'नया सामान जोड़ें',
            short_name: 'नया सामान',
            description: 'Photograph a new piece and start a listing',
            url: '/listing/new/capture',
            icons: [{ src: '/icons/app-icon-192.png', sizes: '192x192' }],
          },
          {
            name: 'मेरे ऑर्डर',
            short_name: 'ऑर्डर',
            description: 'Orders waiting for your reply',
            url: '/orders',
            icons: [{ src: '/icons/app-icon-192.png', sizes: '192x192' }],
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
        // The shell, the fonts, the icons and the message catalogues. NOTE:
        // this comment used to claim the JS glob below "picks up the active
        // one [locale chunk] on first visit and leaves the rest to runtime
        // caching" -- unverified, and almost certainly wrong: globPatterns
        // enumerates the build OUTPUT directory at BUILD time, with no
        // runtime concept of which language a given install ever selects,
        // so it is likely every locale's chunk gets swept into the
        // precache, not just one. Left as-is rather than "fixed" blind --
        // confirm against a real build's sw.js precache manifest (grep for
        // messages/<code> chunk names) before touching globPatterns itself,
        // since excluding the inactive ones for real needs a manifest
        // transform, not a comment edit. See I18N_PLAN.md's F-5.
        /*
         * woff2 is deliberately NOT in this list. Anek Latin ships three
         * subsets totalling 90.7KB and only the `latin` one (43.7KB) is
         * ever reached by Indian content -- precaching all three would put
         * 90.7KB of mostly-unused font on the install path over 2G, and
         * workbox's globIgnores is not honoured through this plugin
         * (verified: the entries came back anyway). The runtime CacheFirst
         * font rule below covers them instead, with a one-year expiry: the
         * first paint uses the system fallback via font-display: swap, the
         * face arrives once, and every later visit is offline-capable.
         */
        globPatterns: ['**/*.{js,css,html,svg,png,webp,json,webmanifest}'],
        globIgnores: ['**/size-report.json', '**/*.map'],
        // A capture screen's queued photos live in IndexedDB, not here; 4MB is
        // for the shell, which should never approach it.
        maximumFileSizeToCacheInBytes: 4 * 1024 * 1024,
        navigateFallback: '/',
        navigateFallbackDenylist: [/^\/api\//],
        cleanupOutdatedCaches: true,
        clientsClaim: false, // pairs with registerType: 'prompt'
        skipWaiting: false,

        runtimeCaching: [
          {
            // API reads. Network first with a short leash: on 2G a request
            // that has not answered in 3s is worse than yesterday's data, and
            // the artisan can see the age of what they are looking at.
            urlPattern: ({ url, request }) =>
              request.method === 'GET' && url.pathname.startsWith('/api/v1'),
            handler: 'NetworkFirst',
            options: {
              cacheName: 'kalakriti-api',
              networkTimeoutSeconds: 3,
              expiration: { maxEntries: 200, maxAgeSeconds: 60 * 60 * 24 },
              cacheableResponse: { statuses: [200] },
              matchOptions: { ignoreVary: true },
            },
          },
          {
            // Craft photographs. Cache first -- a photo at a given URL is
            // immutable, and re-validating it costs data the artisan pays for.
            urlPattern: ({ request, url }) =>
              (request.destination === 'image' || request.destination === 'video') &&
              !url.pathname.startsWith('/icons/'),
            handler: 'CacheFirst',
            options: {
              cacheName: 'kalakriti-media',
              // 60 entries is roughly two screens of thumbnails plus the
              // artisan's own catalogue; past that, oldest out.
              expiration: {
                maxEntries: 60,
                maxAgeSeconds: 60 * 60 * 24 * 30,
                purgeOnQuotaError: true,
              },
              cacheableResponse: { statuses: [0, 200] },
              rangeRequests: true,
            },
          },
          {
            urlPattern: ({ request }) => request.destination === 'font',
            handler: 'CacheFirst',
            options: {
              cacheName: 'kalakriti-fonts',
              expiration: { maxEntries: 24, maxAgeSeconds: 60 * 60 * 24 * 365 },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
        ],
      },

      devOptions: {
        // Off by default: a service worker in dev caches the very thing you
        // are editing. Turn on with PWA_DEV=1 to test install and offline.
        enabled: process.env.PWA_DEV === '1',
        type: 'module',
        navigateFallback: '/',
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
      app: 'artisan',
      budgetKb: ARTISAN_JS_BUDGET_KB,
      // Enforced by default: a budget that only fails when someone remembers to
      // set an env var is a report, not a budget. SIZE_BUDGET_ENFORCE=0 is the
      // escape hatch for a local build you knowingly want to finish anyway.
      enforce: process.env.SIZE_BUDGET_ENFORCE !== '0',
    }),
  ],

  build: {
    // es2020 is the floor for the Android WebView range in the field. Higher
    // would emit syntax those devices cannot parse; lower would ship
    // transpiler helpers nobody needs.
    target: 'es2020',
    cssTarget: 'chrome80',
    sourcemap: true,
    reportCompressedSize: false, // the size reporter already does this, better
    rollupOptions: {
      output: { manualChunks: vendorChunks },
    },
  },

  optimizeDeps: {
    // Workspace packages are source, not builds: esbuild pre-bundling would
    // mangle runes in .svelte.ts. See scripts/vite/workspace.js.
    exclude: WORKSPACE_PACKAGES,
  },

  server: {
    port: 5173,
    strictPort: false,
    fs: {
      allow: [searchForWorkspaceRoot(process.cwd())],
    },
    // The dev server runs inside WSL2 against the project on the Windows
    // filesystem (/mnt/c/...) -- chokidar's default inotify-based watching
    // does not see writes DrvFs makes from the Windows side, so an edit made
    // from a Windows editor/tool never triggers HMR and the server keeps
    // serving stale compiled output indefinitely (confirmed: curling a route
    // after an edit showed the pre-edit code). Polling reads file mtimes
    // instead of relying on inotify events, so it works across the DrvFs
    // boundary. Native Linux checkouts pay a small, harmless CPU cost for
    // this; there is no reliable way to detect "am I on DrvFs" to gate it.
    watch: {
      usePolling: true,
      interval: 300,
    },
    proxy: {
      '/api': {
        target: 'http://localhost:8000',
        changeOrigin: true,
        configure: respondUnavailableOnProxyError,
      },
    },
  },
  preview: { port: 4173, strictPort: true },

  /*
   * Under vitest, resolve Svelte through its BROWSER export condition.
   * Without this, svelte resolves to its server build and every component
   * render throws `mount(...) is not available on the server`. Guarded on
   * VITEST so the real build is untouched -- SvelteKit still runs an SSR
   * pass over the app during `vite build` and must keep the server
   * condition there.
   */
  resolve: process.env.VITEST ? { conditions: ['browser'] } : {},

  test: {
    environment: 'jsdom',
    include: ['src/**/*.{test,spec}.{js,ts}'],
    setupFiles: ['./src/test-setup.ts'],
  },
});
