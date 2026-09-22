// packages/i18n/src/catalogue-audit.test.ts
//
// Ratchet test: each locale's issue count must not exceed its recorded
// baseline (i18n-baseline.json). Translation batches lower a locale's number
// and commit the new baseline -- see I18N_PLAN.md phase 5. The count is
// allowed to go up temporarily when Phase 3 adds new keys to en.ts (every
// other locale is then "missing" those keys until translated); it must never
// go up for a locale nobody touched.

import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { describe, expect, it, beforeAll } from 'vitest';
import { LOCALE_CODES, type LocaleCode } from './locales';
import { en } from './messages/en';
import { auditCatalogue, issuesByKind } from './catalogue-audit';

const baseline: Record<string, number> = JSON.parse(
  readFileSync(fileURLToPath(new URL('../i18n-baseline.json', import.meta.url)), 'utf8'),
);

const NON_EN: LocaleCode[] = LOCALE_CODES.filter((code) => code !== 'en');
const catalogues = new Map<LocaleCode, Record<string, string>>();

beforeAll(async () => {
  const loaded = await Promise.all(
    NON_EN.map((code) => import(`./messages/${code}.ts`) as Promise<Record<string, unknown>>),
  );
  NON_EN.forEach((code, i) => {
    catalogues.set(code, loaded[i][code] as Record<string, string>);
  });
});

describe('catalogue audit (ratchet against i18n-baseline.json)', () => {
  it.each(NON_EN)('%s does not exceed its baseline issue count', (code) => {
    const hi = catalogues.get('hi') ?? {};
    const issues = auditCatalogue(code, catalogues.get(code)!, en, hi);
    const max = baseline[code] ?? 0;
    if (issues.length > max) {
      // eslint-disable-next-line no-console -- diagnostic output on failure only
      console.error(`${code}: ${issues.length} issues (baseline ${max}):`, issuesByKind(issues));
    }
    expect(issues.length).toBeLessThanOrEqual(max);
  });

  it('every locale in LOCALE_CODES has a baseline entry', () => {
    for (const code of NON_EN) {
      expect(baseline, `missing baseline entry for "${code}"`).toHaveProperty(code);
    }
  });
});
