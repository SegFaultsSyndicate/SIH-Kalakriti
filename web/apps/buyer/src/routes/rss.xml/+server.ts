// apps/buyer/src/routes/rss.xml/+server.ts
//
// RSS 2.0 feed of the Virasat Journal (footer "RSS Feed"). This app is a
// static SPA with no server in production, so the feed is prerendered to
// build/rss.xml at build time. Feeds carry one language; this one is the
// English catalogue's text for the same keys the essay pages render.
import { en } from '@kalakriti/i18n';
import { ESSAYS, essayKey } from '$lib/journal';
import type { RequestHandler } from './$types';

export const prerender = true;

const xml = (s: string): string =>
  s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

export const GET: RequestHandler = ({ url }) => {
  // Absolute links need the real site origin; at prerender time url.origin is
  // a placeholder host, so fall back to site-relative links unless configured.
  const origin = import.meta.env.VITE_SITE_URL ?? (url.hostname === 'sveltekit-prerender' ? '' : url.origin);
  const items = [...ESSAYS]
    .sort((a, b) => b.published.localeCompare(a.published))
    .map((e) => {
      const link = `${origin}/journal/${e.slug}`;
      return `    <item>
      <title>${xml(en[essayKey(e, 'title')])}</title>
      <link>${link}</link>
      <guid isPermaLink="true">${link}</guid>
      <pubDate>${new Date(e.published).toUTCString()}</pubDate>
      <author>${xml(en[essayKey(e, 'author')])}</author>
      <description>${xml(en[essayKey(e, 'lead')])}</description>
    </item>`;
    })
    .join('\n');

  const body = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>${xml(en['journal.breadcrumbLabel'])} — ${xml(en['app.name'])}</title>
    <link>${origin}/journal</link>
    <description>${xml(en['home.journal.subheading'])}</description>
    <language>en-IN</language>
${items}
  </channel>
</rss>
`;
  return new Response(body, { headers: { 'Content-Type': 'application/rss+xml; charset=utf-8' } });
};
