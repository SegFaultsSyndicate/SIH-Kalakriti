/*
 * Kalakriti — Batch 8 validation script
 *
 * Unlike each batch's own gen-batchN.mjs, which checks files as it writes
 * them, this walks every SVG actually on disk across every package and
 * validates it against Batch 8's rules. That is the only thing that catches
 * drift: a file hand-edited after generation, an older Batch 2 asset that
 * predates any of this check discipline, a regeneration that silently
 * changed a rule.
 *
 * Run:  node scripts/validate-all.mjs
 * Exits non-zero on any failure.
 */

import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, extname } from 'node:path';
import { dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');

const CATEGORY = {
  'packages/icons/src': { name: 'icon', budget: 2048, viewBoxExact: '0 0 24 24', strokeWidthExact: '1.5' },
  'packages/illustrations/src': { name: 'illustration', budget: 6144, maxColours: 2 },
  'packages/patterns/src': { name: 'pattern/texture', budget: 2048 },
  'packages/identity/src': { name: 'identity', budget: 8192 },
  'packages/ornament/src': { name: 'ornament', budget: 2048 },
  'packages/motion/src': { name: 'motion', budget: 2048 },
};

let failed = 0, checked = 0;
const fail = (f, m) => { console.error('FAIL  ' + f + ': ' + m); failed++; };

// Known, deliberate exceptions — documented here rather than silently
// loosening the rule the file would otherwise trip.
const EXCEPTIONS = {
  'packages/identity/src/seal.svg': ['referenced-id'], // <title id> referenced by aria-labelledby — legitimate, Batch 0's own rule
  'packages/patterns/src/paper-grain.svg': ['referenced-id'], // <filter id="k-grain"> referenced by its own url(#k-grain)

  // Platform-chrome icons: rendered by the OS/browser outside any document
  // that could supply a CSS custom property (browser tab, PWA home screen,
  // iOS touch icon). currentColor/var() cannot resolve there, so a bare
  // brand hex is the correct choice, not a violation of the CSS-context rule.
  'packages/identity/src/favicon.svg': ['bare-hex'],
  'packages/identity/src/app-icon-maskable.svg': ['bare-hex'],
  'packages/identity/src/app-icon-maskable-mono.svg': ['bare-hex'],
  'packages/identity/src/apple-touch-icon.svg': ['bare-hex'],

  // Sprite-defs container (Batch 3): <pattern> ids are consumed elsewhere as
  // fill="url(#k-jaali-hex-24)" after this whole file is inlined once per
  // document — see the file's own header comment. width="0" height="0" and
  // no viewBox are deliberate; it renders nothing standalone. Not a
  // per-asset budget case: it bundles every jaali/blockprint/rule tile.
  'packages/ornament/src/patterns.svg': ['sprite'],

  // Hand-authored verified-badge emblem (user-supplied SVG Repo artwork, see
  // packages/icons/src/badge.svg). Kept verbatim at the user's request: a
  // 96x96 single-colour currentColor emblem, deliberately NOT a 24x24 stroke
  // icon, so the icons category's exact-viewBox and 2KB budget rules don't
  // apply to it. Its shape is re-validated instead by gen-batch4.mjs.
  'packages/icons/src/badge.svg': ['emblem'],

  // Full-lockup identity compositions, not single marks — the 8KB budget
  // was set with individual assets (logomark, seal, app icons) in mind,
  // all of which land under 1KB. These four multi-glyph compositions
  // (custom wordmark letterforms, fine crest linework, mark+wordmark+
  // ministry-name lockup) are already SVGO-optimised to 2-decimal
  // precision; the remaining size is real path data, not bloat. Documented
  // over-budget rather than quietly raising the identity budget for
  // everything in the package.
  'packages/identity/src/wordmark-horizontal.svg': ['budget'],
  'packages/identity/src/wordmark-stacked.svg': ['budget'],
  'packages/identity/src/heritage-crest-fine.svg': ['budget'],
  'packages/identity/src/emblem-lockup.svg': ['budget'],
};

function relPath(p) { return p.split(join(ROOT, '')).join('').replace(/\\/g, '/').replace(/^\//, ''); }

for (const [dir, rules] of Object.entries(CATEGORY)) {
  const full = join(ROOT, dir);
  let files;
  try { files = readdirSync(full).filter((f) => extname(f) === '.svg'); }
  catch { console.log('  (skip ' + dir + ' — not found)'); continue; }

  for (const file of files) {
    checked++;
    const path = join(full, file);
    const rel = relPath(path);
    const exceptions = EXCEPTIONS[rel] || [];
    const raw = readFileSync(path, 'utf8');
    const geom = raw.replace(/<!--[\s\S]*?-->/g, '');
    const bytes = Buffer.byteLength(raw, 'utf8');

    if (!exceptions.includes('sprite') && !exceptions.includes('emblem')) {
      if (!/viewBox="[^"]+"/.test(geom)) fail(rel, 'no viewBox');
      if (rules.viewBoxExact && !geom.includes('viewBox="' + rules.viewBoxExact + '"'))
        fail(rel, 'viewBox is not exactly "' + rules.viewBoxExact + '"');
      if (/<svg[^>]*\swidth=/.test(geom)) fail(rel, 'hardcoded width on root svg');
    }
    if (!exceptions.includes('bare-hex') && /#[0-9a-fA-F]{3,6}\b/.test(geom.replace(/var\([^)]*\)/g, '')))
      fail(rel, 'bare hex colour outside a var() fallback');
    if (/<animate|<set\b/.test(geom)) fail(rel, 'SMIL animation (<animate>/<set>) — CSS only');
    const commentBodies = [...raw.matchAll(/<!--([\s\S]*?)-->/g)].map((m) => m[1]);
    if (commentBodies.some((body) => body.includes('--')))
      fail(rel, 'literal "--" inside an XML comment (invalid XML; the Batch 6 SVGO failure)');

    const ids = [...geom.matchAll(/\bid="([^"]+)"/g)].map((m) => m[1]);
    const labelledbyValues = [...geom.matchAll(/aria-labelledby="([^"]+)"/g)].flatMap((m) => m[1].split(/\s+/));
    for (const id of ids) {
      const referenced = geom.includes('url(#' + id + ')') || labelledbyValues.includes(id);
      if (!referenced && !exceptions.includes('referenced-id') && !exceptions.includes('sprite'))
        fail(rel, 'unreferenced id "' + id + '"');
    }

    if (rules.strokeWidthExact) {
      const strokePaths = geom.match(/<path[^>]*stroke="currentColor"[^>]*>/g) || [];
      for (const sp of strokePaths) {
        if (!new RegExp('stroke-width="' + rules.strokeWidthExact + '"').test(sp))
          fail(rel, 'stroke-width is not the literal ' + rules.strokeWidthExact);
        if (!/stroke-linecap="round"/.test(sp) || !/stroke-linejoin="round"/.test(sp))
          fail(rel, 'not round cap/join (icons are UI chrome, not a geometric motif)');
      }
    }

    if (rules.maxColours) {
      const colours = new Set([...geom.matchAll(/(?:fill|stroke)="(currentColor|var\([^)]+\))"/g)].map((m) => m[1]));
      if (colours.size > rules.maxColours) fail(rel, colours.size + ' colours used, ' + rules.maxColours + '-colour maximum');
    }

    if (!exceptions.includes('sprite') && !exceptions.includes('emblem') && !exceptions.includes('budget') && bytes > rules.budget)
      fail(rel, bytes + 'B over the ' + rules.budget + 'B ' + rules.name + ' budget');
  }
}

console.log('\n' + checked + ' SVGs checked across ' + Object.keys(CATEGORY).length + ' packages.');
if (failed) {
  console.error(failed + ' failure(s).');
  process.exit(1);
}
console.log('All pass.');
