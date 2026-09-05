import { expect, test } from '@playwright/test';

test('unauthorized request triggers session refresh and recovers', async ({ page }) => {
  let refreshCalled = false;
  let retryCount = 0;

  // Intercept the API routes
  await page.route('**/api/v1/auth/refresh', (route) => {
    refreshCalled = true;
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
    const payload = btoa(JSON.stringify({ sub: 'buyer-refreshed', role: 'BUYER', exp: Math.floor(Date.now() / 1000) + 3600 }));
    const newAccessToken = `${header}.${payload}.sig`;
    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        access_token: newAccessToken,
        token_type: 'Bearer',
        expires_in: 3600,
      }),
    });
  });

  await page.route('**/api/v1/crafts', (route) => {
    retryCount++;
    if (retryCount === 1) {
      // First attempt fails with 401
      return route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'unauthorized', message: 'Token expired' }),
      });
    }
    // Subsequent attempt after refresh succeeds
    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { id: 'craft-1', name: 'Blue Pottery', slug: 'blue-pottery' },
      ]),
    });
  });

  // Pre-seed expired access token and valid refresh token
  await page.goto('/');
  await page.evaluate(() => {
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
    const payload = btoa(JSON.stringify({ sub: 'buyer-old', role: 'BUYER', exp: Math.floor(Date.now() / 1000) - 100 }));
    localStorage.setItem('kalakriti_access_token', `${header}.${payload}.sig`);
    localStorage.setItem('kalakriti_refresh_token', 'valid-refresh-token');
  });

  // Trigger an API call from page context
  const recoveredData = await page.evaluate(async () => {
    const { listCrafts } = await import('@kalakriti/api');
    return await listCrafts();
  });

  expect(refreshCalled).toBe(true);
  expect(retryCount).toBe(2);
  expect(recoveredData).toHaveLength(1);
  expect(recoveredData[0].name).toBe('Blue Pottery');
});
