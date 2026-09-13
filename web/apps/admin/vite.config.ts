// apps/admin/vite.config.ts
import svg from '@poppanator/sveltekit-svg';
import { sveltekit } from '@sveltejs/kit/vite';
import { searchForWorkspaceRoot } from 'vite';
import { defineConfig } from 'vitest/config';
import { sizeReport } from '../../scripts/vite/size-report.js';
import { WORKSPACE_PACKAGES, vendorChunks } from '../../scripts/vite/workspace.js';

/** Initial-JS budget, gzip. Enforced on every build; SIZE_BUDGET_ENFORCE=0 downgrades it to a report. */
const ADMIN_JS_BUDGET_KB = 80;

// No SvelteKitPWA import at all: absence of a service worker is a decision
// here, not an omission. See svelte.config.js.
export default defineConfig({
  plugins: [sveltekit(), svg({
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
    }), sizeReport({
      app: 'admin',
      // Internal tool on office connections, so a looser ceiling than the two
      // public apps -- but a ceiling. Currently ~35 KB; this is the headroom
      // before someone should have to argue for it.
      budgetKb: ADMIN_JS_BUDGET_KB,
      enforce: process.env.SIZE_BUDGET_ENFORCE !== '0',
    })],

  build: {
    // Same floor as the other two so the shared packages compile once against
    // one target, even though this app's browsers could take more.
    target: 'es2020',
    cssTarget: 'chrome80',
    sourcemap: true,
    reportCompressedSize: false,
    rollupOptions: { output: { manualChunks: vendorChunks } },
  },

  optimizeDeps: { exclude: WORKSPACE_PACKAGES },

  server: {
    port: 5175,
    strictPort: false,
    fs: {
      allow: [searchForWorkspaceRoot(process.cwd())],
    },
    proxy: {
      '/api': {
        target: 'http://localhost:8000',
        changeOrigin: true,
      },
    },
  },
  preview: { port: 4175, strictPort: true },

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
