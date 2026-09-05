import { expect, test } from '@playwright/test';

test('buyer search to listing view and placing bulk order', async ({ page }) => {
  const listingId = 'listing-order-fixture';
  const orderId = 'order-created-fixture';

  // Intercept search results
  await page.route('**/api/v1/search**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        results: [
          {
            listing_id: listingId,
            matched_terms: ['madhubani', 'painting'],
            explanation: 'Direct craft match',
          },
        ],
        understood: { colours: [], materials: [] },
        detected_language: 'en',
      }),
    }),
  );

  // Intercept listing summary
  await page.route(`**/api/v1/listings/${listingId}/summary`, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: listingId,
        craft_name: 'Madhubani Painting',
        craft_slug: 'madhubani',
        artisan_name: 'Lakshmi Devi',
        type: 'MADE_TO_ORDER',
        price: { amount_paise: 250000, currency: 'INR' },
        min_order_quantity: 5,
        made_to_order_terms: {
          lead_time_days: 14,
          capacity_per_month: 20,
          advance_pct: 30,
        },
        translations: [{ language: 'en', title: 'Handmade Madhubani Folk Art' }],
      }),
    }),
  );

  // Intercept bulk order placement
  await page.route('**/api/v1/orders/bulk', (route) =>
    route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify({
        id: orderId,
        status: 'created',
      }),
    }),
  );

  // Intercept order view
  await page.route(`**/api/v1/orders/${orderId}`, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: orderId,
        quantity: 5,
        state: 'ALLOCATING',
        lots: [
          {
            id: 'lot-1',
            artisan_id: 'artisan-1',
            quantity: 5,
            state: 'OFFERED',
            progress_pct: 0,
          },
        ],
      }),
    }),
  );

  await page.route(`**/api/v1/orders/${orderId}/events**`, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'text/event-stream',
      body: '',
    }),
  );

  // 1. Visit search page and locate product
  await page.goto('/search?q=madhubani');
  const cardLink = page.locator(`a[href="/listing/${listingId}"]`);
  await expect(cardLink).toBeVisible();

  // 2. Click through to product detail page
  await cardLink.click();
  await expect(page).toHaveURL(`/listing/${listingId}`);
  await expect(page.getByText('Handmade Madhubani Folk Art')).toBeVisible();

  // 3. Place order via PurchaseForm
  const placeOrderBtn = page.getByRole('button', { name: /place order/i });
  await expect(placeOrderBtn).toBeVisible();
  await placeOrderBtn.click();

  // 4. Confirmed redirection to order allocation tracker
  await expect(page).toHaveURL(`/orders/${orderId}`);
  await expect(page.locator('.alloc-live, .order-view, [role="main"]')).toBeVisible();
});
