import { expect, test, type Page } from '@playwright/test';

const themes = ['light', 'dark', 'high-contrast'] as const;

async function setTheme(page: Page, theme: (typeof themes)[number]): Promise<void> {
  await page.evaluate((value) => {
    document.documentElement.dataset.theme = value;
  }, theme);
}

for (const theme of themes) {
  test(`admin dashboard visual baseline (${theme})`, async ({ page }) => {
    await page.goto('/');
    await setTheme(page, theme);
    await expect(page).toHaveScreenshot(`admin-dashboard-${theme}.png`, { fullPage: true });
  });
}
