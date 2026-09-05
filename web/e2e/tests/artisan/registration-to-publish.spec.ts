import { expect, test } from '@playwright/test';

test('artisan registration through listing terms and publish', async ({ page }) => {
  // Pre-seed chosen locale and mock session token
  await page.goto('/language');
  await page.evaluate(() => {
    localStorage.setItem('kalakriti_locale', 'en');
    // A minimal valid JWT payload with RoleArtisan
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
    const payload = btoa(JSON.stringify({ sub: 'artisan-e2e-1', role: 'ARTISAN', exp: Math.floor(Date.now() / 1000) + 3600 }));
    const token = `${header}.${payload}.signature`;
    localStorage.setItem('kalakriti_access_token', token);
  });

  // Navigate to first registration step
  await page.goto('/register/name');
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();

  // Step 1: Name
  const nameInput = page.locator('input').first();
  await nameInput.fill('Lakshmi Devi');
  await page.getByRole('button', { name: /next|continue/i }).click();

  // Step 2: Craft
  await expect(page).toHaveURL(/\/register\/craft/);
  await page.getByRole('button', { name: /next|continue/i }).click();

  // Step 3: District
  await expect(page).toHaveURL(/\/register\/district/);
  await page.getByRole('button', { name: /next|continue/i }).click();

  // Step 4: Pehchan
  await expect(page).toHaveURL(/\/register\/pehchan/);
  await page.getByRole('button', { name: /next|continue/i }).click();

  // Step 5: Cluster & Submit
  await expect(page).toHaveURL(/\/register\/cluster/);
  await page.getByRole('button', { name: /submit|finish/i }).click();

  // Registration completes and redirects to home dashboard
  await expect(page).toHaveURL(/\/$/);

  // Now test drafting & publishing flow
  const draftId = 'reg-publish-draft-1';
  await page.evaluate(async (id) => {
    await new Promise<void>((resolve, reject) => {
      const request = indexedDB.open('kalakriti', 2);
      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        const tx = request.result.transaction(['drafts'], 'readwrite');
        tx.oncomplete = () => resolve();
        tx.onerror = () => reject(tx.error);
        tx.objectStore('drafts').put({
          id,
          fields: {
            reviewApproved: true,
            type: 'MADE_TO_ORDER',
            translations: [{ language: 'en', title: 'Handmade Madhubani Painting' }],
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

  // Accept terms and publish
  await page.locator('input[type="checkbox"]').last().check();
  const publishBtn = page.getByRole('button', { name: /publish/i });
  await expect(publishBtn).toBeEnabled();
  await publishBtn.click();

  // Returns to home dashboard
  await expect(page).toHaveURL(/\/$/);

  // Verify outbox queued the publish intents
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
    expect.arrayContaining(['profile.update', 'listing.update', 'listing.submit', 'listing.approve']),
  );
});
