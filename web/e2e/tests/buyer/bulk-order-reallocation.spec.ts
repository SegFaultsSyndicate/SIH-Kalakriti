import { expect, test } from '@playwright/test';

test('a real BFF dropout is narrated and retains the replacement lot', async ({ page }) => {
  const orderId = 'order-reallocation-fixture';
  await page.route(`**/api/v1/orders/${orderId}`, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: orderId,
        quantity: 20,
        state: 'PARTIALLY_ALLOCATED',
        lots: [
          {
            id: 'lot-original',
            artisan_id: 'artisan-original',
            quantity: 10,
            state: 'REALLOCATED',
            progress_pct: 0,
          },
          {
            id: 'lot-replacement',
            artisan_id: 'artisan-replacement',
            quantity: 10,
            state: 'OFFERED',
            progress_pct: 0,
            reallocated_from_lot_id: 'lot-original',
          },
        ],
      }),
    }),
  );
  await page.route('**/api/v1/artisans/*/storefront', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ display_name: 'Meera Devi', district: 'Jaipur' }),
    }),
  );
  await page.route(`**/api/v1/orders/${orderId}/events**`, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'text/event-stream',
      body: [
        'id: fixture-gave-up',
        'data: {"event_id":"fixture-gave-up","bulk_order_id":"order-reallocation-fixture","type":"lot_gave_up","lot":{"id":"lot-original","artisan_id":"artisan-original","quantity":10,"state":"REALLOCATED","progress_pct":0}}',
        '',
        'id: fixture-offered',
        'data: {"event_id":"fixture-offered","bulk_order_id":"order-reallocation-fixture","type":"lot_offered","lot":{"id":"lot-replacement","artisan_id":"artisan-replacement","quantity":10,"state":"OFFERED","progress_pct":0,"reallocated_from_lot_id":"lot-original"}}',
        '',
      ].join('\n'),
    }),
  );

  await page.goto(`/orders/${orderId}`);
  await expect(page.locator('.alloc-live')).toBeVisible();
  await expect(page.locator('.alloc-lot__state', { hasText: 'REALLOCATED' })).toBeVisible();
  await expect(page.getByText(/reallocated from/i)).toBeVisible();
  await expect(page.locator('[aria-live="polite"]')).toContainText(/gave up|reallocat/i);
});
