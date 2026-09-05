import { expect, test } from '@playwright/test';

test('admin views insights dashboard and cluster management', async ({ page }) => {
  // Pre-seed admin session token with MINISTRY role
  await page.goto('/');
  await page.evaluate(() => {
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
    const payload = btoa(JSON.stringify({ sub: 'admin-1', role: 'MINISTRY', exp: Math.floor(Date.now() / 1000) + 3600 }));
    localStorage.setItem('kalakriti_access_token', `${header}.${payload}.sig`);
  });

  // Intercept insights API calls
  await page.route('**/api/v1/insights/artisans-by-category**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [
          { state_code: 'RJ', district: 'Jaipur', craft_id: 'blue-pottery', artisan_count: 42 },
        ],
      }),
    }),
  );

  await page.route('**/api/v1/insights/earnings-by-district**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [
          { state_code: 'RJ', district: 'Jaipur', total_earnings_paise: 8400000 },
        ],
      }),
    }),
  );

  await page.route('**/api/v1/insights/income-comparison**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [
          { state_code: 'RJ', district: 'Jaipur', platform_average_paise: 500000, outside_average_paise: 300000 },
        ],
      }),
    }),
  );

  await page.route('**/api/v1/insights/listings-by-craft-month**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [
          { craft_id: 'blue-pottery', year: 2026, month: 8, listing_count: 15 },
        ],
      }),
    }),
  );

  await page.route('**/api/v1/insights/dying-crafts**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [
          { craft_id: 'rogan-art', craft_name: 'Rogan Art', remaining_artisans: 4, trend_pct: -25 },
        ],
      }),
    }),
  );

  // 1. Visit insights page
  await page.goto('/insights');
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  // Filter or load data
  const applyBtn = page.getByRole('button', { name: /apply|load|refresh/i }).first();
  if (await applyBtn.isVisible()) {
    await applyBtn.click();
  }
  await expect(page.getByText(/Jaipur|Rogan Art/i)).toBeVisible();

  // Intercept cluster endpoints
  const clusterId = 'cluster-jaipur-1';
  await page.route(`**/api/v1/clusters/${clusterId}`, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: clusterId,
        name: 'Jaipur Blue Pottery Collective',
        state: 'RJ',
        district: 'Jaipur',
        created_at: '2026-01-01T00:00:00Z',
      }),
    }),
  );

  await page.route(`**/api/v1/clusters/${clusterId}/members`, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        members: [
          { artisan_id: 'artisan-1', display_name: 'Meera Devi', joined_at: '2026-02-01T00:00:00Z' },
        ],
      }),
    }),
  );

  // 2. Visit clusters management page
  await page.goto('/clusters');
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();

  // Search/load the cluster
  const idInput = page.locator('input').first();
  await idInput.fill(clusterId);
  const loadBtn = page.getByRole('button', { name: /load|find|search/i }).first();
  if (await loadBtn.isVisible()) {
    await loadBtn.click();
    await expect(page.getByText('Jaipur Blue Pottery Collective')).toBeVisible();
  }
});
