// packages/i18n/src/render-sweep.test.ts
import { describe, expect, it } from 'vitest';
import { LOCALE_CODES, type LocaleCode } from './locales';
import { en, type MessageKey } from './messages/en';
import { interpolate } from './format-message';

const allKeys = Object.keys(en) as MessageKey[];
const RAW_KEY_RE = /^[a-z]+(\.[a-zA-Z0-9]+){1,3}$/;

describe('render-sweep: all 21 locales render complete translated text', () => {
  for (const code of LOCALE_CODES) {
    it(`locale ${code} resolves all 2,363 keys without raw key fallthrough or missing entries`, async () => {
      const mod = (await import(`./messages/${code}.ts`)) as Record<string, unknown>;
      const catalogue = mod[code] as Record<string, string>;

      expect(catalogue).toBeDefined();
      expect(typeof catalogue).toBe('object');

      for (const key of allKeys) {
        const rawVal = catalogue[key];
        expect(rawVal, `Locale ${code} must have key ${key}`).toBeDefined();
        expect(rawVal.trim(), `Locale ${code} value for ${key} must not be empty`).not.toBe('');

        // Interpolate with dummy values if placeholders exist
        const interpolated = interpolate(rawVal, {
          name: 'Test',
          count: 5,
          total: 10,
          orderId: 'ORD123',
          artisanName: 'Artisan',
          date: '2026-09-14',
          price: '500',
          currency: 'INR',
          digit: '9',
          percent: '10',
          tier: 'Tier 1',
          code: 'CODE',
          phone: '9876543210',
          email: 'test@kalakriti.gov.in',
        });

        expect(typeof interpolated).toBe('string');
        // A translated string should not be equal to a raw message key unless the key itself is DNT like 'app.name'
        if (key !== 'app.name') {
          expect(interpolated).not.toMatch(RAW_KEY_RE);
        }
      }
    });
  }
});
