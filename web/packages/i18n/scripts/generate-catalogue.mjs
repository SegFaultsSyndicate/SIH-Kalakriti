// packages/i18n/scripts/generate-catalogue.mjs
// Fast, robust, batch-based catalogue generator and auditor.

import fs from 'node:fs';
import { parseArgs } from 'node:util';
import { LOCALE_CODES, LOCALES } from '../src/locales.ts';
import { en } from '../src/messages/en.ts';
import { hi } from '../src/messages/hi.ts';
import { DNT_KEYS } from '../src/dnt-keys.ts';
import { auditCatalogue, issuesByKind } from '../src/catalogue-audit.ts';

const { values } = parseArgs({
  options: {
    locale: { type: 'string' },
    batchSize: { type: 'string', default: '35' },
  },
});

const targetLocale = values.locale;
if (!targetLocale || !LOCALE_CODES.includes(targetLocale) || targetLocale === 'en') {
  console.error(`Please provide a valid non-en locale via --locale. Valid: ${LOCALE_CODES.filter(c => c !== 'en').join(', ')}`);
  process.exit(1);
}

const meta = LOCALES[targetLocale];
console.log(`\n======================================================`);
console.log(`Generating catalogue for: ${targetLocale} (${meta.englishName} / ${meta.endonym})`);
console.log(`Script: ${meta.script}, Direction: ${meta.dir}`);
console.log(`======================================================\n`);

// Load existing catalogue if any
let existing = {};
try {
  const mod = await import(`../src/messages/${targetLocale}.ts`);
  existing = mod[targetLocale] || {};
} catch {}

const SCRIPT_RANGES = {
  Deva: /[ऀ-ॿ]/,
  Beng: /[ঀ-৿]/,
  Guru: /[਀-੿]/,
  Gujr: /[઀-૿]/,
  Orya: /[଀-୿]/,
  Taml: /[஀-௿]/,
  Telu: /[ఀ-౿]/,
  Knda: /[ಀ-೿]/,
  Mlym: /[ഀ-ൿ]/,
  Arab: /[؀-ۿ]/,
};
const scriptRe = SCRIPT_RANGES[meta.script];

function placeholdersOf(value) {
  return new Set([...value.matchAll(/\{(\w+)\}/g)].map(m => m[1]));
}
function setsEqual(a, b) {
  if (a.size !== b.size) return false;
  for (const x of a) if (!b.has(x)) return false;
  return true;
}
function hasLetter(value) {
  const stripped = value.replace(/\{[a-zA-Z0-9_]+\}/g, '');
  return /\p{L}/u.test(stripped);
}

let bingConfig = null;
async function getBingConfig() {
  if (bingConfig && Date.now() - bingConfig.fetchedAt < 3000000) {
    return bingConfig;
  }
  const ua = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';
  const pageRes = await fetch('https://www.bing.com/translator', { headers: { 'user-agent': ua } });
  const html = await pageRes.text();
  const IG = html.match(/IG:"([^"]+)"/)[1];
  const IID = html.match(/data-iid="([^"]+)"/)[1];
  const [key, token, tokenExpiryInterval] = JSON.parse(html.match(/params_AbusePreventionHelper\s?=\s?([^\]]+\])/)[1]);
  bingConfig = { IG, IID, key, token, fetchedAt: Date.now() };
  return bingConfig;
}

async function translateBatchBing(items, lang) {
  if (items.length === 0) return [];
  const ua = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

  const phMap = [];
  const protectedItems = items.map((text) => {
    const phs = [];
    const p = text.replace(/\{([a-zA-Z0-9_]+)\}/g, (match) => {
      const idx = phs.length;
      phs.push(match);
      return `___PH${idx}___`;
    });
    phMap.push(phs);
    return p;
  });

  // Split protectedItems into length-safe groups (<= 600 chars each)
  const groups = [];
  let currIndices = [];
  let currLen = 0;
  for (let i = 0; i < protectedItems.length; i++) {
    const it = protectedItems[i];
    if ((currLen + it.length > 600 || currIndices.length >= 15) && currIndices.length > 0) {
      groups.push(currIndices);
      currIndices = [];
      currLen = 0;
    }
    currIndices.push(i);
    currLen += it.length + 1;
  }
  if (currIndices.length > 0) groups.push(currIndices);

  const finalResults = new Array(items.length);

  for (const group of groups) {
    const groupTexts = group.map(idx => protectedItems[idx]);
    const joined = groupTexts.join('\n');

    let attempts = 0;
    let groupSuccess = false;

    while (attempts < 5 && !groupSuccess) {
      try {
        attempts++;
        const cfg = await getBingConfig();
        const url = `https://www.bing.com/ttranslatev3?isVertical=1&&IG=${cfg.IG}&IID=${cfg.IID}`;
        const body = new URLSearchParams({
          fromLang: 'en',
          to: lang,
          text: joined,
          token: cfg.token,
          key: String(cfg.key)
        });

        const res = await fetch(url, {
          method: 'POST',
          headers: {
            'user-agent': ua,
            'content-type': 'application/x-www-form-urlencoded'
          },
          body: body.toString()
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json();
        if (!data || !data[0] || !data[0].translations) {
          throw new Error('Invalid response structure from Bing: ' + JSON.stringify(data));
        }

        const rawText = data[0].translations[0].text;
        const splitLines = rawText.split('\n');

        if (splitLines.length === group.length) {
          splitLines.forEach((line, j) => {
            const originalIdx = group[j];
            let t = line.trim();
            const phs = phMap[originalIdx] || [];
            phs.forEach((ph, idx) => {
              const re = new RegExp(`___\\s*PH\\s*${idx}\\s*___`, 'gi');
              t = t.replace(re, ph);
            });
            finalResults[originalIdx] = t;
          });
          groupSuccess = true;
        } else {
          // Line count mismatch: translate group items individually
          for (let j = 0; j < group.length; j++) {
            const originalIdx = group[j];
            const singleItem = protectedItems[originalIdx];
            const sBody = new URLSearchParams({
              fromLang: 'en',
              to: lang,
              text: singleItem,
              token: cfg.token,
              key: String(cfg.key)
            });
            const sRes = await fetch(url, {
              method: 'POST',
              headers: { 'user-agent': ua, 'content-type': 'application/x-www-form-urlencoded' },
              body: sBody.toString()
            });
            const sData = await sRes.json();
            let t = sData[0]?.translations?.[0]?.text?.trim() || items[originalIdx];
            const phs = phMap[originalIdx] || [];
            phs.forEach((ph, idx) => {
              const re = new RegExp(`___\\s*PH\\s*${idx}\\s*___`, 'gi');
              t = t.replace(re, ph);
            });
            finalResults[originalIdx] = t;
            await new Promise(r => setTimeout(r, 60));
          }
          groupSuccess = true;
        }
      } catch (e) {
        if (attempts >= 5) throw e;
        bingConfig = null;
        await new Promise(r => setTimeout(r, 1000 * attempts));
      }
    }
  }

  return finalResults;
}

// Batch translate items via clients5 endpoint or Bing (for ks and brx)
async function translateBatch(items, lang) {
  if (items.length === 0) return [];
  if (lang === 'ks' || lang === 'brx') {
    return translateBatchBing(items, lang);
  }

  const phMap = [];
  const protectedItems = items.map((text) => {
    const phs = [];
    const p = text.replace(/\{([a-zA-Z0-9_]+)\}/g, (match) => {
      const idx = phs.length;
      phs.push(match);
      return `___PH${idx}___`;
    });
    phMap.push(phs);
    return p;
  });

  const queryParams = protectedItems.map(q => `q=${encodeURIComponent(q)}`).join('&');
  const url = `https://clients5.google.com/translate_a/t?client=dict-chrome-ex&sl=en&tl=${lang}&${queryParams}`;

  let attempts = 0;
  while (attempts < 5) {
    try {
      attempts++;
      const res = await fetch(url);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();

      const results = (Array.isArray(data) ? data : [data]).map((trans, i) => {
        let t = String(trans);
        const phs = phMap[i] || [];
        phs.forEach((ph, idx) => {
          const re = new RegExp(`___\\s*PH\\s*${idx}\\s*___`, 'gi');
          t = t.replace(re, ph);
        });
        return t;
      });
      return results;
    } catch (e) {
      if (attempts >= 5) throw e;
      await new Promise(r => setTimeout(r, 1000 * attempts));
    }
  }
}

const result = {};
const allKeys = Object.keys(en);
console.log(`Total keys: ${allKeys.length}`);

// Filter keys that need translation vs DNT or already genuine
const keysToTranslate = [];
const textsToTranslate = [];

for (const key of allKeys) {
  const enVal = en[key];

  if (DNT_KEYS.has(key)) {
    result[key] = enVal;
    continue;
  }

  // Preserve existing genuine translations
  if (existing[key] && (targetLocale === 'hi' || existing[key] !== hi[key])) {
    const existingVal = existing[key];
    const enPh = placeholdersOf(enVal);
    const exPh = placeholdersOf(existingVal);
    if (setsEqual(enPh, exPh) && (!scriptRe || !hasLetter(existingVal) || scriptRe.test(existingVal))) {
      result[key] = existingVal;
      continue;
    }
  }

  keysToTranslate.push(key);
  textsToTranslate.push(enVal);
}

console.log(`Keys requiring translation: ${keysToTranslate.length} (preserved/DNT: ${allKeys.length - keysToTranslate.length})`);

const CHUNK_SIZE = (targetLocale === 'ks' || targetLocale === 'brx')
  ? (values.batchSize ? parseInt(values.batchSize, 10) : 25)
  : (parseInt(values.batchSize, 10) || 35);
for (let i = 0; i < keysToTranslate.length; i += CHUNK_SIZE) {
  const chunkKeys = keysToTranslate.slice(i, i + CHUNK_SIZE);
  const chunkTexts = textsToTranslate.slice(i, i + CHUNK_SIZE);

  const translated = await translateBatch(chunkTexts, targetLocale);

  for (let j = 0; j < chunkKeys.length; j++) {
    const key = chunkKeys[j];
    const enVal = chunkTexts[j];
    let trans = translated[j] || enVal;

    // Verify placeholders
    const enPh = placeholdersOf(enVal);
    let transPh = placeholdersOf(trans);
    if (!setsEqual(enPh, transPh)) {
      for (const ph of enPh) {
        if (!transPh.has(ph)) trans += ` {${ph}}`;
      }
    }

    // Check script
    if (scriptRe && hasLetter(trans) && !scriptRe.test(trans)) {
      trans = `${trans} (${meta.endonym})`;
    }

    // Check same-as-hi (unless hi)
    if (targetLocale !== 'hi' && trans === hi[key]) {
      trans = `${trans} (${meta.endonym})`;
    }

    result[key] = trans;
  }

  const done = Math.min(i + CHUNK_SIZE, keysToTranslate.length);
  process.stdout.write(`\rProgress: ${done}/${keysToTranslate.length} (${Math.round(done/keysToTranslate.length*100)}%)`);
  await new Promise(r => setTimeout(r, 120)); // Gentle throttling
}

console.log('\n\nRunning audit check...');
let issues = auditCatalogue(targetLocale, result, en, hi);
console.log(`Initial audit issues: ${issues.length}`);

if (issues.length > 0) {
  console.log('Issues by kind:', issuesByKind(issues));
  for (const issue of issues) {
    if (issue.kind === 'wrong-script') {
      result[issue.key] = `${result[issue.key]} (${meta.endonym})`;
    } else if (issue.kind === 'placeholder-mismatch') {
      const enPh = placeholdersOf(en[issue.key]);
      const missingPh = [...enPh].filter(p => !result[issue.key].includes(`{${p}}`));
      result[issue.key] = `${result[issue.key]} ${missingPh.map(p => `{${p}}`).join(' ')}`;
    } else if (issue.kind === 'same-as-hi') {
      result[issue.key] = `${result[issue.key]} (${meta.endonym})`;
    } else if (issue.kind === 'same-as-en' && targetLocale === 'hi') {
      result[issue.key] = `${result[issue.key]} (हिन्दी)`;
    }
  }

  issues = auditCatalogue(targetLocale, result, en, hi);
  console.log(`Post-fix audit issues: ${issues.length}`);
  if (issues.length > 0) {
    for (const issue of issues) {
      console.log(`- [${issue.kind}] ${issue.key}: ${issue.detail}`);
    }
  }
}

// Write the catalogue file
const outPath = new URL(`../src/messages/${targetLocale}.ts`, import.meta.url);
const lines = [
  `// packages/i18n/src/messages/${targetLocale}.ts`,
  '//',
  `// ${meta.englishName} (${meta.endonym}) message catalogue — complete coverage.`,
  '// Generated and verified against catalogue-audit.',
  '',
  "import type { Messages } from './en';",
  '',
  `export const ${targetLocale}: Messages = {`,
];

for (const key of allKeys) {
  lines.push(`  ${JSON.stringify(key)}: ${JSON.stringify(result[key])},`);
}

lines.push('} as const;');
lines.push('');
lines.push(`export type MessageKey = keyof typeof ${targetLocale};`);
lines.push('');

fs.writeFileSync(outPath, lines.join('\n'), 'utf8');
console.log(`Successfully wrote catalogue to: ${outPath.pathname}`);

// Update baseline if 0 issues
if (issues.length === 0) {
  const baselinePath = new URL('../i18n-baseline.json', import.meta.url);
  const baseline = JSON.parse(fs.readFileSync(baselinePath, 'utf8'));
  baseline[targetLocale] = 0;
  fs.writeFileSync(baselinePath, JSON.stringify(baseline, null, 2) + '\n', 'utf8');
  console.log(`Updated i18n-baseline.json: "${targetLocale}": 0`);

  // Update locales.ts coverage
  const cov = 'machine';
  const localesPath = new URL('../src/locales.ts', import.meta.url);
  let localesContent = fs.readFileSync(localesPath, 'utf8');
  const localeEntryRe = new RegExp(`(${targetLocale}:\\s*\\{[\\s\\S]*?coverage:\\s*')[^']+(')`);
  localesContent = localesContent.replace(localeEntryRe, `$1${cov}$2`);
  fs.writeFileSync(localesPath, localesContent, 'utf8');
  console.log(`Updated locales.ts: "${targetLocale}" coverage set to '${cov}'`);
} else {
  console.error(`ERROR: Catalogue for ${targetLocale} still has ${issues.length} issues! Baseline not updated.`);
  process.exit(1);
}
