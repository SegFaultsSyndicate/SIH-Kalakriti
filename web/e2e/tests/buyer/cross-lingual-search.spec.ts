import { expect, test } from '@playwright/test';

test('search surfaces a real cross-lingual listing match', async ({ page }) => {
  const listingId = 'listing-cross-lingual-fixture';
  await page.route('**/api/v1/search**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        results: [
          {
            listing_id: listingId,
            machine_generated: true,
            matched_terms: ['नीला', 'घड़ा'],
            explanation: 'Matched a Hindi translation.',
          },
        ],
        understood: { colours: [], materials: [] },
        detected_language: 'hi',
      }),
    }),
  );
  await page.route(`**/api/v1/listings/${listingId}/summary`, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: listingId,
        craft_name: 'Blue Pottery',
        craft_slug: 'blue-pottery',
        artisan_name: 'Asha Devi',
        type: 'READY_STOCK',
        price: { amount_paise: 125000, currency: 'INR' },
        translations: [{ language: 'hi', title: 'नीला घड़ा' }],
      }),
    }),
  );

  await page.goto(
    '/search?q=%E0%A4%A8%E0%A5%80%E0%A4%B2%E0%A4%BE%20%E0%A4%98%E0%A4%A1%E0%A4%BC%E0%A4%BE',
  );
  await expect(page.locator(`a[href="/listing/${listingId}"]`)).toBeVisible();
  await expect(page.getByText('Matched across languages')).toBeVisible();
});
