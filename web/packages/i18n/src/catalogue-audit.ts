// packages/i18n/src/catalogue-audit.ts
//
// Measures a message catalogue against the real problem this repo hit: a
// catalogue can type-check as Partial<Messages> while being 90% pasted
// Hindi, in the wrong script, missing placeholders, or missing plural
// categories the target language actually needs -- none of which the type
// system or a hand-picked "coverage" literal can catch. This is the
// instrument; it does not fix anything, it only measures.
//
// Deliberately plain data in, data out -- no file I/O here -- so both the
// vitest suite (catalogue-audit.test.ts) and the CLI (scripts/audit.mjs)
// can load catalogues however suits them and share this one check.

import { LOCALES, type LocaleCode } from './locales';
import { DNT_KEYS, PROPER_NOUN_KEY_PATTERNS } from './dnt-keys';

export type IssueKind =
  | 'missing'
  | 'extra'
  | 'empty'
  | 'placeholder-mismatch'
  | 'same-as-hi'
  | 'same-as-en'
  | 'wrong-script'
  | 'plural-category-missing';

export interface CatalogueIssue {
  readonly locale: LocaleCode;
  readonly key: string;
  readonly kind: IssueKind;
  readonly detail: string;
}

/** Unicode block per script code actually used in LOCALES (see locales.ts). */
const SCRIPT_RANGES: Readonly<Record<string, RegExp>> = {
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

const PLURAL_SUFFIX_RE = /\.(zero|one|two|few|many|other)$/;

function placeholdersOf(value: string): Set<string> {
  return new Set([...value.matchAll(/\{(\w+)\}/g)].map((m) => m[1]));
}

function setsEqual(a: Set<string>, b: Set<string>): boolean {
  if (a.size !== b.size) return false;
  for (const x of a) if (!b.has(x)) return false;
  return true;
}

function hasLetter(value: string): boolean {
  // Strip placeholders {placeholder} before checking for letters, per I18N_PLAN §2.4:
  // "values that are 100% placeholder/punctuation/digits" must not be flagged
  const stripped = value.replace(/\{[a-zA-Z0-9_]+\}/g, '');
  return /\p{L}/u.test(stripped);
}

/** Every base in en that represents a plural group (having at least .one and .other). */
function pluralBasesOf(catalogue: Record<string, string>): Set<string> {
  const bases = new Set<string>();
  const candidates = new Map<string, Set<string>>();
  for (const key of Object.keys(catalogue)) {
    const m = PLURAL_SUFFIX_RE.exec(key);
    if (m) {
      const base = key.slice(0, key.length - m[0].length);
      const cat = m[1];
      let set = candidates.get(base);
      if (!set) {
        set = new Set();
        candidates.set(base, set);
      }
      set.add(cat);
    }
  }
  for (const [base, cats] of candidates) {
    if (cats.has('one') && cats.has('other')) {
      bases.add(base);
    }
  }
  return bases;
}

/**
 * Audit one locale's catalogue against English (the key-set source of truth)
 * and Hindi (the thing a lazy fill actually pastes -- see CLAUDE.md's i18n
 * section). Call with `code: 'hi'` and `hi` === `catalogue` to audit Hindi
 * against English alone.
 */
export function auditCatalogue(
  code: LocaleCode,
  catalogue: Readonly<Record<string, string>>,
  en: Readonly<Record<string, string>>,
  hi: Readonly<Record<string, string>>,
): CatalogueIssue[] {
  const issues: CatalogueIssue[] = [];
  const scriptRe = SCRIPT_RANGES[LOCALES[code].script];
  const isHi = code === 'hi';

  for (const key of Object.keys(en)) {
    const value = catalogue[key];
    if (value === undefined) {
      issues.push({ locale: code, key, kind: 'missing', detail: 'absent from catalogue' });
      continue;
    }
    if (value.trim() === '') {
      issues.push({ locale: code, key, kind: 'empty', detail: 'value is empty' });
      continue;
    }
    if (DNT_KEYS.has(key)) continue;

    const enPlaceholders = placeholdersOf(en[key]);
    const valuePlaceholders = placeholdersOf(value);
    if (!setsEqual(enPlaceholders, valuePlaceholders)) {
      issues.push({
        locale: code,
        key,
        kind: 'placeholder-mismatch',
        detail: `expected {${[...enPlaceholders].join(', ')}}, got {${[...valuePlaceholders].join(', ')}}`,
      });
    }

    if (isHi) {
      if (value === en[key]) {
        issues.push({ locale: code, key, kind: 'same-as-en', detail: 'identical to English' });
      }
    } else {
      const hiValue = hi[key];
      if (hiValue !== undefined && value === hiValue && !PROPER_NOUN_KEY_PATTERNS.some((re) => re.test(key))) {
        issues.push({ locale: code, key, kind: 'same-as-hi', detail: 'identical to Hindi' });
      }
    }

    if (scriptRe && hasLetter(value) && !scriptRe.test(value)) {
      issues.push({
        locale: code,
        key,
        kind: 'wrong-script',
        detail: `no ${LOCALES[code].script} character found`,
      });
    }
  }

  for (const key of Object.keys(catalogue)) {
    if (!(key in en)) {
      issues.push({ locale: code, key, kind: 'extra', detail: 'not a key in en.ts' });
    }
  }

  const pluralCategories = new Intl.PluralRules(LOCALES[code].tag).resolvedOptions()
    .pluralCategories;
  for (const base of pluralBasesOf(en)) {
    for (const cat of pluralCategories) {
      const key = `${base}.${cat}`;
      if (key in en) continue; // already caught by the missing-key loop above
      if (!(key in catalogue)) {
        issues.push({
          locale: code,
          key,
          kind: 'plural-category-missing',
          detail: `${LOCALES[code].tag} requires plural category "${cat}"`,
        });
      }
    }
  }

  return issues;
}

/** Kinds that mean a key is not usably translated, for the coverage % below. */
const UNTRANSLATED_KINDS: ReadonlySet<IssueKind> = new Set([
  'missing',
  'empty',
  'same-as-hi',
  'same-as-en',
  'wrong-script',
]);

/** Rough "% of `total` keys actually translated" for the audit CLI's table. */
export function coverageOf(issues: readonly CatalogueIssue[], total: number): number {
  if (total === 0) return 100;
  const badKeys = new Set(
    issues.filter((i) => UNTRANSLATED_KINDS.has(i.kind)).map((i) => i.key),
  );
  return Math.round((100 * (total - badKeys.size)) / total);
}

/** Issue counts grouped by kind, for a readable failure message. */
export function issuesByKind(
  issues: readonly CatalogueIssue[],
): Readonly<Record<string, number>> {
  const counts: Record<string, number> = {};
  for (const issue of issues) counts[issue.kind] = (counts[issue.kind] ?? 0) + 1;
  return counts;
}
