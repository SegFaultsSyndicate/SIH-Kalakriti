/*
 * Kalakriti assets — Batch 7 generator
 * Motion and state assets. The charkha spinner (Batch 4) and skeleton
 * shimmer (Batch 6) already satisfy two of this batch's deliverables —
 * referenced here, not rebuilt. New in this batch: the ML pipeline stage
 * glyphs, sync-state marks, success/failure stitch marks, and the seal
 * draw-on outline.
 *
 * Run:  node scripts/gen-batch7.mjs
 */

import { writeFileSync, readFileSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const SRC = join(ROOT, 'packages/motion/src');
mkdirSync(SRC, { recursive: true });

const n = (v) => String(Math.round(v * 100) / 100);
const pt = (x, y) => n(x) + ' ' + n(y);
const line = (x1, y1, x2, y2) => 'M' + pt(x1, y1) + ' L' + pt(x2, y2);
const poly = (pts, close) => 'M' + pts.map((p) => pt(p[0], p[1])).join(' L ') + (close ? ' Z' : '');
const circ = (cx, cy, r) =>
  'M' + pt(cx - r, cy) + ' A' + n(r) + ' ' + n(r) + ' 0 1 0 ' + pt(cx + r, cy) +
  ' A' + n(r) + ' ' + n(r) + ' 0 1 0 ' + pt(cx - r, cy);
const dot = (cx, cy, r) => circ(cx, cy, r) + ' Z';
const rad = (deg) => (deg * Math.PI) / 180;
const arc = (cx, cy, r, a0, a1, sweep = 1) => {
  const x0 = cx + r * Math.cos(rad(a0)), y0 = cy + r * Math.sin(rad(a0));
  const x1 = cx + r * Math.cos(rad(a1)), y1 = cy + r * Math.sin(rad(a1));
  const large = ((sweep ? a1 - a0 : a0 - a1) + 360) % 360 > 180 ? 1 : 0;
  return 'M' + pt(x0, y0) + ' A' + n(r) + ' ' + n(r) + ' 0 ' + large + ' ' + sweep + ' ' + pt(x1, y1);
};

const assets = [];

function addIcon(name, comment, body, opts = {}) {
  const svg =
    '<?xml version="1.0" encoding="UTF-8"?>\n' +
    '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">\n' +
    '  <!-- ' + comment + ' -->\n' +
    '  ' + body + '\n' +
    '</svg>\n';
  assets.push({ name, svg, w: 24, h: 24, ...opts });
  return svg;
}

const S = (d, sw = 1.5) => '<path fill="none" stroke="currentColor" stroke-width="' + sw + '" stroke-linecap="round" stroke-linejoin="round" d="' + d + '"/>';
const F = (d) => '<path fill="currentColor" d="' + d + '"/>';

/* ==================================================== PIPELINE STAGE GLYPHS
 * enhance -> attributes -> describe -> translate. Standalone glyphs; the
 * per-stage pending/active/complete/failed VISUAL is drawn by CSS around
 * the glyph (a ring + state overlay in motion.css), not baked into the SVG
 * — one glyph file per stage, not sixteen combinatorial states.
 */

addIcon('pipeline-enhance',
  'ML pipeline, stage 1: enhance. A sparkle/wand — the image-enhancement step. State (pending/active/complete/failed) is drawn by motion.css around this glyph, not inside it.',
  S(line(12, 3, 12, 7)) + S(line(12, 17, 12, 21)) + S(line(4.5, 8, 7.5, 9.8)) + S(line(16.5, 14.2, 19.5, 16)) +
  S(poly([[9, 15], [10.4, 11.4], [14, 10], [10.4, 8.6], [9, 5], [7.6, 8.6], [4, 10], [7.6, 11.4]], true), 1.2));

addIcon('pipeline-attributes',
  'ML pipeline, stage 2: attributes. A tag with a punch hole — the material/technique tagging step.',
  S('M4 12 L11 5 L19 5 L19 13 L12 20 Z') + S(dot(15.5, 8.5, 1.2)));

addIcon('pipeline-describe',
  'ML pipeline, stage 3: describe. Text lines forming — the description-generation step.',
  S(line(5, 7, 19, 7)) + S(line(5, 12, 16, 12)) + S(line(5, 17, 12, 17)));

addIcon('pipeline-translate',
  'ML pipeline, stage 4: translate. Two overlapping speech bubbles — the multi-language translation step.',
  S(poly([[3, 5], [14, 5], [14, 13], [8, 13], [5.5, 15.5], [5.5, 13], [3, 13]], true), 1.3) +
  S(poly([[10, 9], [21, 9], [21, 17], [18.5, 17], [18.5, 19.5], [16, 17], [10, 17]], true), 1.3));

/* =========================================================== SYNC STATES
 * offline reuses Batch 4's icons/offline.svg (not duplicated here — see
 * motion.css). pending, syncing (thread being drawn) and synced
 * (completed stitch) are new: small, header-scale, non-distracting.
 */

addIcon('sync-pending',
  'Sync state: pending. A dotted, unfilled ring — queued but not yet moving. Distinct from the charkha spinner (Batch 4) so "queued" never reads as "in progress".',
  '<path fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-dasharray="0.1 4" d="' + circ(12, 12, 7) + '"/>');

addIcon('sync-syncing',
  'Sync state: syncing. A thread being drawn — motion.css animates this path\'s stroke-dashoffset from full to zero (the "thread drawing itself in") rather than rotating a spinner, so syncing reads distinctly from the charkha loading indicator.',
  S('M4 16 Q9 8 12 12 Q15 16 20 8', 1.6));

addIcon('sync-synced',
  'Sync state: synced. A completed stitch — two short crossing strokes, static. Also reused as the general "success" moment (see motion.css) rather than drawing a second near-identical mark.',
  S(line(5, 14, 10, 19)) + S(line(10, 19, 19, 6)));

/* ==================================================== SUCCESS / FAILURE
 * success reuses sync-synced.svg above. failure is new: a broken thread.
 */

addIcon('failure-thread',
  'Failure moment: a broken thread — one continuous line that snaps and the two ends drop out of alignment, brief. Static by default; motion.css adds a quick separate-and-settle on the two ends, under the 400ms UI-feedback cap.',
  S('M4 7 Q8 9 9.5 11') + S('M14.5 13 Q17 15.5 20 17') + F(dot(9.5, 11, 1.3)) + F(dot(14.5, 13, 1.3)));

/* ================================================================ SEAL
 * Draw-on outline: a stroke-only reconstruction of identity/seal.svg's ring
 * and cross-weave (that file is fill-based, which stroke-dashoffset cannot
 * animate), used only for the sealing MOMENT. It draws on once, then the
 * real filled seal.svg crossfades in — see SealAnimation.svelte.
 */

{
  const teeth = [];
  for (let i = 0; i < 16; i++) {
    const a = (Math.PI * 2 * i) / 16;
    teeth.push(line(100 + 88 * Math.cos(a), 100 + 88 * Math.sin(a), 100 + 98 * Math.cos(a), 100 + 98 * Math.sin(a)));
  }
  const body =
    '<g fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round">\n' +
    '    <path stroke-width="3" d="' + circ(100, 100, 92) + '"/>\n' +
    '    <path stroke-width="2" d="' + teeth.join(' ') + '"/>\n' +
    '    <path stroke-width="6" d="' + line(33.5, 100, 166.5, 100) + '"/>\n' +
    '    <path stroke-width="6" d="' + line(100, 33.5, 100, 84) + '"/>\n' +
    '    <path stroke-width="6" d="' + line(100, 116, 100, 166.5) + '"/>\n' +
    '  </g>';
  const svg =
    '<?xml version="1.0" encoding="UTF-8"?>\n' +
    '<svg viewBox="0 0 200 200" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">\n' +
    '  <!-- Seal draw-on outline. Stroke-only reconstruction of identity/seal.svg\'s ring, radial ticks and warp/weft cross, for the one-time "sealing" animation (SealAnimation.svelte): each path\'s stroke-dashoffset animates from its own length to 0 over 400ms (the UI-feedback cap), then the component crossfades to the real filled seal.svg and this outline is removed from the DOM. Not intended for standalone display — it is a transient animation frame, not the seal icon. -->\n' +
    '  ' + body + '\n' +
    '</svg>\n';
  writeFileSync(join(SRC, 'seal-draw.svg'), svg);
  assets.push({ name: 'seal-draw', svg, w: 200, h: 200 });
}

/* --------------------------------------------------------------- emit */

for (const a of assets) writeFileSync(join(SRC, a.name + '.svg'), a.svg);

/* ============================================================== CHECKS */

let failed = 0;
const fail = (m) => { console.error('FAIL  ' + m); failed++; };
const ok = (m) => console.log('ok    ' + m);

for (const a of assets) {
  const geom = a.svg.replace(/<!--[\s\S]*?-->/g, '');
  if (!/viewBox=/.test(geom)) fail(a.name + '.svg: no viewBox');
  if (/<svg[^>]*\swidth=/.test(geom)) fail(a.name + '.svg: hardcoded width on root svg');
  if (/#[0-9a-fA-F]{3,6}\b/.test(geom.replace(/var\([^)]*\)/g, ''))) fail(a.name + '.svg: bare hex colour outside a var() fallback');
  if (/\bid=/.test(geom)) fail(a.name + '.svg: unreferenced id');
  if (/<animate|<set\b/.test(geom)) fail(a.name + '.svg: SMIL animation — CSS only, per Batch 7');
  const bytes = Buffer.byteLength(a.svg, 'utf8');
  if (bytes > 2048) fail(a.name + '.svg: ' + bytes + 'B over the 2KB budget');
}
if (!failed) ok(assets.length + ' motion assets: viewBox / no width-height / no bare hex / no id / no SMIL / size budget');

console.log('\n' + assets.length + ' assets -> packages/motion/src');
for (const a of assets) console.log('  ' + String(Buffer.byteLength(a.svg, 'utf8')).padStart(5) + 'B  ' + a.name);

/* --- motion.css: every animated property must be transform/opacity/
       stroke-dashoffset. width/height/top/left/margin trigger layout,
       which is this batch's actual acceptance criterion, and it lives in
       CSS, not in any SVG the loop above already checked. --- */
try {
  const css = readFileSync(join(ROOT, 'packages/motion/motion.css'), 'utf8');
  const allowed = new Set(['transform', 'opacity', 'stroke-dashoffset', 'stroke-dasharray']);

  // Brace-counting extraction, not a line-ending regex guess — a one-line
  // @keyframes rule (no internal newlines) broke the earlier regex-only
  // version by letting its non-greedy match run on into unrelated CSS.
  const blocks = [];
  const starts = [...css.matchAll(/@keyframes\s+[\w-]+\s*\{/g)];
  for (const m of starts) {
    let depth = 1, i = m.index + m[0].length;
    while (depth > 0 && i < css.length) {
      if (css[i] === '{') depth++;
      else if (css[i] === '}') depth--;
      i++;
    }
    blocks.push(css.slice(m.index, i));
  }

  let propCount = 0;
  for (const block of blocks) {
    const props = [...block.matchAll(/([a-zA-Z-]+)\s*:/g)].map((m) => m[1]);
    for (const p of props) {
      propCount++;
      if (!allowed.has(p)) fail('motion.css: @keyframes animates "' + p + '", not in the transform/opacity/stroke-dashoffset allowlist');
    }
  }
  if (!failed) ok('motion.css: ' + propCount + ' animated propert' + (propCount === 1 ? 'y' : 'ies') + ' across ' + blocks.length + ' @keyframes block(s), all in the transform/opacity/stroke-dashoffset allowlist');

  // transition: declarations outside @keyframes are the other place a
  // layout property could sneak in.
  const transitions = [...css.matchAll(/transition:\s*([^;]+);/g)].map((m) => m[1]);
  for (const t of transitions) {
    const prop = t.trim().split(/\s+/)[0];
    if (!['all', 'none', 'border-color', 'color', 'opacity'].includes(prop) && !allowed.has(prop))
      fail('motion.css: transition animates "' + prop + '" — check it does not trigger layout');
  }
} catch (e) {
  console.log('  (motion.css not written yet — run again after it exists to check its @keyframes)');
}

/* --- no fake timer driving the pipeline: grep the component source --- */
try {
  const raw = readFileSync(join(ROOT, 'packages/motion/PipelineProgress.svelte'), 'utf8');
  // strip the HTML comment block (the doc header explains the no-timer rule
  // in prose, which would otherwise trip its own check) before scanning
  // for actual code.
  const pipelineSrc = raw.replace(/<!--[\s\S]*?-->/g, '');
  if (/setTimeout|setInterval/.test(pipelineSrc)) fail('PipelineProgress.svelte: contains setTimeout/setInterval — stage must come from real backend state, never a timer');
  else ok('PipelineProgress.svelte: no setTimeout/setInterval — stages are prop-driven');
} catch (e) {
  console.log('  (PipelineProgress.svelte not written yet)');
}

if (failed) { console.error('\n' + failed + ' check(s) failed'); process.exit(1); }
console.log('\nall checks passed');
