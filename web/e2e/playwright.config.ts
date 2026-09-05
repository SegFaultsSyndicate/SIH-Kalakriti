// e2e/playwright.config.ts
//
// One config, three apps. Each gets its own project so a run can target a
// single surface, and each brings up its own preview server on the port that
// app's vite config already uses -- there is no shared dev server to
// coordinate, because there are three separate builds.
import { defineConfig, devices } from '@playwright/test';

const ARTISAN = 'http://localhost:4173';
const BUYER = 'http://localhost:4174';
const ADMIN = 'http://localhost:4175';

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  // Hydration on a cold preview server, with three servers and four workers
  // starting at once, regularly needs more than the 5s default.
  expect: { timeout: 10_000 },
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : [['list']],

  use: {
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },

  projects: [
    {
      name: 'artisan',
      testMatch: /artisan\/.*\.spec\.ts/,
      use: {
        // The artisan target is a mid-range Android, not a desktop Chrome
        // window scaled down: the touch target and viewport assertions are
        // only meaningful against a real device profile.
        ...devices['Pixel 5'],
        baseURL: ARTISAN,
      },
    },
    {
      name: 'buyer',
      testMatch: /buyer\/.*\.spec\.ts/,
      use: { ...devices['Desktop Chrome'], baseURL: BUYER },
    },
    {
      name: 'admin',
      testMatch: /admin\/.*\.spec\.ts/,
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1440, height: 900 },
        baseURL: ADMIN,
      },
    },
  ],

  webServer: [
    {
      command: 'pnpm -F @kalakriti/artisan preview --port 4173',
      url: ARTISAN,
      reuseExistingServer: !process.env.CI,
      cwd: '..',
    },
    {
      command: 'pnpm -F @kalakriti/buyer preview --port 4174',
      url: BUYER,
      reuseExistingServer: !process.env.CI,
      cwd: '..',
    },
    {
      command: 'pnpm -F @kalakriti/admin preview --port 4175',
      url: ADMIN,
      reuseExistingServer: !process.env.CI,
      cwd: '..',
    },
  ],
});
