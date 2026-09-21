const { chromium } = require('@playwright/test');
const path = require('path');

const IMAGE = '/home/braine_dead/Pictures/hollowKnightIcon.png';
const BASE = 'http://localhost:8081';

async function dumpDexie(page, label) {
  const dump = await page.evaluate(async () => {
    return new Promise((resolve) => {
      const req = indexedDB.open('kalakriti'); // guess; will log actual DB names too
      req.onsuccess = () => resolve({ ok: true });
      req.onerror = () => resolve({ ok: false });
    });
  }).catch(() => null);
  console.log(label, 'dexie probe:', dump);
}

(async () => {
  const browser = await chromium.launch({ args: ['--use-fake-ui-for-media-stream', '--use-fake-device-for-media-stream'] });
  const context = await browser.newContext({ permissions: ['microphone'] });
  const page = await context.newPage();

  page.on('console', (msg) => console.log('CONSOLE', msg.type().toUpperCase() + ':', msg.text()));
  page.on('pageerror', (err) => console.log('PAGE ERROR:', err.message));

  await page.goto(BASE + '/', { waitUntil: 'networkidle' });
  console.log('step0 url:', page.url());
  if (page.url().includes('/language')) {
    await page.locator('button.language__select', { hasText: 'English' }).click();
    await page.waitForURL('**/welcome', { timeout: 10000 }).catch((e) => console.log('  no nav to /welcome:', e.message));
  }
  console.log('step1 url:', page.url());
  if (page.url().includes('/welcome')) {
    await page.getByRole('button', { name: 'Get started' }).click();
    await page.waitForURL('**/login', { timeout: 10000 }).catch((e) => console.log('  no nav to /login:', e.message));
  }
  console.log('step2 url:', page.url());
  if (page.url().includes('/login')) {
    await page.fill('#phone-input', '9999999999');
    await page.getByRole('button', { name: 'Send code' }).click();
    await page.waitForURL('**/verify', { timeout: 10000 }).catch((e) => console.log('  no nav to /verify:', e.message));
  }
  console.log('step3 url:', page.url());
  if (page.url().includes('/verify')) {
    const otpInputs = page.locator('input[inputmode="numeric"]');
    for (let i = 0; i < 6; i++) await otpInputs.nth(i).fill('0');
    for (let i = 0; i < 10; i++) {
      await page.waitForTimeout(500);
      console.log('  verify poll', i, page.url());
      if (!page.url().includes('/verify')) break;
    }
  }
  console.log('post-auth URL:', page.url());
  await page.screenshot({ path: '/home/braine_dead/.claude/jobs/1a02db01/tmp/shots/debug-post-verify.png', fullPage: true });

  await page.goto(BASE + '/listing/new/capture', { waitUntil: 'networkidle' });
  const draftId = new URL(page.url()).searchParams.get('d');
  console.log('draftId:', draftId);

  await page.locator('input[type="file"]').setInputFiles(IMAGE);
  await page.waitForTimeout(500);
  await page.getByRole('button', { name: /^Next/ }).click();
  await page.waitForURL('**/listing/new/studio*', { timeout: 15000 });

  const continueBtn = page.getByRole('button', { name: /Continue to Video/i });
  await continueBtn.waitFor({ state: 'visible', timeout: 15000 });
  await continueBtn.click();
  await page.waitForURL('**/listing/new/video*', { timeout: 15000 });

  await page.getByRole('button', { name: /Skip|Next/ }).click();
  await page.waitForLoadState('networkidle');
  console.log('URL after video:', page.url());

  // Story: pick craft. Options load async (GET /crafts) -- poll until non-empty.
  await page.waitForLoadState('networkidle');
  const craftSelect = page.locator('select').first();
  let optionValues = [];
  for (let i = 0; i < 20 && optionValues.length === 0; i++) {
    optionValues = await craftSelect.locator('option').evaluateAll((opts) => opts.map((o) => o.value).filter((v) => v !== ''));
    if (optionValues.length === 0) await page.waitForTimeout(300);
  }
  console.log('craft options found:', optionValues.length);
  if (optionValues.length) await craftSelect.selectOption(optionValues[0]);
  await page.waitForTimeout(300);

  const recordBtn = page.getByRole('button', { name: /record|voice/i }).first();
  if (await recordBtn.count()) {
    await recordBtn.click();
    await page.waitForTimeout(1500);
    const stopBtn = page.getByRole('button', { name: /stop|done/i }).first();
    if (await stopBtn.count()) await stopBtn.click();
    else await recordBtn.click();
    await page.waitForTimeout(1000);
  }

  // Dump IndexedDB right after craft pick, before Next.
  const dbNames = await page.evaluate(async () => {
    if (indexedDB.databases) {
      const dbs = await indexedDB.databases();
      return dbs.map((d) => d.name);
    }
    return ['(databases() unsupported)'];
  });
  console.log('IndexedDB databases:', dbNames);

  const storyNext = page.getByRole('button', { name: 'Next' });
  const storyNextDisabled = await storyNext.isDisabled();
  console.log('story Next disabled?', storyNextDisabled);
  if (!storyNextDisabled) {
    await storyNext.click();
    await page.waitForLoadState('networkidle');
  }
  console.log('URL after story next:', page.url());

  if (page.url().includes('/processing')) {
    // Let the sync engine actually attempt drains -- media.upload, then
    // listing.create, then listing.media.attach, each depends on the last.
    await page.waitForTimeout(20000);
  }

  // Now dump the outbox + media + drafts tables via Dexie's own DB.
  const dump = await page.evaluate(async (dbName) => {
    function openDb(name) {
      return new Promise((resolve, reject) => {
        const req = indexedDB.open(name);
        req.onsuccess = () => resolve(req.result);
        req.onerror = () => reject(req.error);
      });
    }
    function getAll(db, storeName) {
      return new Promise((resolve, reject) => {
        try {
          const tx = db.transaction(storeName, 'readonly');
          const store = tx.objectStore(storeName);
          const req = store.getAll();
          req.onsuccess = () => resolve(req.result);
          req.onerror = () => reject(req.error);
        } catch (e) {
          resolve({ error: String(e) });
        }
      });
    }
    const db = await openDb(dbName);
    const storeNames = Array.from(db.objectStoreNames);
    const out = { storeNames };
    for (const s of storeNames) {
      out[s] = await getAll(db, s);
    }
    return out;
  }, dbNames[0]).catch((e) => ({ error: String(e) }));

  console.log('=== DEXIE DUMP ===');
  console.log(JSON.stringify(dump, null, 2));

  await browser.close();
})().catch((e) => { console.error('SCRIPT FAILED:', e.message); process.exit(1); });
