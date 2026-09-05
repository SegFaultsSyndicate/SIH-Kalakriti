import { expect, test } from '@playwright/test';

test('a sealed QR code opens the anonymous buyer verification page', async ({ page }) => {
  const code = 'sealed-fixture-42';
  await page.route(`**/v/${code}/verify.json`, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        short_code: code,
        content_hash: 'sha256:fixture-content',
        signature: 'fixture-signature',
        signature_algo: 'Ed25519',
        public_key_id: 'kalakriti-fixture-key',
        technique_matched: true,
        signature_valid: true,
        sealed_at: '2026-01-15T10:00:00Z',
        verify_url: `/v/${code}`,
        media_hashes: ['sha256:fixture-media'],
      }),
    }),
  );

  await page.goto(`/verify/${encodeURIComponent(code)}`);
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  await expect(page.getByText(code)).toBeVisible();
  await expect(page.locator('.verify__status--valid')).toBeVisible();
});
