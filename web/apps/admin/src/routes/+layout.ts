// Static SPA: no Node runtime in production, so nothing renders on a server.
// Public, crawlable listing and artisan pages are rendered by the Go BFF from
// the same data -- see Batch 11 -- which is why SSR here would add a runtime
// dependency without adding a crawlable page.
export const ssr = false;
export const prerender = false;
export const trailingSlash = 'never';
