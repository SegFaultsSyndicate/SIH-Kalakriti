# Kalakriti full-coverage i18n remediation plan

**Status:** plan only — nothing in this document has been implemented.
**Audience:** the implementing session (Sonnet 5) and any reviewer.
**Scope:** `web/packages/i18n`, `web/apps/{buyer,artisan,admin}`, `web/packages/{ui,offline,voice}`.

---

## 0. Read this first: why the "20 languages" commit did not do what it claimed

Commit `2aa4613` ("cut down 22 -> 20 lang + populated it across all the
components") created 20 catalogue files, wired 20 loaders, and made the picker
list 20 languages. It did not translate anything. The evidence, measured
against the checked-in files, not inferred:

| Catalogue | Keys present | Missing vs `en` | Values byte-identical to `hi` | Genuinely in own language |
|---|---|---|---|---|
| `en.ts` | 1411 | — | — | source of truth |
| `hi.ts` | 1093 | 318 | — | 1093 |
| `ta.ts` | 808 | 603 | 673 | **135** |
| `ml.ts` | 808 | 603 | 673 | **135** |
| `te.ts`, `kn.ts`, `or.ts`, `pa.ts`, `gu.ts`, `as.ts`, `bn.ts`, `ur.ts` | 808 | 603 | ~673 | **132–136** |
| `mr.ts` | 808 | 603 | 676 | 132 |
| `sa.ts`, `ne.ts`, `brx.ts`, `kok.ts`, `ks.ts`, `sd.ts`, `doi.ts`, `mai.ts` | 808 | 603 | 718–723 | **85–90** |

So for Tamil, a user sees:

- 135 strings (9.6% of the surface) in Tamil,
- 673 strings (47.7%) in **Hindi**, sitting in `ta.ts` pretending to be Tamil,
- 603 strings (42.7%) absent from `ta.ts`, resolving through `#lookup`'s
  `active → hi → en` chain, which means **Hindi again** (because `hi.ts` has
  them), and English for the 318 keys Hindi is also missing.

A character-script census of the value strings confirms it independently:

```
ta.ts  Deva 80%, Taml 18%      ml.ts  Deva 81%, Mlym 17%
te.ts  Deva 82%, Telu 16%      kn.ts  Deva 81%, Knda 16%
or.ts  Deva 82%, Orya 15%      pa.ts  Deva 84%, Guru 14%
gu.ts  Deva 83%, Gujr 14%      bn.ts  Deva 83%, Beng 14%
as.ts  Deva 82%, Beng 15%      ur.ts  Deva 83%, Arab 14%
ks.ts  Deva 89%, Arab  8%      sd.ts  Deva 90%, Arab  7%
```

`ks.ts` (Kashmiri, Perso-Arabic script) being 89% Devanagari is wrong twice
over — wrong language *and* wrong script.

That is the whole explanation for "some things are getting changed their
wordings upon changing the language" and "a flash of translated lang in
between". Nothing is mysterious. The work was never done.

**Second, independent problem:** 2,226 user-visible strings are hardcoded in
components and data files and never reach the catalogue at all — no language
setting can translate them. Breakdown in §4.

### 0a. Your screenshots are additionally corrupted by Chrome's own translator

Before any further manual evaluation: **Chrome's page translation is ON** in
those tabs. Proof from your own screenshots, not a guess:

- Screenshot 1's address bar shows the Translate icon in the active state.
- The language picker renders `LOCALES[code].endonym` verbatim from
  `packages/i18n/src/locales.ts` — a literal, never passed through `t()`. Bodo's
  endonym there is `बड़ो`. Your screenshot shows **"elders"**, which is Google
  Translate's Hindi→English rendering of `बड़ो`. Our code cannot produce that.
- Same cause for Konkani/Maithili/Marathi/Nepali appearing in Latin script in
  screenshot 2 instead of their Devanagari endonyms.
- Screenshot 1's body reads "swadoorvanisankhya writing" / "Shrinotu". Those are
  Google's transliterations of `sa.ts`'s `"login.heading": "स्वदूरवाणीसङ्ख्यां लिखतु"`
  and `"action.speak": "शृणोतु"` — i.e. Chrome chewing on the **Sanskrit**
  catalogue while the picker's checkmark sat on Hindi.

**Action, before executing any step below:** right-click the page → *Translate to
English* → off, and set Chrome Settings → Languages → "Offer to translate pages"
to off for `localhost`. Re-test. Anything evaluated through Chrome's translator
is unmeasurable.

That last bullet does leave one real anomaly to chase — picker showing Hindi
selected while the body rendered Sanskrit keys. See §6, task **F-3**; do not
fold it into "partial translation" and lose it.

### 0b. Why CI never caught any of this

`packages/i18n/src/locales.test.ts:24-30` asserts:

```ts
const complete = new Set(['en','hi','bn','gu','mr','or','pa','sd','ta','te','ur']);
```

It asserts that the `coverage` **literal in `locales.ts`** equals that set. It
never reads a single catalogue file. Nine languages are declared `'complete'`
while being 9%-translated Hindi copies. The test is green and has always been
green. This test is the reason the problem survived a commit that claimed to
fix it, and fixing the test is therefore **Phase 1, before any translation
work**.

---

## 1. Definition of done

Numbered so the implementing session can check each one mechanically.

- **D1** — For every one of the 21 locale codes, `messages/<code>.ts` contains
  **every** key in `en.ts` (currently 1411; this number grows in Phase 4).
- **D2** — For every non-`en` locale, no value is byte-identical to `hi[key]`
  unless the key is on the do-not-translate list (§2.4). For `hi` itself, no
  value is byte-identical to `en[key]` under the same exception.
- **D3** — For every locale whose script is not Devanagari, every translatable
  value contains at least one character in that locale's expected script block.
- **D4** — Every value's `{placeholder}` set is exactly equal to the English
  value's `{placeholder}` set, for every key, every locale.
- **D5** — Every plural base key has one entry per CLDR category that
  `new Intl.PluralRules(LOCALES[code].tag).resolvedOptions().pluralCategories`
  reports for that locale.
- **D6** — Zero hardcoded user-visible strings remain in `apps/**` and
  `packages/{ui,offline,voice}/**`, as measured by the committed lint gate
  (§4.4), which fails the build on regression.
- **D7** — Every catalogue is typed `Messages`, not `Partial<Messages>`, so
  TypeScript enforces D1 forever with zero test code.
- **D8** — With the backend fully stopped, all three apps render every screen
  they can render at all, fully in the selected language, including the error
  and empty states (§7).
- **D9** — `ks`, `sd`, `ur` render right-to-left with correct per-script fonts,
  and layout is not broken in RTL (§8).
- **D10** — `pnpm -r test` and `pnpm -r check` pass.

### What this plan deliberately does not claim

Translations produced here are **model-generated with mechanical validation**,
not human-reviewed by a native speaker. `LocaleMeta.coverage` already has a
`'machine'` value for exactly this. Every locale this plan fills is set to
`'machine'`, never `'complete'`. Only `en` (authored) and `hi` (if a native
reviewer signs it off) may be `'complete'`.

Overclaiming quality is precisely what produced the current state — a commit
message that said "populated across all the components" for a 9.6% fill. Do not
repeat it.

---

## 2. Phase 1 — Build the validator before touching a single string

Nothing else in this plan is safe without this. It is the instrument that makes
"not a single thing left untranslated" a measurable claim rather than a hope.

### 2.1 New file: `packages/i18n/src/catalogue-audit.ts`

Exported, pure, no test framework dependency, so both the vitest suite and a CLI
script can use it. It must **read the message files at runtime** (via
`import.meta.glob` under vitest, or `fs` under the CLI), not import the typed
objects — the whole point is to check things the type system cannot.

```ts
export interface CatalogueIssue {
  locale: LocaleCode;
  key: string;
  kind: 'missing' | 'extra' | 'same-as-hi' | 'same-as-en'
      | 'wrong-script' | 'placeholder-mismatch' | 'plural-category-missing'
      | 'empty';
  detail: string;
}
export function auditCatalogue(
  code: LocaleCode,
  catalogue: Record<string, string>,
  en: Record<string, string>,
  hi: Record<string, string>,
): CatalogueIssue[];
export function coverageOf(issues: CatalogueIssue[], total: number): number;
```

Checks, in this order:

1. **`missing`** — key in `en`, absent from `catalogue`.
2. **`extra`** — key in `catalogue`, absent from `en`. (The `Partial<Messages>`
   type already catches this; keep the check for the CLI path.)
3. **`empty`** — value is `''` or whitespace only.
4. **`placeholder-mismatch`** — `new Set(v.match(/\{(\w+)\}/g))` differs from
   English's. A dropped `{name}` renders a sentence with a hole in it and no
   test currently catches it.
5. **`wrong-script`** — for locales whose `LOCALES[code].script` is not `Latn`,
   the value must contain ≥1 char in that script's Unicode block. Use the map
   below. **Skipped for do-not-translate keys (§2.4) and for values that are
   100% placeholder/punctuation/digits.**
6. **`same-as-hi`** — for every locale except `hi` and `en`, value ===
   `hi[key]`. This is the *only* check that catches the 8 Devanagari locales
   (`sa ne brx kok ks doi mai mr`), where a script check is blind by
   construction. It is the single most important check in the file.
7. **`same-as-en`** — for `hi` (and, once filled, any locale), value ===
   `en[key]` outside the DNT list.
8. **`plural-category-missing`** — for each key matching `/\.(zero|one|two|few|many|other)$/`,
   group by base; assert the group has every category
   `Intl.PluralRules(LOCALES[code].tag).resolvedOptions().pluralCategories`
   returns. Missing categories make `tPlural` select a key `#lookup` cannot
   find, and `#lookup` then returns **the raw dotted key to the screen**.

Script block map (the `sc()` ranges verified against the existing files):

```
Deva U+0900–097F : hi mr ne sa brx doi kok mai (+ ks/sd only if Deva variant chosen — it is not, see §8)
Beng U+0980–09FF : bn as
Guru U+0A00–0A7F : pa
Gujr U+0A80–0AFF : gu
Orya U+0B00–0B7F : or
Taml U+0B80–0BFF : ta
Telu U+0C00–0C7F : te
Knda U+0C80–0CFF : kn
Mlym U+0D00–0D7F : ml
Arab U+0600–06FF : ur sd ks
Latn             : en
```

### 2.2 New test: `packages/i18n/src/catalogue-audit.test.ts`

One `describe.each` over `LOCALE_CODES`. It must **not** be written as one giant
assertion — report per-locale issue counts by `kind` so a failure names what is
wrong, e.g.:

```
ta: 603 missing, 673 same-as-hi, 0 placeholder-mismatch  →  9.6% translated
```

During Phases 3–5 the suite runs against a **ratchet file** (§2.5) rather than
demanding zero immediately, so the repo is never red for weeks.

### 2.3 Rewrite `locales.test.ts`

Delete the hardcoded `complete` set. Replace with: for every locale, the
`coverage` literal declared in `locales.ts` must agree with what
`auditCatalogue` actually measures —

- `complete` requires 0 issues **and** an explicit entry in a
  `HUMAN_REVIEWED: readonly LocaleCode[]` list (initially `['en']`),
- `machine` requires 0 issues,
- `fallback` requires the catalogue to be absent from `CATALOGUE_LOADERS`.

Then a lie in `locales.ts` fails CI instead of blessing itself.

### 2.4 New file: `packages/i18n/src/dnt-keys.ts`

The do-not-translate key list, so the script/sameness checks do not flag
correct behaviour and tempt someone into "translating" a proper noun.

Seed it with, at minimum:

```ts
export const DNT_KEYS = [
  'app.name',          // "Kalakriti" — brand, never translated
  'app.ministry',      // rendered per the ministry's own bilingual convention
  'currency.inr',      // symbol
  // craft/GI proper nouns: Banarasi, Ajrakh, Pochampally, Dhokra, Kanjeevaram, …
] as const;
```

Audit `en.ts`'s `craft.*` (21 keys), `handloom.*` (8) and `gem.*` (24)
namespaces while building this list — proper craft names belong here; their
descriptions do not. This interacts with the existing `DntTerm` type in
`dnt-terms.ts`, which covers runtime-supplied terms; `DNT_KEYS` covers
build-time catalogue keys. Two different mechanisms, same rule.

### 2.5 Ratchet file: `packages/i18n/i18n-baseline.json`

`{ "<locale>": <issue count> }`, committed. The test asserts
`issues.length <= baseline[locale]` and **fails if the number goes up**. Each
batch in Phase 5 lowers a number and commits the new baseline. This gives
monotonic, reviewable progress instead of one 28,000-string commit nobody can
review.

### 2.6 CLI: `packages/i18n/scripts/audit.mjs`

`pnpm --filter @kalakriti/i18n audit` → the per-locale table from §0.
`--locale ta --kind same-as-hi --list` prints the offending keys, which is what
the translation batches consume as their worklist. Add
`"audit": "node scripts/audit.mjs"` to that package's `scripts`.

**Phase 1 exit gate:** `pnpm --filter @kalakriti/i18n audit` reproduces the
table in §0 from the current files. If it does not, the validator is wrong —
fix the validator before proceeding. This is the calibration step; skipping it
means everything downstream is measured with an uncalibrated instrument.

---

## 3. Phase 2 — Fix the lookup and init paths

These are bugs that will persist even at 100% translation coverage. Fix them
before translating, so the translation work is validated against correct
runtime behaviour.

### F-1 — Fire-and-forget `init()` causes the flash you saw

`apps/{buyer,artisan,admin}/src/routes/+layout.svelte` all call:

```ts
void locale.init();
```

`init()` is async (it awaits a dynamic `import()`). First paint therefore runs
with `#code = DEFAULT_LOCALE` and `#catalogue = {}`, so every `#lookup` falls
through to `en[key]`. The catalogue lands a tick later, `#version++` fires, and
the whole tree re-renders in the real language. **That is the flash.**

Fix: block first paint on locale resolution.

- Make the root layout `await locale.init()` inside a `{#await}` (or gate the
  `<slot/>` on an `initialised` flag on `LocaleState`) so nothing renders in the
  wrong language.
- Cheaper and better: read `localStorage['kalakriti.locale']` **synchronously**
  in `app.html`'s inline head script and set `<html lang>`/`dir` there, then
  only the catalogue chunk is async. `readStoredLocale()` is already synchronous;
  only the `import()` is not.
- Because `en` is statically imported, `en`-selected users have zero async path;
  the flash only affects the other 20 — i.e. everyone this plan is for.

### F-2 — `set()` has no failure path

```ts
const loader = CATALOGUE_LOADERS[code] ?? (async () => FALLBACK_CATALOGUES[code] ?? {});
this.#catalogue = await loader();
```

If the dynamic `import()` rejects (chunk 404 after a redeploy, offline before
the chunk was ever cached, flaky 2G — the exact network this app targets), the
whole `set()` promise rejects. `#code` is never assigned, `<html lang>` is never
updated, `persist` never runs, and the caller in `LanguageSelector.svelte` does
`void locale.set(code)` — so the rejection is an unhandled promise rejection
and **the UI silently does nothing**. The user taps their language and nothing
happens.

Fix: wrap the loader in try/catch; on failure fall back to
`FALLBACK_CATALOGUES[code] ?? {}`, still assign `#code`, still persist, and
surface a toast. A language switch must never be a no-op.

### F-3 — Reproduce the Hindi-selected / Sanskrit-rendered anomaly

From screenshot 1, after Chrome's translator is off (§0a). Suspect ordering in
`set()`: `#catalogue` is assigned *before* `#code`, and `#version++` happens
last, so there is a window where `#catalogue` is the new language's but `meta`
(used by `tPlural` for `Intl.PluralRules`) is still the old one. Assign `#code`
and `#catalogue` together, bump `#version` once, after both.

If it does not reproduce with the translator off, record that in the commit and
close it — do not carry a ghost.

### F-4 — `MegaMenuNav.svelte:59-67` is a stale two-language toggle

```ts
const nextLang = locale.code === 'hi' ? 'en' : 'hi';
```

The buyer header has a hi/en toggle that silently discards the user's choice of
any of the other 19 languages: a Malayalam user taps it and lands on Hindi.
Replace with the real `LanguageSelector` from `@kalakriti/ui` (already used
elsewhere), and delete `nav.lang.switchedHindi` / `nav.lang.switchedEnglish` in
favour of one `nav.lang.switched` with a `{language}` placeholder.

### F-5 — Correct the false comment in `locales.ts`

`locales.ts:3-5` says "the artisan service worker precaches exactly one message
bundle". There is **no service worker file anywhere in the repo** (`find` for
`*service-worker*`, `sw.ts` → nothing). The stated justification for dynamic
catalogue imports is therefore fiction, and future work planned around it will
be wrong.

Either write the service worker (out of scope here) or correct the comment to
describe what the code actually does. Also note `en.ts`'s `sw.*` namespace (4
keys) is messaging for a service worker that does not exist.

### F-6 — Make the fallback chain honest

`FALLBACK_CATALOGUES` maps every non-`en` code to `{}`, and `CATALOGUE_LOADERS`
has an entry for all 21 codes, so the `??` fallback in `set()` is dead code
today. Once D1 holds, `#lookup`'s `active → hi → en` chain should essentially
never reach `hi`. Keep the chain (it is correct defensive behaviour) but add a
DEV-mode counter that warns when it fires, so a regression is visible in
development rather than shipped.

---

## 4. Phase 3 — Extract the 2,226 hardcoded strings

Measured with a comment-stripped, style-stripped scanner over markup text
nodes, user-visible attributes (`aria-label`, `placeholder`, `title`, `alt`,
`aria-description`), and prose-shaped string literals in `<script>`/`.ts`:

| Area | Hardcoded strings |
|---|---|
| `apps/buyer` | **1640** |
| `apps/artisan` | 374 |
| `apps/admin` | 134 |
| `packages/ui` | 27 |
| `packages/offline` | 10 |
| `packages/voice` | 7 |
| everything else | ~34 |
| **Total** | **2226** across 141 files |

Worst files:

```
182  apps/buyer/src/routes/account/+page.svelte
108  apps/buyer/src/routes/gi-tagged/+page.svelte
107  apps/buyer/src/routes/company/register/+page.svelte
106  apps/buyer/src/lib/stub-listings.ts
102  apps/buyer/src/routes/case-studies/+page.svelte
 96  apps/buyer/src/routes/+page.svelte
 87  apps/buyer/src/lib/MegaMenuNav.svelte
 68  apps/buyer/src/lib/craft-categories.ts
 66  apps/buyer/src/routes/catalog/+page.svelte
 64  apps/buyer/src/routes/fairs/+page.svelte
 62  apps/artisan/src/lib/StallCardModal.svelte
 62  apps/buyer/src/routes/login/+page.svelte
 58  apps/buyer/src/lib/RegionalBeltNavigator.svelte
 55  apps/artisan/src/routes/trends/+page.svelte
 54  apps/buyer/src/lib/CategorySubnav.svelte
 53  apps/buyer/src/routes/contact/+page.svelte
 48  apps/admin/src/routes/companies/+page.svelte
```

Note `apps/buyer/src/lib/ArtisanCraftGrid.svelte` specifically: it imports no
i18n at all and hardcodes `"Indigenous Craft Taxonomy"`, `"Browse by Artisan
Craft Discipline"`, `"View Cluster Directory"`, `"Explore ➔"` — these are the
exact English strings visible in your screenshot 5 while the rest of the page
was in Hindi.

### 4.1 Three kinds of string, three different treatments — decide, don't defer

**(a) UI chrome** — labels, buttons, headings, empty states, errors, aria text.
→ New keys in `en.ts`, then translated. The default; most of the 2,226.

**(b) Seed/demo content** — `stub-listings.ts` (106), `craft-categories.ts` (68),
`apps/artisan/src/lib/ontology.ts` (63 incl. the new `STATE_CODES`),
`craft-icon.ts` (27). This is *content* that in production comes from the
backend, and the backend has no translation columns.

**Decision (do not re-open):** these are demo fixtures that must render in all
20 languages with the backend down (D8), so they become catalogue keys like
everything else — `stub.listing.<id>.title`, `craft.<id>.name`,
`craft.<id>.tagline`, etc. This adds roughly 300–400 keys. It is the only
option that satisfies D8 without inventing a backend translation layer.

Real backend-sourced content (an artisan's actual listing title) is **not**
translated and never was — it renders in whatever language the artisan typed.
That is correct and stays.

**(c) The `hindiName` anti-pattern** — `craft-categories.ts` carries a
`hindiName` field per craft, rendered by `ArtisanCraftGrid.svelte:52` as
`{craft.hindiName}`. This is a hardcoded *second* language baked into a data
structure: it shows Hindi to a Tamil user, forever, regardless of the picker.
Delete the field; the craft name becomes a catalogue key, and the "native name"
line becomes `craft.<id>.nativeName` resolved through the active locale.

Grep for other instances of this shape before starting:
`grep -rn "hindiName\|nameHi\|hi:\s*'" apps packages --include=*.ts`.

### 4.2 Key naming convention

`<surface>.<component>.<element>[.<variant>]`, matching the existing 68
namespaces in `en.ts` (`home` 139 keys, `listing` 110, `footer` 93, `profile`
71, `nav` 64, …). Extend existing namespaces; do not invent parallel ones.
New namespaces only for genuinely new surfaces (`caseStudies`, `giTagged`,
`fairs`, `companyRegister`).

### 4.3 Per-file extraction procedure

For each file, in the batch order of §4.5:

1. `node scripts/i18n-scan.mjs <file>` → the candidate list.
2. Triage each: translate / DNT / false positive (CSS class, URL, test id).
3. Add keys to `en.ts` in the right namespace, alphabetically within it.
4. Replace the literal with `t('key')`; add `const t = $derived(locale.t)` if
   the file lacks it (**note:** `$derived`, not a plain import, or the component
   will not re-render on language change — this is a second, subtler class of
   bug to watch for: files that `import { t }` and call it non-reactively).
   Grep for it: `grep -rn "import {[^}]*\bt\b[^}]*} from '@kalakriti/i18n'" apps packages`
   and check each call site is inside a `$derived`.
5. Attributes: `aria-label={t('...')}`.
6. Interpolation: `t('key', { count })` — never string concatenation, which
   breaks word order in every language whose word order is not English's
   (i.e. all 20 of these, which are verb-final).
7. Re-run the scanner on that file; it must report 0.
8. `pnpm --filter <app> check` must pass.

### 4.4 Commit the scanner as a lint gate (this is D6's teeth)

`web/scripts/i18n-scan.mjs`, wired as `pnpm i18n:lint` and into CI. Without a
committed gate, "not a single thing left untranslated" is true for about a week.
Allow-list false positives in `web/.i18n-scan-ignore` with a required reason
comment per entry, so the escape hatch is auditable.

### 4.5 Batch order (do not deviate — it front-loads what you can see)

| Batch | Files | ~strings |
|---|---|---|
| 3.1 | `packages/{ui,offline,voice}` — shared, affects all 3 apps | 44 |
| 3.2 | `apps/artisan` routes (the accessibility-critical app) | 374 |
| 3.3 | `apps/admin` | 134 |
| 3.4 | `apps/buyer` `lib/` components | ~500 |
| 3.5 | `apps/buyer` routes A–L | ~570 |
| 3.6 | `apps/buyer` routes M–Z | ~570 |
| 3.7 | Data files (§4.1b) | ~300 keys |

After each batch: `pnpm i18n:lint` for that area reports 0; `en.ts` grows;
`i18n-baseline.json` numbers for all 20 locales go **up** (new untranslated
keys) — that is expected and correct, and Phase 5 pulls them back down.

---

## 5. Phase 4 — Freeze the key set, then translate

Do not start translating before Phase 3 completes. Translating 1411 keys ×20
and then adding 400 more keys means doing the tail twice.

At Phase 3 exit, `en.ts` is expected to hold roughly **1,800–1,900 keys**.

### 5.1 Regenerate every catalogue from scratch

Do **not** patch the existing files. They are 47–89% Hindi and patching them
means diffing against a corrupt baseline. Generate fresh from `en.ts`.

### 5.2 Batch shape — this is the part that fails if unspecified

The total is ~1,850 × 20 ≈ **37,000 strings**. A single session attempting a
whole language will truncate, and a truncated generation is exactly how
`ta.ts` ended up full of Hindi in the first place.

Mandatory shape:

- **One locale at a time.** Never interleave.
- **Within a locale, one namespace group at a time**, 100–150 keys per turn.
  Use the namespace census as the unit (`home` 139, `listing` 110, `footer` 93,
  `profile` 71, `nav` 64, `listings` 45, `company` 45, `subnav` 44, `insights`
  44, `clusters` 40, `a11y` 40, `allocation` 37, `ui` 32, `provenance` 31, then
  batch the long tail of 55 smaller namespaces into groups of ~120).
- **After every single batch**, run
  `pnpm --filter @kalakriti/i18n audit --locale <code>` and confirm the issue
  count dropped by exactly the batch size. If it dropped by less, some values
  came back in the wrong script or identical to Hindi — **fix that batch before
  starting the next one.** This is the gate that makes the failure impossible to
  repeat.
- Commit per locale, message: `i18n(<code>): machine catalogue, N keys, audit clean`.

### 5.3 Translation rules given to every batch

1. Target script only — Tamil in Tamil script, Kashmiri in Perso-Arabic, etc.
   Never Devanagari as a stand-in.
2. Every `{placeholder}` preserved exactly, including case.
3. DNT keys (§2.4) copied verbatim from English.
4. Register: plain, respectful, low-literacy-friendly. The artisan app's users
   are the constraint — short sentences, common words, no bureaucratic register.
   `en.ts`'s `literacy.*` namespace (17 keys) documents the intended tone.
5. Plurals per that locale's actual CLDR categories, not English's two.
6. No English left in the value unless the English word is the genuinely
   current usage in that language (e.g. "OTP", "PIN", "SMS"). The existing
   `hi.ts` pattern `"बड़ा कर्सर (Large Cursor)"` — native term with the English
   in parentheses — is good practice for technical UI terms; keep it.
7. RTL locales: no directional characters hardcoded into strings; punctuation
   ordering left to the bidi algorithm.

### 5.4 Locale order

`ta te kn ml` (large speaker bases, distinct scripts, easiest to spot-check) →
`bn gu mr pa or as` → `ur sd ks` (RTL, do together with Phase 6's RTL work) →
`ne kok mai doi sa brx` (Devanagari — **highest risk of Hindi bleed-through,
so the `same-as-hi` check is doing all the work here; do them last, when the
validator is proven**).

### 5.5 Flip the types (D7)

As each locale reaches zero issues:

```ts
export const ta: Messages = { … };   // was Partial<Messages>
```

TypeScript then makes a missing key a compile error permanently, and the
`missing` check in the audit becomes redundant belt-and-braces. Do this
per-locale as each one lands, not in one sweep at the end.

---

## 6. Phase 5 — Verification

### 6.1 Automated

- `pnpm --filter @kalakriti/i18n audit` → 0 issues, all 21 locales.
- `pnpm -r check` → clean (this is what proves D7).
- `pnpm i18n:lint` → 0 hardcoded strings.
- `pnpm -r test` → green, including the rewritten `locales.test.ts`.

### 6.2 New test: render-sweep

A vitest + `@testing-library/svelte` test that, for each of the 21 locales,
mounts every route component and asserts no rendered text node matches a raw
message key (`/^[a-z]+(\.[a-zA-Z]+){1,3}$/`). That is what `#lookup` emits when
a key resolves nowhere, and it is the single most visible failure mode. Cheap
to write, catches D1/D5 violations the type system cannot.

### 6.3 Manual, with the translator off

Per §0a. For 4 spot-check locales (`ta`, `ur`, `sa`, `ml`) walk: login →
language → register → home → listing → orders → profile → accessibility panel,
in all three apps. Screenshot each. Verify:

- no English, no Hindi, no raw keys,
- `<html lang>` and `dir` correct in devtools,
- numbers/dates/currency in the locale's numbering (`formatMoney` already uses
  `numberLocale` — confirm it actually renders Indian digit grouping),
- RTL mirrors correctly.

---

## 7. Backend-down rendering (D8)

Separate this cleanly, because it is two unrelated problems and conflating them
wastes effort:

**Translations are not affected by the backend at all.** Catalogues are bundled
JS chunks served by Vite/NGINX from the same origin as the app. If the page
loaded, the catalogue can load. There is no API call anywhere in the i18n path.
The only backend-down risk is F-2 (a chunk fetch failing on a bad network), and
Phase 2 fixes that.

**What the backend actually takes down is data.** Per `CLAUDE.md`,
`GET /api/v1/listings` returns `{"listings":[]}` because there is no listing
seeder, and `GET /crafts` is what `loadCrafts()` in
`apps/artisan/src/lib/ontology.ts` needs for the craft picker. With the backend
down these fail. Required work:

1. Every `call()` failure path must render a **translated** error/empty state,
   not a raw fetch error and not an English string. Audit `api.error.*` (8 keys)
   and `error.*` (9 keys) in `en.ts` for completeness against the actual
   failure modes in `packages/api/src/transport.ts`.
2. `loadCrafts()` already caches via `@kalakriti/offline`'s `getCached`/
   `setCached` (added in commit `2aa4613`). Verify the cold-cache path — first
   ever visit, backend down, nothing cached — renders a translated empty state
   rather than a spinner forever.
3. Seed/demo content from §4.1(b) is catalogue-resident, so it renders with the
   backend down by construction. That is the main reason that decision went the
   way it did.
4. `packages/offline`'s 10 hardcoded strings and the `offline.*` (15) / `sync.*`
   (8) / `outbox.*` (7) / `network.*` (3) namespaces are the exact strings a
   user sees in this state. Translate them **first** within each locale's batch,
   not last.

---

## 8. RTL and fonts (D9)

`ks`, `sd`, `ur` are `dir: 'rtl'` in `locales.ts`, and `set()` already writes
`document.documentElement.dir`. What is missing:

1. **`ks.ts` is 89% Devanagari.** Kashmiri's official script in J&K is
   Perso-Arabic. Its `LOCALES.ks.script` says `'Arab'`. The catalogue must be
   regenerated in Perso-Arabic; the `wrong-script` check enforces it.
2. **Fonts.** `locales.ts`'s header says scripts are recorded "so the font layer
   can bind a face per script". Verify that font layer exists — check
   `packages/tokens` for `@font-face` per script and a `[lang]`/`[data-script]`
   binding. If it does not exist, it is a task here: 9 distinct scripts
   (Deva, Beng, Guru, Gujr, Orya, Taml, Telu, Knda, Mlym, Arab) need a face,
   with `font-display: swap` and subsetting — this is the artisan app's 2G
   budget, so unsubsetted Noto for 10 scripts is not acceptable.
3. **Layout audit under RTL.** Grep for physical properties that will not mirror:
   `grep -rn "margin-left\|margin-right\|padding-left\|padding-right\|left:\|right:\|text-align:\s*left" apps packages --include=*.svelte`
   → convert to logical properties (`margin-inline-start`, `inset-inline-start`,
   `text-align: start`). This is mechanical and sizeable; budget for it.
4. Directional icons (`arrow-right` in `ArtisanCraftGrid.svelte`, `"Explore ➔"`)
   must flip under RTL.

---

## 9. Execution checklist for the implementing session

Work top to bottom. Do not start a phase before its predecessor's exit gate.

- [ ] **P0** Turn off Chrome's page translation; re-confirm the symptoms.
- [x] **P1.1** `catalogue-audit.ts` with all 8 checks.
- [x] **P1.2** `dnt-keys.ts` -- craft/handloom/gem audit found no proper-noun
      DNT candidates there (they're all translatable prose); seeded instead
      with the 3 real hits the validator itself surfaced: `app.name`,
      `a11y.statement.contact.email`, `profile.email.placeholder`,
      `login.phone.countryCode`.
- [x] **P1.3** `catalogue-audit.test.ts` + `i18n-baseline.json` ratchet.
- [x] **P1.4** Rewrite `locales.test.ts` to derive coverage, not assert a literal.
      Also corrected `locales.ts`'s `coverage` literal for the 10 locales it
      was lying about (hi + the 9 falsely "complete" ones) to `'fallback'`,
      since the rewritten test would otherwise fail honestly forever.
- [x] **P1.5** `scripts/audit.mjs` CLI (+ `audit-main.mjs`, `ts-extension-hook.mjs`
      -- split needed because this package's TS omits import extensions,
      which plain Node ESM can't resolve without a small hook; see the file
      headers).
- [x] **P1 GATE** CLI reproduces §0's table (ta/ml/etc ~10% coverage, sa/ne/
      brx/etc ~6%, hi 77%). Calibration proven. Committed as `38418bd`.
      **Real finding surfaced by the validator, not anticipated by this
      plan:** `en.ts`'s own `nav.lang.switchedHindi` value is Hindi text
      ("भाषा हिन्दी में बदली गई"), not English -- the "source of truth"
      catalogue has a bug in itself. Left unfixed (content work, Phase 3/4
      scope); flagged here so it isn't lost.
      **Also noted:** `en.ts`/`hi.ts` picked up unrelated concurrent edits
      (new `company.*`/`trends.*`/`nav.companies`/`nav.trends` keys) from
      the in-progress b2b/trends feature visible elsewhere in the working
      tree. Not part of this plan's commits; left as-is for that work to
      land on its own.
- [x] **P2** F-1 … F-6:
      - **F-1**: all three `+layout.svelte` now gate first paint on locale
        readiness (buyer/admin: new `localeReady` state flipped by
        `locale.init().then()`, gating `{@render children()}` with a
        `t('state.loading')` fallback; artisan: merged `locale.init()` into
        the existing `booted` `Promise.all`, reusing that app's own gate
        rather than adding a parallel one).
      - **F-2**: `locale.svelte.ts`'s `set()` now catches a failed catalogue
        `import()` and falls back to `FALLBACK_CATALOGUES[code] ?? {}` --
        `#code` and persistence still happen, so a language switch can no
        longer be a silent no-op. DEV-mode `console.warn` on the fallback
        path; no toast plumbing added (would need a new i18n->ui dependency
        the package doesn't have and shouldn't grow for this).
      - **F-3**: does not reproduce. Verified live in the artisan app (dev
        server, backend down) with Chrome's translator not a factor (Browser
        pane tool, no translate extension): selected Hindi from `/language`
        fresh -> `/welcome` -> `/login` all rendered fully in Hindi, then
        switched live to Sanskrit from the header's language selector --
        `html[lang]` and every string on screen matched sa-IN consistently,
        no Hindi/English bleed-through at any point. Screenshot 1's anomaly
        is fully explained by Chrome's page-translate corruption (§0a) and
        nothing else; closed.
      - **F-4**: investigated, not fixed -- `MegaMenuNav.svelte`'s hi/en
        toggle turned out to be **dead code**: `grep` for the component name
        across the whole repo finds only its own file, no import site
        anywhere. It is not wired into any route, so it cannot affect a real
        user. Fixing an unreachable component's internals would be pure
        speculative work; flagged here instead per the "notice dead code,
        don't delete it uninvited" rule. If this component is ever wired up,
        replace `toggleLang`'s hi/en toggle with `<LanguageSelector />`
        (already the pattern in all three real headers) before it ships.
      - **F-5**: corrected the false "artisan service worker precaches
        exactly one message bundle" claim in `locales.ts`, and found it is
        **not simply false but actively misleading in a new way**: a real
        Workbox service worker does exist per app (`vite-plugin-pwa`,
        configured in each `vite.config.ts`, not a hand-written file) -- but
        its `globPatterns: ['**/*.js', ...]` enumerates the build OUTPUT
        directory at build time, with no runtime concept of "the selected
        language," so the claim that it "picks up the active [locale] one
        and leaves the rest to runtime caching" (artisan's own
        `vite.config.ts` comment, separate from `locales.ts`) is very likely
        false too -- probably every locale chunk gets precached, not one.
        Could not get clean confirmation from an actual build: the
        checked-in `apps/artisan/build/` output is stale/inconsistent (its
        `sw.js` precache manifest is shaped like a Node-adapter SSR build --
        `server/nodes/*.js` entries -- while `svelte.config.js` currently
        specifies `adapter-static`; the two don't match, so that artifact
        predates the current adapter config and proves nothing about it).
        Both comments corrected to state what's verified vs. suspected-but-
        unverified, rather than repeating the original confident claim.
        Actually confirming/fixing the PWA caching strategy is out of scope
        here -- flagged as a real, separate follow-up.
      - **F-6**: `#lookup` now emits one DEV-only, deduped-per-key
        `console.warn` the first time a non-hi/en locale falls through to
        Hindi/English for a given key, distinct from the existing
        "missing everywhere" warning. Noisy today by design (almost nothing
        is translated yet) -- it becomes a real regression signal once a
        locale reaches 'machine'/'complete' in Phase 4.
- [x] **P2 GATE** Verified live (Browser pane, artisan + buyer dev servers,
      backend down): language switch is instant with no flash (booted/
      localeReady gates hold), survives page load and a live in-session
      switch identically, and a failed-chunk-load path is covered by F-2's
      catch even though a real chunk failure wasn't separately staged (hard
      to force a mid-flight 404 against a live Vite dev server without
      tampering with the network layer -- code-reviewed instead: the
      try/catch is unconditional around the one `await loader()` call).
- [ ] **P3** Batches 3.1 → 3.7, `pnpm i18n:lint` = 0 after each.
  - [x] **3.1** `packages/{ui,offline,voice}` -- **one real fix, everything
        else was scanner noise.** Manually read all ~17 flagged files
        (the automated scan's "44 strings" estimate for this batch was
        ~98% false positives: CSS class-name template literals, `event.key`
        comparisons, IndexedDB schema strings, doc-comment usage examples,
        and internal `Error()` messages that are caught and swallowed with
        a bare `catch {}` everywhere they're used -- never rendered to a
        user). This recalibrates the plan's global 2,226-string estimate
        downward for `.ts` files specifically; markup-heavy `.svelte` route
        files (batches 3.2-3.6) had a much higher real-hit ratio in the
        original scan and are the actual long pole.
    - Fixed: `Breadcrumbs.svelte` hardcoded `aria-label="Breadcrumb"` and
      defaulted `homeLabel = 'Home'` in the component itself -- both now
      use `t('breadcrumb.label')` / `t('nav.home')`, keys that already
      existed in `en.ts` unused. Verified live (buyer dev server): the
      aria-label now correctly falls through the hi->en chain today (no
      `breadcrumb.label` in `hi.ts` yet) and will pick up the real
      translation the moment Phase 4 adds it -- no code change needed then.
    - Confirmed false positives, no fix needed: `commands.ts`'s per-locale
      voice-command grammar (`GRAMMAR.en`/`GRAMMAR.hi`) is not UI text, it's
      speech-recognition match phrases -- `t()` doesn't apply to it at all.
      It already follows the same "en+hi built, rest fall back to en"
      pattern as the message catalogues; expanding it to more locales is
      real translation work but a *different* artifact from a catalogue key
      and needs its own pass, not folded into Phase 4. `listen.ts`/
      `speak.ts`'s `Error('... is not available')` messages are internal,
      never reach a user (every call site swallows them in a bare `catch`).
      `SpeakButton.svelte`'s actual user-facing unavailability message
      already correctly uses `t('voice.unavailable', {...})`.
    - **New finding, out of this batch's scope, flagged for 3.4-3.6**: six+
      buyer routes (`privacy`, `terms`, `case-studies`, `catalog`, `fairs`,
      `journal/[slug]`) build their own `{ label: t('nav.home') || 'Home',
      href: '/' }` as `items[0]` *and* pass `homeLabel="Marketplace"` to
      `<Breadcrumbs>` -- which already renders its own leading home link
      unconditionally. Result: two "home" links in every one of these
      breadcrumb trails (confirmed live: `/privacy` renders "Marketplace"
      then a second link "मुख्य पृष्ठ" pointing at `/` again). Pre-existing,
      not caused by this batch's fix. Also note the dead `t(key) || 'X'`
      fallback idiom repeated at every one of these call sites -- `t()`
      never returns falsy for a real key, so the `|| 'X'` never fires; harmless
      but worth deleting when these files are touched for real in 3.4-3.6.
  - [x] **3.2** `apps/artisan` -- ~40 files manually reviewed against the
        scanner's flagged-string list; real hardcoded-string hits were
        concentrated in a handful of modal/wizard components, the rest
        (`.ts` files, already-wired `.svelte` pages) were scanner noise
        (CSS classes, type literals, comments, doc examples), confirming
        3.1's recalibration held for this app too.
    - Fixed (wired existing/new `t()` keys, no prose left hardcoded):
      `Breadcrumbs.svelte` (3.1), `StallCardModal.svelte` (11 new
      `exhibition.stallCard.*` keys), `profile/+page.svelte` (13 new keys
      plus 4 previously-orphaned keys wired in), `IncomeGrowthChart.svelte`
      (restructured `QUARTERS` to carry `MessageKey`s instead of hardcoded
      period/highlight strings, 13 new `growth.*` keys), `+page.svelte`
      (home, 1 new key), `BusinessCardModal.svelte` (12 new `card.*` keys:
      subtitle, ministry line, PM Vishwakarma badge, QR alt, scan-to-buy,
      DBT/GI line, copy-link/WhatsApp button labels, clipboard toast,
      parametrized share message, ID label), `DigitalLiteracyTutorial.svelte`
      (8 new `literacy.tutorial.*` keys: per-step badges pulled out of the
      hardcoded bilingual `STEPS` array into `badgeKey: MessageKey`, sahayak
      badge, "Listen in Hindi" label, voice hint, dot aria-label), `listing/
      new/studio/+page.svelte` (13 new `listing.studio.*` keys: aria-labels,
      view-mode tab labels, before/after alt text and captions, background
      pill descriptions, lighting toggle Active/Off), `ImageCropModal.svelte`
      (1 key: crop preview alt text), `GemExportPreview.svelte` (4 keys:
      marketplace name/sub used both in the rendered card header and in the
      plain-text export, so one key pair drives both surfaces).
    - Confirmed false positives, no fix needed (scanner flagged, manual
      review found the file already fully wired or the "hit" was a type
      literal / enum value / internal `Error()` never rendered):
      `login/+page.svelte`, `offer/+page.svelte`, `lots/[lotId]/+page.svelte`,
      `listings/[id]/+page.svelte`, `listings/[id]/provenance/+page.svelte`,
      `listing/new/processing/+page.svelte`, `verify/+page.svelte`,
      `register/district/+page.svelte`, `offline/+page.svelte`, `listing/new/
      review/+page.svelte`, `listing/new/pricing/+page.svelte`, `language/
      +page.svelte`, `HandloomVerdict.svelte`, `register/pehchan/+page.svelte`,
      `register/cluster/+page.svelte`, `listings/+page.svelte`, `earnings/
      +page.svelte`, `SahayakTooltip.svelte`, `RegisterStep.svelte`,
      `PwaUpdatePrompt.svelte`, `listing-draft.ts`, `listings.ts`, `orders.ts`,
      `route-guard.ts`, `provenance.ts`, `outbox-send.ts`.
    - Deliberately deferred to **3.7 (data pass)**, not UI chrome: `demo-
      listing.ts` (sample listing title/description/attribute values, canvas
      GI-seal text -- demo content, not translatable interface text),
      `ml-mock.ts` (fabricated ML pipeline output explicitly marked for
      replacement per `ml_wiring.md` -- mock AI descriptions, not product
      copy), the FAIRS array and demo-default state values in
      `StallCardModal.svelte`/`profile/+page.svelte` (flagged in 3.1/3.2
      review, unchanged), `district.name`/`district.state` and other
      `ontology.ts`-sourced data rendered as-is in `register/district`.
      `trends/+page.svelte` and `ontology.ts` remain skipped (pre-existing
      concurrent WIP / dead-data-table, per the session's established rule).
    - en.ts grew 1411 → 1501 keys this batch. `i18n-baseline.json` ratchet
      ceilings raised accordingly for every non-en locale (each is "missing"
      every new key until its own Phase 4 translation batch lands) --
      verified via `pnpm --filter @kalakriti/i18n run audit`, no ceiling
      raised without a matching real key increase in `en.ts`.
  - [x] **3.3** `apps/admin` -- **zero fixes needed.** Manually read every
        route (`+layout`, dashboard, accessibility, clusters, crafts,
        insights, moderation, login, login/verify, +error) and every `lib/`
        component (`BarChart`, `Breadcrumb`, `CommandPalette`, `nav.ts`,
        `demo.ts`, `bulk-onboard.ts`, `csv.ts`, `onboard-validation.ts`).
        F-1's `localeReady` gate was already in place on `+layout.svelte`
        from Phase 2. Every visible string already routes through `t()`;
        the plan's ~134-string estimate for this app doesn't hold up under
        manual review -- the only literal-string hits were a placeholder
        example (`"IN-GJ"`, a state-code format hint, not prose) and a
        doc-comment usage example on `BarChart.svelte`, whose real
        suppressed-value string is correctly `t('insights.suppressed')`.
        `apps/admin/src/routes/companies/` is untracked concurrent WIP
        (b2b/trends feature) and out of scope, per the session's standing
        rule on pre-existing uncommitted work. No commit for this batch --
        nothing changed.
  - [x] **3.4** `apps/buyer/src/lib` (17 components) -- real hit rate was
        much higher here than in `.ts` files or already-audited routes;
        several components had **no `locale`/`t` import at all** and were
        entirely hardcoded English.
    - Fixed: `CategorySubnav.svelte` (46 hardcoded mega-menu link labels
      across the Home & Living / Furniture / Paintings dropdowns, pulled
      into 46 new `subnav.*` leaf keys -- craft names/hindiNames from
      `ARTISAN_CRAFT_CATEGORIES` left as data), `AccountMenu.svelte`
      (imported `locale` for the first time; 25 new/reused `accountMenu.*`
      keys covering the entire header popover -- greeting, sign-in, both
      portal-grid columns, sign-out), `SellerShowcaseBanner.svelte` (kicker/
      title/subtitle, 3 perk cards, CTA, testimonials header, 6 government
      accreditation strip labels -- the 3 testimonial quotes/bios left as
      data), `ListingCard.svelte` (21 new keys: alt text, GI-pin and
      wishlist/share titles+aria-labels, toasts, artisan portrait alt,
      verified-artisan title, discount badge, Order/Wishlist CTA labels),
      `ArtisanConciergeModal.svelte` (14 new keys: success note
      parametrized with `{name}`, both input placeholders, 6 craft-option
      labels, window label + 3 time-slot options), `FaqAccordion.svelte`
      (imported `locale` for the first time; all 5 Q&A pairs plus header
      chrome moved from a hardcoded array to `MessageKey` references -- 19
      new `faq.*` keys, the longest-prose fix in this batch), `VoicesRe
      elCarousel.svelte` (6 new keys: dialog title/name/badge fallbacks,
      Mute Voice, Previous, Next -- the `FALLBACK_STORIES` demo array left
      as data), `RegionalBeltNavigator.svelte` (5 new keys: register label,
      sidebar header/detail, both metric labels -- the `BELTS` array's
      state names and featured-craft names/GI-years left as data),
      `InstitutionalProcurementBanner.svelte` (9 new keys: 3 tier-tag
      labels via one parametrized key, and the entire downloadable
      `.txt` catalogue content -- title, ministry line, heading, 3 tier
      lines, provenance/GST lines -- mirroring the `t()`-per-line pattern
      already used in `GemExportPreview.svelte`'s plain-text export),
      `ArtisanCraftGrid.svelte` (imported `locale` for the first time; 6
      new keys: kicker, title, subtitle, directory link, parametrized GI-
      hub count, Explore CTA).
    - Confirmed false positives / no fix needed: `BuyerFooter.svelte`
      (already fully wired -- the flagged hits were proper-noun aria-labels
      like "Facebook"/"LinkedIn" and literal phone/email contact data, both
      correctly left untranslated), `PurchaseForm.svelte`,
      `ProvenanceTerminal.svelte`, `DisputeDialog.svelte`,
      `CurrencySelector.svelte`, `AtelierCommissionCard.svelte`.
    - Skipped: `MegaMenuNav.svelte` -- confirmed dead code (zero import
      sites) in an earlier phase; not worth localizing code nothing
      renders.
    - en.ts grew 1501 -> 1671 keys. `i18n-baseline.json` ceilings raised to
      match (`pnpm --filter @kalakriti/i18n run audit` re-run before
      writing each number). `npx vitest run` and `apps/buyer` svelte-check
      both clean.
  - [x] **3.5** `apps/buyer` routes A-L (`account`, `gi-tagged`, `contact`,
        `card/[slug]`, root home `+page.svelte`, `case-studies`, `+layout`,
        `catalog`, `journal/[slug]`, `fairs`, `artisan/[slug]`, `craft/[slug]`)
        -- same pattern as 3.4: several route files had zero `locale`/`t`
        import at all despite being almost entirely hardcoded English.
    - Fixed: `account/+page.svelte` (2346-line file, ~90 new
      `account.page.*`/`account.hub.*` keys -- svelte:head, 6 hub cards, tab
      nav, and all 5 panels: personal details, addresses, security, orders
      chrome, consultations chrome, plus the phone-change and add-address
      modals; the two seed order/consultation cards' product/artisan/date/
      price content and the state/native-language `<option>` lists left as
      data), `gi-tagged/+page.svelte` (24 new `giTagged.*` keys -- view
      switchers, items counter, state/sort selects, no-results panel,
      registration-number title, share/copy actions, plus reused
      `listingCard.discountPercent`/`removeFromWishlist`/`saveToWishlist`/
      `wishlistAriaLabel`/`saved`/`wishlist` from 3.4 for the product-card
      action row -- `GI_PRODUCTS` and the filter-option arrays left as
      data), `contact/+page.svelte` (imported `locale` for the first time
      -- wait, it already had `t`, but nearly the entire page was
      unwired; ~55 new `contact.*` keys covering the 3 help cards, the
      full inquiry form incl. all 6 select options, and the ministry
      office panel -- the physical address and toll-free/email contact
      literals left untranslated per the established proper-noun-contact-
      data rule), `card/[slug]/+page.svelte` (imported `locale` for the
      first time; 24 new `shareCard.*` keys for the standalone shareable
      business-card route, reusing `card.idLabel` and
      `account.page.addresses.country` -- the 3 fallback sample catalogue
      pieces and the catch-block demo artisan bio left as data), root
      `+page.svelte` (10 new keys: hero edition/slide aria-labels, sr-only
      loading text, 3 journal cluster-tags + leads, and the entire 3-column
      directory/case-studies banner under `home.directory.*`),
      `case-studies/+page.svelte` (19 new `caseStudies.*` keys for all
      chrome around the `CASE_STUDIES` data array -- kicker, income-bar
      labels, comparison-table headers, 3 narrative headings, 2 action
      links -- the case-study content itself, quotes, and artisan names
      left as data), `+layout.svelte` (10 new keys: hamburger aria-labels,
      both nav lists' 5 link labels reused across desktop+mobile, meta
      description -- the `ld+json` structured-data block left untouched as
      non-visible SEO data), `catalog/+page.svelte` (17 new `catalog.*`
      keys -- kicker/title/subhead, search box, 6 belt-filter chips,
      certified-lots link -- `FALLBACK_CRAFTS` left as data),
      `journal/[slug]/+page.svelte` (9 new `journal.*` keys for the
      breadcrumb/head-title/kicker/not-found/back-link chrome around the
      `ESSAYS` data array -- essay bodies, quotes, and bylines left as
      data), `fairs/+page.svelte` (11 new `exhibition.fairs.*` keys --
      breadcrumb, gov badges, 3 filter-tab labels, live/upcoming status
      badges, crafts label, artisan-count suffix -- the `FAIRS` array left
      as data), `artisan/[slug]/+page.svelte` (6 new `artisanStorefront.*`
      keys for the fair/stall welcome banner's parametrized copy).
    - Confirmed false positive / no fix needed: `craft/[slug]/+page.svelte`
      -- already fully wired through `t()`; every string the scanner
      flagged was either already a `t()` call or live API data.
    - Spot-checked zero-hit files for false negatives: `feed/+page.svelte`,
      `bulk-order/+page.svelte`, `assets/+page.svelte`,
      `accessibility/+page.svelte`, `+error.svelte` -- all clean.
    - en.ts grew 1671 -> 1988 keys. `i18n-baseline.json` ceilings raised to
      match (`pnpm --filter @kalakriti/i18n run audit` re-run before
      writing each number; the CLI's own exit code is always 1 whenever any
      locale has outstanding issues -- that's expected pre-Phase-4 noise,
      not a ratchet failure. The real gate is `npx vitest run`'s
      `catalogue-audit.test.ts`, which reads the baseline and passed clean
      at 46/46). `apps/buyer` svelte-check also clean (767 files, 0 errors).
  - [x] **3.6** `apps/buyer` routes M-Z (`listing/[slug]`, `login`, `orders`,
        `orders/[id]`, `orders/thank-you`, `privacy`, `profile`, `search`,
        `terms`, `verify/[code]`) -- lighter batch: two full legal documents
        and one entirely-unwired multi-portal gateway page accounted for
        most of the new keys; several route files were already fully wired
        from earlier phases and needed zero changes.
    - Fixed: `privacy/+page.svelte` (23 new `privacy.*` keys -- the complete
      DPDP Act 2023 policy document, all 6 sections), `terms/+page.svelte`
      (17 new `terms.*` keys -- the complete Terms of Service document, all
      6 sections), `login/+page.svelte` (52 new `login.*` keys -- the
      entire 3-portal sign-in gateway: toast messages, brand header, portal
      tabs, buyer auth form incl. OTP/password modes, and both artisan/admin
      portal-redirect panels with their feature lists), `orders/thank-you/
      +page.svelte` (26 new `orderConfirmation.*` keys -- kicker, seal box,
      guild pledge card, all 4 roadmap steps, both CTA buttons; the prefix/
      suffix split pattern from 3.5's `account.page.orders.craftedByLabel`
      reused twice here to keep `<strong>{orderId}</strong>` and
      `<strong>{artisanName}</strong>` inline within their sentences -- the
      URL-param demo fallback values for order id/craft/artisan/cluster
      left as data), `orders/+page.svelte` and `orders/[id]/+page.svelte`
      (reused the existing `orders.units` key for two `{quantity} units`
      spans that had been left as raw string concatenation), `listing/
      [slug]/+page.svelte` (9 new `listing.*` keys for the share button,
      response-guarantee promise card, artisan-profile link title, and
      quick-order-dock aria-label -- reused `listing.by` and
      `profile.verifiedBadgeTitle` rather than duplicating), `profile/
      +page.svelte` (1 new key for the redirect-notice text).
    - Confirmed false positives / no fix needed: `search/+page.svelte` and
      `verify/[code]/+page.svelte` -- both already fully wired through
      `t()`.
    - Found and fixed one duplicate-key regression during this batch: a new
      `login.heading` collided with a pre-existing key of the same name
      (artisan-app phone-login flow, `'Enter your phone number'`) --
      svelte-check caught it as a TS object-literal error immediately.
      Renamed the new key to `login.gatewayHeading` rather than touching
      the pre-existing one.
    - en.ts grew 1988 -> 2111 keys. `i18n-baseline.json` ceilings raised to
      match. `npx vitest run` (46/46) and `apps/buyer` svelte-check (767
      files, 0 errors) both clean.
- [ ] **P3 GATE** `en.ts` frozen. Record the final key count here: ______
- [ ] **P4** 20 locales × namespace batches, audit after every batch, type flip
      per locale.
- [ ] **P4 GATE** `audit` = 0 issues for all 21; `pnpm -r check` clean.
- [ ] **P5** Render-sweep test; manual 4-locale walk with screenshots.
- [ ] **P6** RTL + fonts.
- [ ] **P7** Set `coverage: 'machine'` for all 20; `'complete'` only for `en`.
      Update `CLAUDE.md` with the audit command and the ratchet convention.

---

## 10. Estimate, honestly

| Phase | Size |
|---|---|
| P1 validator | ~400 lines, 1 session |
| P2 runtime fixes | ~150 lines across 8 files, 1 session |
| P3 extraction | 2,226 strings, 141 files — **the long pole**, 6–10 sessions |
| P4 translation | ~37,000 strings, ~250 batches — 15–25 sessions |
| P5 verification | 2 sessions |
| P6 RTL + fonts | 2–3 sessions |

P4 is mechanical but unskippable, and its per-batch audit gate is the only
thing standing between this plan and a repeat of `2aa4613`. If a batch's issue
count does not drop by exactly the batch size, **stop and fix that batch**.
Everything in §0 happened because nobody checked.
