import { expect, test } from '@playwright/test';

test('publishing while offline queues the complete listing intent locally', async ({
  page,
  context,
}) => {
  const draftId = 'offline-publish-fixture';
  await page.goto('/');
  await page.evaluate(async (id) => {
    await new Promise<void>((resolve, reject) => {
      const request = indexedDB.open('kalakriti', 2);
      request.onerror = () => reject(request.error);
      request.onupgradeneeded = () => {
        const db = request.result;
        for (const [name, keyPath] of [
          ['drafts', 'id'],
          ['outbox', 'id'],
          ['media', 'id'],
          ['cache', 'key'],
          ['prefs', 'key'],
        ] as const) {
          if (!db.objectStoreNames.contains(name)) db.createObjectStore(name, { keyPath });
        }
      };
      request.onsuccess = () => {
        const tx = request.result.transaction('drafts', 'readwrite');
        tx.oncomplete = () => resolve();
        tx.onerror = () => reject(tx.error);
        tx.objectStore('drafts').put({
          id,
          fields: {
            reviewApproved: true,
            type: 'MADE_TO_ORDER',
            translations: [{ language: 'en', title: 'Offline Blue Pot' }],
          },
          mediaIds: [],
          createdAt: Date.now(),
          updatedAt: Date.now(),
        });
      };
    });
  }, draftId);
  await page.goto(`/listing/new/terms?d=${draftId}`);
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  await context.setOffline(true);
  await page.locator('input[type="checkbox"]').last().check();
  await page.getByRole('button', { name: /publish/i }).click();
  await expect(page).toHaveURL(/\/$/);
  const queuedKinds = await page.evaluate(async () => {
    const request = indexedDB.open('kalakriti', 2);
    return await new Promise<string[]>((resolve) => {
      request.onsuccess = () => {
        const tx = request.result.transaction('outbox', 'readonly');
        const get = tx.objectStore('outbox').getAll();
        get.onsuccess = () => resolve(get.result.map((entry: { kind: string }) => entry.kind));
      };
    });
  });
  expect(queuedKinds).toEqual(
    expect.arrayContaining(['listing.update', 'listing.submit', 'listing.approve']),
  );
  await context.setOffline(false);
});
