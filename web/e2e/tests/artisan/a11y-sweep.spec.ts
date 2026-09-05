// e2e/tests/artisan/a11y-sweep.spec.ts
//
// Batch 14: axe-core across every reachable route, not just /stories.
//
// Only routes reachable with NO session and NO registration are listed here
// -- the route guard (see $lib/route-guard.ts) redirects anything past
// /login to /language otherwise, and a redirected page's axe result would
// be testing /language again under a different name, not the route it
// claims to check. Everything past onboarding (/listings, /orders,
// /profile, /listing/new/*, /listings/[id], /orders/[orderId]/...) needs an
// authenticated, registered artisan with real listing/order data -- there
// is no seed-data endpoint yet (see web/DEMO.md) to put a test in that
// state deterministically, so those routes are NOT swept here. They are
// listed, unchecked, in web/ACCESSIBILITY.md instead of being silently
// skipped: a route this suite never mentions looks covered when it isn't,
// which is worse than an honest gap.
import AxeBuilder from '@axe-core/playwright';
import { expect, test, type Page } from '@playwright/test';

const THEMES = ['light', 'dark', 'high-contrast'] as const;

const ROUTES = ['/', '/welcome', '/language', '/login', '/accessibility', '/offline', '/stories'];

async function setTheme(page: Page, theme: (typeof THEMES)[number]): Promise<void> {
  await page.evaluate((value) => {
    document.documentElement.dataset.theme = value;
  }, theme);
}

for (const route of ROUTES) {
  for (const theme of THEMES) {
    test(`${route} has zero axe violations in ${theme}`, async ({ page }) => {
      await page.goto(route);
      await setTheme(page, theme);
      const results = await new AxeBuilder({ page }).analyze();
      expect(results.violations, JSON.stringify(results.violations, null, 2)).toEqual([]);
    });
  }
}
