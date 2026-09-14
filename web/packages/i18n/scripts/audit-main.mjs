// packages/i18n/scripts/audit-main.mjs
//
// The actual audit CLI. Split from audit.mjs because the resolve hook that
// makes catalogue-audit.ts's extensionless internal imports loadable under
// plain Node must be registered *before* anything imports that file -- and
// module.register() only affects imports that happen after it runs, not the
// static imports of the file that calls it. audit.mjs registers the hook,
// then dynamically imports this file so everything below sees it active.
//
// Usage:
//   node scripts/audit.mjs                              per-locale summary table
//   node scripts/audit.mjs --locale ta                   one locale, by kind
//   node scripts/audit.mjs --locale ta --kind same-as-hi --list   offending keys

import { parseArgs } from 'node:util';
import { LOCALE_CODES } from '../src/locales.ts';
import { en } from '../src/messages/en.ts';
import { auditCatalogue, coverageOf, issuesByKind } from '../src/catalogue-audit.ts';

const { values } = parseArgs({
  options: {
    locale: { type: 'string' },
    kind: { type: 'string' },
    list: { type: 'boolean', default: false },
  },
});

const NON_EN = LOCALE_CODES.filter((code) => code !== 'en');
const total = Object.keys(en).length;

if (values.locale && !NON_EN.includes(values.locale)) {
  console.error(`Unknown locale "${values.locale}". Known: ${NON_EN.join(', ')}`);
  process.exit(1);
}

const catalogues = new Map();
for (const code of NON_EN) {
  const mod = await import(`../src/messages/${code}.ts`);
  catalogues.set(code, mod[code]);
}
const hi = catalogues.get('hi') ?? {};

const codes = values.locale ? [values.locale] : NON_EN;
let exitCode = 0;

if (values.locale && values.list) {
  const issues = auditCatalogue(values.locale, catalogues.get(values.locale), en, hi);
  const filtered = values.kind ? issues.filter((i) => i.kind === values.kind) : issues;
  for (const issue of filtered) console.log(`${issue.key}\t${issue.kind}\t${issue.detail}`);
  console.log(`\n${filtered.length} issue(s)`);
  process.exit(0);
}

console.log(`en keys: ${total}`);
console.log('locale'.padEnd(8), 'issues'.padStart(7), 'coverage'.padStart(9), '  by kind');
for (const code of codes) {
  const issues = auditCatalogue(code, catalogues.get(code), en, hi);
  if (issues.length > 0) exitCode = 1;
  const byKind = issuesByKind(issues);
  const kindStr = Object.entries(byKind)
    .sort((a, b) => b[1] - a[1])
    .map(([k, n]) => `${k} ${n}`)
    .join(', ');
  console.log(
    code.padEnd(8),
    String(issues.length).padStart(7),
    `${coverageOf(issues, total)}%`.padStart(9),
    '  ' + kindStr,
  );
}
process.exit(exitCode);
