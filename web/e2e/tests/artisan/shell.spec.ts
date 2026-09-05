// e2e/tests/artisan/shell.spec.ts
//
// Batch 1's acceptance criteria, as a test rather than a claim: the app boots
// as a static SPA, the offline route is reachable and useful, and the PWA
// pieces the installability audit looks for are actually served.
//
// The default locale in a headless Chromium is English, so the assertions
// below match the English catalogue.
import { expect, test, type Page } from '@playwright/test';

/** Scope of the active worker, or null if there is not one yet. */
function activeScope(page: Page): Promise<string | null> {
  return page.evaluate(async () => {
    const registration = await navigator.serviceWorker.getRegistration();
    return registration?.active ? registration.scope : null;
  });
}

/**
 * Registration starts after hydration, so anything that depends on the worker
 * waits for it. Deliberately not navigator.serviceWorker.ready: with
 * clientsClaim off the first worker never takes control of the page that
 * registered it, and `ready` can sit unresolved for the life of the test.
 */
async function waitForServiceWorker(page: Page): Promise<string> {
  await expect
    .poll(() => activeScope(page), { timeout: 45_000, intervals: [250, 500, 1000] })
    .not.toBeNull();
  const scope = await activeScope(page);
  if (scope === null) throw new Error('service worker went away after activating');
  return scope;
}

test('the shell renders and the skip link is the first stop for a keyboard', async ({
  page,
}) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();

  await page.keyboard.press('Tab');
  const skip = page.locator('.k-skip-link');
  await expect(skip).toBeFocused();
  await expect(skip).toBeInViewport();
});

test('the offline route shows the queue and a way to keep working', async ({ page }) => {
  await page.goto('/offline');

  await expect(page.getByRole('heading', { name: 'Your work is safe' })).toBeVisible();

  // An empty queue says so; it does not render nothing.
  await expect(page.getByText('Nothing is waiting.')).toBeVisible();

  // The point of the screen: it is not a dead end.
  await expect(page.getByRole('link', { name: 'Photograph a new piece' })).toBeVisible();
});

test('the app still works with the network cut', async ({ page, context }) => {
  await page.goto('/');
  await waitForServiceWorker(page);

  // The worker installs but does not claim the page that registered it --
  // that is the deliberate consequence of clientsClaim: false. One reload puts
  // the page under its control, which is the state a returning artisan is in.
  await page.reload();
  await expect
    .poll(() => page.evaluate(() => navigator.serviceWorker.controller !== null))
    .toBe(true);

  await context.setOffline(true);
  await page.goto('/offline');
  await expect(page.getByRole('heading', { name: 'Your work is safe' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Photograph a new piece' })).toBeVisible();
  await context.setOffline(false);
});

test('the manifest is served and declares what installability needs', async ({
  page,
  request,
}) => {
  await page.goto('/');
  // The link element matters as much as the file: without it the browser never
  // looks for a manifest at all.
  await expect(page.locator('link[rel="manifest"]')).toHaveAttribute(
    'href',
    /manifest\.webmanifest$/,
  );

  const response = await request.get('/manifest.webmanifest');
  expect(response.ok()).toBeTruthy();

  const manifest = await response.json();
  expect(manifest.name).toBeTruthy();
  expect(manifest.short_name).toBeTruthy();
  expect(manifest.start_url).toBeTruthy();
  expect(manifest.display).toBe('standalone');

  const png192 = manifest.icons.find(
    (icon: { sizes: string; type: string }) =>
      icon.sizes === '192x192' && icon.type === 'image/png',
  );
  expect(png192, 'installability requires a PNG icon of at least 192x192').toBeTruthy();

  const maskable = manifest.icons.filter((icon: { purpose?: string }) =>
    icon.purpose?.includes('maskable'),
  );
  expect(maskable.length).toBeGreaterThanOrEqual(2);

  expect(manifest.shortcuts.map((s: { url: string }) => s.url)).toEqual([
    '/listing/new/capture',
    '/orders',
  ]);
});

test('the service worker registers at the root scope', async ({ page, baseURL }) => {
  await page.goto('/');
  expect(await waitForServiceWorker(page)).toBe(`${baseURL}/`);
});

test.describe('Hindi, the default language', () => {
  // Everything above runs in the headless default (en-US), which is not the
  // locale the primary user ever sees. This block is the cheapest check that
  // the dynamic-catalogue mechanism works end to end: the hi chunk is
  // fetchable, it is applied, and <html lang> follows it -- which is what a
  // screen reader switches voice on.
  test.use({ locale: 'hi-IN' });

  test('loads the Hindi catalogue and sets the document language', async ({ page }) => {
    await page.goto('/offline');

    await expect(page.getByRole('heading', { name: 'आपका काम सुरक्षित है' })).toBeVisible();
    await expect(page.locator('html')).toHaveAttribute('lang', 'hi-IN');
    await expect(page.getByRole('link', { name: 'नई चीज़ की तस्वीर लें' })).toBeVisible();
  });
});
