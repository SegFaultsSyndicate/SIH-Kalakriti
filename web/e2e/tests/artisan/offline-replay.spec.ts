import { expect, test } from '@playwright/test';

test('offline mutations queue in outbox and display in offline queue screen', async ({
  page,
  context,
}) => {
  // Establish session and locale
  await page.goto('/language');
  await page.evaluate(() => {
    localStorage.setItem('kalakriti_locale', 'en');
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
    const payload = btoa(JSON.stringify({ sub: 'artisan-e2e-2', role: 'ARTISAN', exp: Math.floor(Date.now() / 1000) + 3600 }));
    localStorage.setItem('kalakriti_access_token', `${header}.${payload}.sig`);
  });

  // Cut network
  await context.setOffline(true);

  // Seed an outbox entry representing an offline action
  const entryId = 'offline-action-1';
  await page.evaluate(async (id) => {
    await new Promise<void>((resolve, reject) => {
      const request = indexedDB.open('kalakriti', 2);
      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        const tx = request.result.transaction(['outbox'], 'readwrite');
        tx.oncomplete = () => resolve();
        tx.onerror = () => reject(tx.error);
        tx.objectStore('outbox').put({
          id,
          kind: 'listing.update',
          payload: { title: 'Clay Pot Updated Offline' },
          mediaIds: [],
          attempts: 0,
          createdAt: Date.now(),
        });
      };
    });
  }, entryId);

  // Navigate to offline screen
  await page.goto('/offline');
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();

  // Verify the queued mutation is visible to the artisan
  await expect(page.getByText(/listing\.update|update|pending/i)).toBeVisible();

  // Restore network
  await context.setOffline(false);
});
