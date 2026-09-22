const { chromium } = require('playwright');
const path = require('path');

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1920, height: 1080 } });
  const filePath = 'file:///' + path.resolve(__dirname, 'slide5.html').replace(/\\/g, '/');
  await page.goto(filePath);
  await page.screenshot({ path: path.resolve(__dirname, 'slide5.png') });
  await browser.close();
})();
