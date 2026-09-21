const { chromium } = require('@playwright/test');
const path = require('path');

const SHOTS = '/home/braine_dead/.claude/jobs/1a02db01/tmp/shots';
const IMAGE = '/home/braine_dead/Pictures/hollowKnightIcon.png';
const BASE = 'http://localhost:8081';

let shotN = 0;
async function shot(page, name) {
  shotN += 1;
  const file = path.join(SHOTS, `${String(shotN).padStart(2, '0')}-${name}.png`);
  await page.screenshot({ path: file, fullPage: true });
  console.log('SHOT', file, '|', page.url());
}

(async () => {
  const browser = await chromium.launch({
    args: ['--use-fake-ui-for-media-stream', '--use-fake-device-for-media-stream'],
  });
  const context = await browser.newContext({ permissions: ['microphone'] });
  const page = await context.newPage();

  page.on('console', (msg) => {
    if (msg.type() === 'error') console.log('CONSOLE ERROR:', msg.text());
  });
  page.on('requestfailed', (req) => {
    console.log('REQUEST FAILED:', req.method(), req.url(), req.failure()?.errorText);
  });
  page.on('response', async (res) => {
    const url = res.url();
    if (res.status() >= 400 || url.includes('/api/') || url.includes(':9000/')) {
      console.log('HTTP', res.status(), res.request().method(), url);
      if (res.status() >= 400 && url.includes('/api/')) {
        try {
          console.log('  REQ BODY:', res.request().postData());
          console.log('  RES BODY:', await res.text());
        } catch (e) {
          console.log('  (could not read body)', e.message);
        }
      }
    }
  });

  await page.goto(BASE + '/', { waitUntil: 'networkidle' });
  await shot(page, 'landing');

  if (page.url().includes('/language')) {
    await page.locator('button.language__select', { hasText: 'English' }).click();
    await page.waitForLoadState('networkidle');
    await shot(page, 'after-language');
  }

  if (page.url().includes('/welcome')) {
    await page.getByRole('button', { name: 'Get started' }).click();
    await page.waitForLoadState('networkidle');
    await shot(page, 'after-welcome');
  }

  if (page.url().includes('/login')) {
    await page.fill('#phone-input', '9999999999');
    await page.getByRole('button', { name: 'Send code' }).click();
    await page.waitForURL('**/verify', { timeout: 10000 });
    await shot(page, 'on-verify');
  }

  if (page.url().includes('/verify')) {
    const otpInputs = page.locator('input[inputmode="numeric"]');
    const count = await otpInputs.count();
    console.log('otp input boxes:', count);
    for (let i = 0; i < 6; i++) {
      await otpInputs.nth(i).fill(String(0));
    }
    await page.waitForLoadState('networkidle');
    await shot(page, 'after-verify');
  }

  console.log('URL after auth:', page.url());
  await shot(page, 'post-auth-state');

  // If bounced to registration wizard despite the phone already being
  // registered server-side, that IS the bug -- log it loudly and stop.
  if (page.url().includes('/register/')) {
    console.log('*** LANDED ON REGISTRATION WIZARD FOR AN ALREADY-REGISTERED PHONE ***');
    await browser.close();
    return;
  }

  // Navigate straight to a new listing capture, same as tapping "+ Add product" on home.
  await page.goto(BASE + '/listing/new/capture', { waitUntil: 'networkidle' });
  await shot(page, 'capture-page');

  const fileInput = page.locator('input[type="file"]');
  await fileInput.setInputFiles(IMAGE);
  await page.waitForTimeout(500);
  await shot(page, 'capture-after-file-selected');
  await page.getByRole('button', { name: /^Next/ }).click();
  await page.waitForURL('**/listing/new/studio*', { timeout: 15000 });
  await shot(page, 'studio-after-upload');

  // Studio: wait for whatever processing/apply state resolves, then continue.
  const continueBtn = page.getByRole('button', { name: /Continue to Video/i });
  await continueBtn.waitFor({ state: 'visible', timeout: 15000 });
  await shot(page, 'studio-before-continue');
  await continueBtn.click();
  await page.waitForURL('**/listing/new/video*', { timeout: 15000 });
  await shot(page, 'video-page');

  // Video step: skip.
  const skipOrNext = page.getByRole('button', { name: /Skip|Next/ });
  await skipOrNext.click();
  await page.waitForLoadState('networkidle');
  await shot(page, 'after-video-skip');
  console.log('URL after video skip:', page.url());

  if (page.url().includes('/register/')) {
    console.log('*** BOUNCED TO REGISTRATION RIGHT AFTER VIDEO SKIP -- REPRODUCED ***');
    await browser.close();
    return;
  }

  // Story step: pick a craft, try recording a voice note, then continue.
  await shot(page, 'story-page');
  const craftSelect = page.locator('select').first();
  if (await craftSelect.count()) {
    const optionValues = await craftSelect.locator('option').evaluateAll((opts) =>
      opts.map((o) => o.value).filter((v) => v !== ''),
    );
    if (optionValues.length) {
      await craftSelect.selectOption(optionValues[0]);
    }
  }
  await shot(page, 'story-craft-picked');

  const recordBtn = page.getByRole('button', { name: /record|voice/i }).first();
  if (await recordBtn.count()) {
    await recordBtn.click();
    await page.waitForTimeout(2000);
    await shot(page, 'story-recording');
    // Click again to stop, if it's a toggle.
    const stopBtn = page.getByRole('button', { name: /stop|done/i }).first();
    if (await stopBtn.count()) {
      await stopBtn.click();
    } else {
      await recordBtn.click();
    }
    await page.waitForTimeout(1000);
    await shot(page, 'story-after-recording');
  }

  console.log('URL after voice note attempt:', page.url());
  if (page.url().includes('/register/')) {
    console.log('*** BOUNCED TO REGISTRATION RIGHT AFTER VOICE RECORDING -- REPRODUCED ***');
    await browser.close();
    return;
  }

  const storyNext = page.getByRole('button', { name: 'Next' });
  if (await storyNext.count()) {
    const disabled = await storyNext.first().isDisabled();
    console.log('story Next disabled?', disabled);
    if (!disabled) {
      await storyNext.first().click();
      await page.waitForLoadState('networkidle');
      await shot(page, 'after-story-next');
    }
  }

  console.log('FINAL URL:', page.url());
  await shot(page, 'final-state');

  if (page.url().includes('/listing/new/processing')) {
    for (let i = 0; i < 35; i++) {
      const text = await page.textContent('body');
      const line = text.match(/Uploading[^\n]*|left|done/i);
      console.log('processing poll', i, ':', line ? text.slice(Math.max(0, line.index - 20), line.index + 40).replace(/\s+/g, ' ') : '(no match)');
      const nextDisabled = await page.getByRole('button', { name: /^Next/ }).isDisabled().catch(() => null);
      console.log('  Next disabled?', nextDisabled);
      if (nextDisabled === false) break;
      await page.waitForTimeout(3000);
    }
    await shot(page, 'processing-polled');
    const nextBtn = page.getByRole('button', { name: /^Next/ });
    if (!(await nextBtn.isDisabled())) {
      await nextBtn.click();
      await page.waitForLoadState('networkidle');
      await shot(page, 'review-page');
      console.log('REVIEW URL:', page.url());
    }
  }

  await browser.close();
})().catch(async (e) => {
  console.error('SCRIPT FAILED:', e.message);
  process.exit(1);
});
