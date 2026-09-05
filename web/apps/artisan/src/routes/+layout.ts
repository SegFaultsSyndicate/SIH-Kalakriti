// apps/artisan/src/routes/+layout.ts
//
// There is no Node runtime in production: the Go BFF serves this build
// directory as static files and rewrites unknown paths to index.html. So the
// whole app is client-rendered, and every route inherits these three.
//
// SEO-visible pages (public listing and artisan profile pages) are rendered
// server-side by the BFF from the same data, separately -- see Batch 11. That
// is why turning SSR on here would not buy the crawler anything and would cost
// a Node process this deployment does not have.

/** No server render. */
export const ssr = false;

/** No build-time render either: every screen is behind a session. */
export const prerender = false;

/** SPA fallback. */
export const trailingSlash = 'never';
