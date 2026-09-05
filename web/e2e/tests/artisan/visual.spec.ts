// e2e/tests/artisan/visual.spec.ts
//
// Visual regression baselines: the stories page (packages/ui's own gallery)
// across all three themes. Screenshots are inherently platform-specific
// (font hinting, subpixel rounding differ Linux vs macOS vs Windows) --
// Playwright's own guidance is to generate baselines in the same
// environment that will compare against them. This session has no
// Playwright browser binaries installed (no network to fetch them), so no
// baseline PNGs are committed alongside this spec. Run once in CI (or any
// Linux box matching the `e2e` job's runner) with:
//
//   pnpm -F @kalakriti/e2e exec playwright install --with-deps chromium
//   pnpm e2e -- --update-snapshots
//
// and commit the resulting `*-snapshots/` directory. Tracked in
// web/ACCESSIBILITY.md until that first run happens.
import { expect, test, type Page } from '@playwright/test';

const THEMES = ['light', 'dark', 'high-contrast'] as const;

async function setTheme(page: Page, theme: (typeof THEMES)[number]): Promise<void> {
  await page.evaluate((value) => {
    document.documentElement.dataset.theme = value;
  }, theme);
}

for (const theme of THEMES) {
  test(`stories page visual baseline (${theme})`, async ({ page }) => {
    await page.goto('/stories');
    await setTheme(page, theme);
    await expect(page).toHaveScreenshot(`stories-${theme}.png`, { fullPage: true });
  });
}
