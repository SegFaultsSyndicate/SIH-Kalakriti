/*
 * Kalakriti assets — Batch 5 generator
 * Illustrations and empty states. Warli-derived economy of line, geometric,
 * flat, two-colour maximum (currentColor + one accent CSS var per asset).
 *
 * Run:  node scripts/gen-batch5.mjs
 *
 * Composed the same way Batches 3-4 were: reusable primitives (a Warli
 * figure builder, a loom, a cloth fold, a basket weave) rather than
 * hand-typed path data per scene, so ~24 illustrations stay geometrically
 * and stylistically consistent.
 */

import { writeFileSync, readFileSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const SRC = join(ROOT, 'packages/illustrations/src');
mkdirSync(SRC, { recursive: true });

/* ---------------------------------------------------------------- utils */

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
const rrect = (x, y, w, h, r) =>
  'M' + pt(x + r, y) +
  ' L' + pt(x + w - r, y) + ' A' + n(r) + ' ' + n(r) + ' 0 0 1 ' + pt(x + w, y + r) +
  ' L' + pt(x + w, y + h - r) + ' A' + n(r) + ' ' + n(r) + ' 0 0 1 ' + pt(x + w - r, y + h) +
  ' L' + pt(x + r, y + h) + ' A' + n(r) + ' ' + n(r) + ' 0 0 1 ' + pt(x, y + h - r) +
  ' L' + pt(x, y + r) + ' A' + n(r) + ' ' + n(r) + ' 0 0 1 ' + pt(x + r, y) + ' Z';

// illustration stroke = canvas-width / 120, clamp min 1.5 (stroke-geometry-spec.md)
const illStroke = (canvasW) => Math.max(1.5, Math.round((canvasW / 120) * 100) / 100);

const S = (d, w) => ({ d, fill: 'none', stroke: 'main', w });
const A = (d, w) => ({ d, fill: 'none', stroke: 'accent', w }); // the one permitted accent colour
const F = (d) => ({ d, fill: 'main', stroke: false });
const FA = (d) => ({ d, fill: 'accent', stroke: false });

/*
 * Warli figure — circle head, two triangles apex-to-apex (torso/pelvis),
 * straight-line limbs. motifs.md: economy of line, no facial features, no
 * clothing detail, no colour fill beyond the two permitted tones.
 * cx,cy = point between the feet (the figure's "ground" anchor). s = scale
 * (roughly shoulder-to-hip height). pose selects an arm position.
 */
function warli(cx, cy, s, pose = 'rest', accent = false) {
  const stroke = accent ? A : S;
  const fill = accent ? FA : F;
  const headR = s * 0.22;
  const shoulderY = cy - s * 1.9;
  const headCy = shoulderY - headR * 1.8;
  const waistY = cy - s * 1.15;   // triangle apexes meet here
  const hipY = cy - s * 0.55;
  const footY = cy;
  const els = [
    fill(dot(cx, headCy, headR)),
    stroke(poly([[cx, shoulderY], [cx - s * 0.5, waistY], [cx + s * 0.5, waistY]], true)),
    stroke(poly([[cx, waistY], [cx - s * 0.42, hipY], [cx + s * 0.42, hipY]], true)),
    stroke(line(cx - s * 0.42, hipY, cx - s * 0.55, footY)),
    stroke(line(cx + s * 0.42, hipY, cx + s * 0.55, footY)),
  ];
  const armY = shoulderY + s * 0.15;
  if (pose === 'up') {
    els.push(stroke(line(cx, armY, cx - s * 0.75, armY - s * 0.65)));
    els.push(stroke(line(cx, armY, cx + s * 0.75, armY - s * 0.65)));
  } else if (pose === 'carry') {
    els.push(stroke(line(cx, armY, cx - s * 0.7, armY + s * 0.4) + ' L' + pt(cx - s * 0.9, armY + s * 0.35)));
    els.push(stroke(line(cx, armY, cx + s * 0.7, armY + s * 0.4) + ' L' + pt(cx + s * 0.9, armY + s * 0.35)));
  } else if (pose === 'weave') {
    els.push(stroke(line(cx, armY, cx - s * 0.65, armY + s * 0.1)));
    els.push(stroke(line(cx, armY, cx + s * 0.65, armY - s * 0.15)));
  } else if (pose === 'sit') {
    // overridden entirely by callers that need a seated pose (see loomScene)
    els.push(stroke(line(cx, armY, cx - s * 0.55, armY + s * 0.35)));
    els.push(stroke(line(cx, armY, cx + s * 0.55, armY + s * 0.35)));
  } else {
    els.push(stroke(line(cx, armY, cx - s * 0.4, armY + s * 0.55)));
    els.push(stroke(line(cx, armY, cx + s * 0.4, armY + s * 0.55)));
  }
  return els;
}

/* A loom: frame + strung warp threads, optional weft-in-progress lines. */
function loom(x, y, w, h, threads, opts = {}) {
  const els = [S(poly([[x, y], [x, y + h], [x + w, y + h], [x + w, y]]))];
  const step = w / (threads + 1);
  for (let i = 1; i <= threads; i++) {
    const tx = x + i * step;
    els.push(S(line(tx, y, tx, y + h - (opts.slack || 0))));
  }
  if (opts.weftRows) {
    for (const wy of opts.weftRows) els.push((opts.weftAccent ? A : S)(line(x + 2, wy, x + w - 2, wy)));
  }
  return els;
}

const icons = []; // { name, svg, group }

function buildScene(name, group, w, h, sw, elements, comment) {
  const body = elements.map((el) => {
    const colour = el.fill === 'main' ? 'currentColor'
      : el.fill === 'accent' ? 'var(--k-illustration-accent, #B5462B)'
      : 'none';
    const strokeColour = el.stroke === 'main' ? 'currentColor'
      : el.stroke === 'accent' ? 'var(--k-illustration-accent, #B5462B)'
      : undefined;
    if (strokeColour) {
      return '<path fill="none" stroke="' + strokeColour + '" stroke-width="' + (el.w || sw) +
        '" stroke-linecap="round" stroke-linejoin="round" d="' + el.d + '"/>';
    }
    return '<path fill="' + colour + '" d="' + el.d + '"/>';
  }).join('\n  ');
  const svg =
    '<?xml version="1.0" encoding="UTF-8"?>\n' +
    '<svg viewBox="0 0 ' + w + ' ' + h + '" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">\n' +
    '  <!-- ' + comment + ' -->\n' +
    '  ' + body + '\n' +
    '</svg>\n';
  icons.push({ name, svg, group, w, h });
  return svg;
}

/* ======================================================== EMPTY STATES
 * 320x240, two colours max (currentColor + one accent). Legible at 200px
 * wide is the acceptance criterion — kept deliberately spare.
 */

const EW = 320, EH = 240, es = illStroke(EW);

buildScene('empty-no-listings', 'empty', EW, EH, es,
  loom(70, 55, 180, 130, 7, { slack: 4 }),
  'Handloom structure. Empty loom, warp strung and waiting, no weft yet — "no listings yet".');

buildScene('empty-no-orders', 'empty', EW, EH, es, [
  ...[0, 1, 2].map((i) => S(rrect(90 - i * 4, 150 - i * 14, 140 + i * 8, 14, 2))),
  S(rrect(190, 70, 60, 76, 2)),
  S(line(200, 86, 240, 86)), S(line(200, 100, 240, 100)), S(line(200, 114, 230, 114)),
], 'Handloom structure (folded cloth) beside an open ledger — "no orders yet".');

buildScene('empty-no-search-results', 'empty', EW, EH, es, (() => {
  const els = [S(rrect(90, 40, 140, 160, 4))];
  const cx = 160, cy = 120, cols = 4, rows = 5, stepX = 130 / cols, stepY = 150 / rows;
  for (let c = 1; c < cols; c++) els.push(S(line(90 + c * stepX, 45, 90 + c * stepX, 195)));
  for (let r = 1; r < rows; r++) els.push(S(line(95, 40 + r * stepY, 225, 40 + r * stepY)));
  return els;
})(), 'Jaali. A pierced-lattice window looking onto empty space — "no search results".');

buildScene('empty-no-notifications', 'empty', EW, EH, es, (() => {
  const cx = 160, cy = 120, r = 55;
  const els = [S(circ(cx, cy, r))];
  for (let a = 0; a < 360; a += 45) {
    els.push(S(line(cx + 22 * Math.cos(rad(a)), cy + 22 * Math.sin(rad(a)), cx + r * Math.cos(rad(a)), cy + r * Math.sin(rad(a)))));
  }
  els.push(S(circ(cx, cy, 22)));
  els.push(A(circ(cx + 46, cy + 34, 15)));
  return els;
})(), 'Charkha, used sparingly and abstractly per motifs.md — a still spinning wheel, no motion lines — "no notifications".');

buildScene('empty-offline', 'empty', EW, EH, es, [
  S(line(45, 120, 150, 120)),
  S(dot(45, 120, 4)), S(dot(150, 120, 4)),
  A('M185 112 Q195 120 185 128'),
  A(dot(185, 120, 3.5)),
], 'Handloom structure. A thread crossing a gap, one end unattached and curling loose (accent) — "offline".');

buildScene('empty-error', 'empty', EW, EH, es, [
  S(poly([[120, 60], [180, 60], [170, 100], [130, 100]], true)),
  S(line(150, 100, 150, 140)),
  A(line(150, 140, 130, 155) + ' L170 165 L' + pt(125, 180) + ' L185 190'),
  S(dot(150, 140, 3)),
], 'Handloom structure. A dropped shuttle (the trapezoid) with tangled thread (accent) trailing from it — "error".');

buildScene('empty-no-followers', 'empty', EW, EH, es, [
  ...loom(150, 60, 130, 110, 6, { slack: 20 }),
  ...warli(90, 190, 34, 'weave'),
], 'Warli figure-abstraction (economy of line, per motifs.md) at a loom — "no followers yet".');

buildScene('empty-cart', 'empty', EW, EH, es, (() => {
  const els = [];
  const bx = 90, by = 110, bw = 140, bh = 80;
  els.push(S(poly([[bx, by], [bx + bw, by], [bx + bw - 14, by + bh], [bx + 14, by + bh]], true)));
  for (let i = 1; i < 6; i++) {
    const x0 = bx + (bw / 6) * i, x1 = bx + (bw - 28) / 6 * i + 14;
    els.push(S(line(x0, by, x1, by + bh)));
  }
  for (let j = 1; j < 4; j++) els.push(S(line(bx + 6, by + (bh / 4) * j, bx + bw - 6, by + (bh / 4) * j)));
  els.push(S('M' + pt(bx + 20, by) + ' Q' + pt(160, by - 55) + ' ' + pt(bx + bw - 20, by)));
  return els;
})(), 'Basketry (interlaced cane). An empty market basket — "empty cart".');

/* ========================================================== ONBOARDING
 * 3-4 scenes, wordless, for the artisan welcome flow. Same 320x240 canvas
 * and two-colour rule as the empty states, for a single consistent set.
 */

buildScene('onboard-photograph', 'onboard', EW, EH, es, [
  S(rrect(55, 90, 100, 70, 3)),
  S(line(65, 100, 65, 150) + ' M75 100 L75 150 M85 100 L85 150 M95 100 L95 150'),
  S(line(140, 130, 165, 110) + ' L' + pt(150, 100)),   // near hand/wrist to phone
  S(line(140, 130, 175, 145) + ' L' + pt(160, 155)),   // far hand/wrist to phone
  S(rrect(150, 85, 55, 80, 6)),
  S(circ(177, 148, 11)), S(circ(177, 148, 5)),
  A(dot(177, 100, 3)),
], 'Handloom structure (folded cloth) with two hands and arms — reduced Warli line, no torso — holding a phone up to it — "photograph your work".');

buildScene('onboard-description', 'onboard', EW, EH, es, [
  S(rrect(50, 60, 90, 120, 3)),
  ...Array.from({length: 6}, (_, i) => S(line(60, 80 + i * 15, 130, 80 + i * 15))),
  A('M150 120 Q180 90 210 100 Q235 108 250 90'),
  S(rrect(190, 60, 90, 110, 3)),
  ...Array.from({length: 5}, (_, i) => S(line(200, 78 + i * 18, 270, 78 + i * 18))),
], 'Handloom structure. Warp threads (left) becoming text lines (right), joined by a drawn thread — "we write the description".');

buildScene('onboard-buyers-find-you', 'onboard', EW, EH, es, (() => {
  const nodes = [[160, 60], [90, 110], [230, 100], [70, 180], [160, 190], [250, 175], [200, 140]];
  const els = [S(poly([[30, 210], [60, 40], [290, 30], [270, 215]]))]; // abstract map outline
  const edges = [[0,1],[0,2],[1,3],[2,5],[0,6],[6,4],[4,3],[6,5]];
  for (const [a,b] of edges) els.push(S(line(nodes[a][0], nodes[a][1], nodes[b][0], nodes[b][1])));
  for (const [x,y] of nodes) els.push(F(dot(x, y, 4)));
  els.push(FA(dot(160, 60, 5.5)));
  return els;
})(), 'Kolam-derived node network across an abstract outline of India — "buyers find you across India".');

buildScene('onboard-paid-directly', 'onboard', EW, EH, es, [
  ...warli(70, 195, 34, 'rest'),
  ...warli(250, 195, 34, 'rest'),
  A(line(95, 130, 225, 130)),
  A(poly([[210, 122], [225, 130], [210, 138]])),
  FA(dot(160, 130, 4)),
], 'Warli figure-abstraction. A single direct thread from buyer to artisan, no intermediary node — "you are paid directly".');

/* ====================================================== FEATURE SCENES
 * Buyer-marketplace features. Same canvas and colour rule.
 */

buildScene('feature-collective-fulfilment', 'feature', EW, EH, es, (() => {
  const els = [];
  const looms = [[40, 60], [40, 130], [40, 195]];
  for (const [x,y] of looms) els.push(...loom(x, y - 40, 60, 55, 3, {slack: 6}));
  els.push(A('M100 40 Q170 20 170 90'));
  els.push(A('M100 110 L170 90'));
  els.push(A('M100 180 Q170 200 170 90'));
  els.push(S(rrect(170, 70, 90, 60, 3)));
  for (let i=1;i<5;i++) els.push(S(line(170 + i*15, 70, 170 + i*15, 130)));
  return els;
})(), 'Handloom structure. Many looms, one cloth emerging from their combined threads — "collective fulfilment".');

buildScene('feature-provenance', 'feature', EW, EH, es, [
  S(circ(230, 80, 34)), S(circ(230, 80, 22)),
  S(poly([[218,80],[226,88],[242,70]])),
  A('M200 100 Q150 130 110 150 Q80 165 60 190'),
  ...loom(35, 165, 55, 45, 3, {slack: 5}),
], 'The provenance seal with a thread trail running back to the originating loom — "provenance".');

buildScene('feature-handloom-verification', 'feature', EW, EH, es, (() => {
  const els = [];
  // left: irregular hand-woven grid
  const lx = 45, ly = 60, lw = 100, lh = 110;
  els.push(S(rrect(lx, ly, lw, lh, 2)));
  const cols = 7;
  for (let c = 1; c < cols; c++) {
    const wob = (c % 2 ? 3 : -3);
    els.push(S(line(lx + (lw/cols)*c + wob, ly + 4, lx + (lw/cols)*c - wob, ly + lh - 4)));
  }
  // right: regular machine grid
  const rx = 185, ry = 60, rw = 100, rh = 110;
  els.push(S(rrect(rx, ry, rw, rh, 2)));
  for (let c = 1; c < cols; c++) els.push(S(line(rx + (rw/cols)*c, ry + 4, rx + (rw/cols)*c, ry + rh - 4)));
  els.push(A(line(150, 115, 178, 115)));
  els.push(A(poly([[172,109],[182,115],[172,121]])));
  return els;
})(), 'Handloom structure. Irregular hand-woven grid (left) versus a regular machine grid (right) — the handloom-detector explainer, also feature copy — "handloom verification".');

buildScene('feature-fair-pricing', 'feature', EW, EH, es, [
  S(line(160, 60, 160, 90)),
  S(line(90, 90, 230, 90)),
  S(line(90, 90, 65, 130) + ' A26 22 0 0 0 115 130 L90 90'),
  S(line(230, 90, 205, 130) + ' A26 22 0 0 0 255 130 L230 90'),
  S(rrect(150, 160, 20, 20, 2)),
  A(line(55, 105, 100, 118)), A(dot(55,105,3)), A(dot(100,118,3)),
  S(rrect(200, 100, 24, 18, 2)), S(line(205, 108, 219, 108)),
], 'A balance scale weighing material, hours (accent thread) and skill (a small block) on one side — "fair pricing".');

/* ================================================== CRAFT PROCESS STEPS
 * Three sequences, four numbered steps each, 120x120, single colour.
 */

const PW = 120, ps = illStroke(PW);

// Steps are marked by dot-count (1-4 filled dots in the corner circle)
// rather than a numeral: rendering a digit legibly needs the glyph
// converted to a path (as Batch 2's wordmark did for "Kalakriti"), which
// is disproportionate for 12 small step icons, and a dot count reads
// language-independently — a real advantage for a low-literacy-friendly
// product per Batch 0's own accessibility framing.
function stepScene(name, seq, index, label, elements) {
  const dots = Array.from({ length: index + 1 }, (_, i) =>
    F(dot(11 + i * 6.5, 108, 2)));
  const els = [...elements, S(rrect(2, 98, 12 + index * 6.5, 20, 4)), ...dots];
  return buildScene(name, 'process', PW, PW, ps, els,
    'Process step ' + (index + 1) + ' of 4 (' + seq + '): ' + label + '. Step number shown as a dot count, not a numeral.');
}

// block printing: carve, dye, stamp, wash
stepScene('process-blockprint-1-carve', 'block printing', 0, 'carve', [S(rrect(35, 30, 55, 45, 3)), S(line(50, 45, 65, 45) + ' M50 60 L75 60'), S(line(80, 30, 95, 45))]);
stepScene('process-blockprint-2-dye', 'block printing', 1, 'dye', [S(rrect(30, 70, 60, 30, 2)), S('M60 30 C50 45 50 55 60 65 C70 55 70 45 60 30 Z')]);
stepScene('process-blockprint-3-stamp', 'block printing', 2, 'stamp', [S(rrect(40, 30, 40, 30, 2)), S(line(60, 60, 60, 75)), S(rrect(30, 78, 60, 22, 1)), S(dot(60, 89, 3))]);
stepScene('process-blockprint-4-wash', 'block printing', 3, 'wash', [S('M40 40 Q60 20 80 40'), S(line(35, 55, 85, 55)), A(line(45, 65, 45, 75)), A(line(60, 65, 60, 78)), A(line(75, 65, 75, 72))]);

// weaving: warp, weft, beat, finish
stepScene('process-weaving-1-warp', 'weaving', 0, 'warp', [S(line(35, 30, 35, 90)), S(line(50, 30, 50, 90)), S(line(65, 30, 65, 90)), S(line(80, 30, 80, 90))]);
stepScene('process-weaving-2-weft', 'weaving', 1, 'weft', [S(line(35, 30, 35, 90) + ' M50 30 L50 90 M65 30 L65 90 M80 30 L80 90'), A(line(28, 55, 87, 55))]);
stepScene('process-weaving-3-beat', 'weaving', 2, 'beat', [S(line(35, 30, 35, 90) + ' M50 30 L50 90 M65 30 L65 90 M80 30 L80 90'), S(line(28, 50, 87, 50)), A(line(30, 68, 90, 62))]);
stepScene('process-weaving-4-finish', 'weaving', 3, 'finish', [S(rrect(32, 35, 56, 45, 2)), S(line(32, 50, 88, 50) + ' M32 65 L88 65')]);

// pottery: centre, throw, dry, fire
stepScene('process-pottery-1-centre', 'pottery', 0, 'centre', [S(circ(60, 65, 22)), S(dot(60, 65, 4))]);
stepScene('process-pottery-2-throw', 'pottery', 1, 'throw', [S('M42 80 C40 55 45 35 60 32 C75 35 80 55 78 80')]);
stepScene('process-pottery-3-dry', 'pottery', 2, 'dry', [S('M42 82 C40 55 45 38 60 35 C75 38 80 55 78 82'), A(line(30,40,36,46)), A(line(84,40,90,46)), A(line(60,25,60,32))]);
stepScene('process-pottery-4-fire', 'pottery', 3, 'fire', [S('M40 82 L42 55 C44 40 76 40 78 55 L80 82 Z'), A('M55 82 C50 70 55 62 60 55 C65 62 70 70 65 82 Z')]);

/* ============================================================ HERO BACKDROP
 * A wide, very low-contrast jaali + weft composition to sit BEHIND a
 * photograph on the buyer home hero — a ground, not a scene, so it ships as
 * a small tileable pattern (matching Batch 3's approach for backgrounds)
 * rather than one fixed-width canvas: a 1440px-wide one-off would blow the
 * illustration size budget for no visual gain, since the pattern repeats
 * regardless. Tile via CSS background-repeat across the hero's full width;
 * the host composites it at 4-6% opacity so it stays behind the photo.
 */

{
  const HT = 60, hs = 1;
  const els = [];
  // diamond lattice (jaali octagonal-star cell logic, at hero density)
  for (const cx of [0, HT]) els.push(S(poly([[cx, 0], [cx + HT/2, HT/2], [cx, HT], [cx - HT/2, HT/2]], true), hs));
  // weft hairline, offset so it doesn't coincide with the lattice points
  els.push(S(line(0, HT * 0.75, HT, HT * 0.75), hs));
  const svg = buildScene('hero-backdrop-tile', 'hero', HT, HT, hs, els,
    'Jaali diamond lattice (octagonal-star cell logic) with a weft hairline, one 60x60 tile. Tile with background-repeat: repeat across the hero width at 4-6% opacity via CSS — see illustrations.css — so it sits behind a photograph without competing.');
}

/* ================================================= FFT COMPARISON GRAPHIC
 * Two fabric swatches (irregular handwoven vs. regular machine-made) each
 * paired with an annotated frequency-spectrum bar chart. This is the
 * handloom-detector explainer and a slide asset — the data shape (bar
 * heights) is a placeholder the ML track's real FFT output can replace
 * without changing the layout.
 */

{
  const FW = 480, FH = 300, fs = illStroke(FW);
  const els = [];
  // left: handwoven swatch (irregular) + its spectrum (broad, uneven)
  els.push(S(rrect(30, 30, 160, 90, 3)));
  for (let c = 1; c < 8; c++) {
    const wob = (c % 3 - 1) * 2.5;
    els.push(S(line(30 + (160/8)*c + wob, 34, 30 + (160/8)*c - wob, 116)));
  }
  const leftBars = [0.3, 0.55, 0.4, 0.7, 0.5, 0.65, 0.35, 0.6, 0.45, 0.3];
  leftBars.forEach((v, i) => els.push(S(line(35 + i * 17, 260, 35 + i * 17, 260 - v * 110))));
  els.push(S(line(30, 260, 200, 260)));

  // right: machine swatch (regular) + its spectrum (one sharp peak)
  els.push(S(rrect(290, 30, 160, 90, 3)));
  for (let c = 1; c < 8; c++) els.push(S(line(290 + (160/8)*c, 34, 290 + (160/8)*c, 116)));
  const rightBars = [0.08, 0.1, 0.12, 0.9, 0.95, 0.15, 0.1, 0.08, 0.06, 0.05];
  rightBars.forEach((v, i) => els.push(A(line(295 + i * 17, 260, 295 + i * 17, 260 - v * 110))));
  els.push(S(line(290, 260, 460, 260)));

  els.push(S(line(245, 30, 245, 260)));

  const svg = buildScene('fft-comparison', 'fft', FW, FH, fs, els,
    'Handloom structure. Two fabric swatches (irregular hand-woven left, regular machine right) each above its frequency-spectrum bars: a broad, uneven spread for handwoven versus one sharp accent peak for machine-made. Bar heights are placeholder data — coordinate with the ML track to drop in the real FFT output at the same 10-bar layout.');
}

/* --------------------------------------------------------------- emit */

for (const icon of icons) writeFileSync(join(SRC, icon.name + '.svg'), icon.svg);

/* ============================================================== CHECKS */

let failed = 0;
const fail = (m) => { console.error('FAIL  ' + m); failed++; };
const ok = (m) => console.log('ok    ' + m);

for (const icon of icons) {
  const geom = icon.svg.replace(/<!--[\s\S]*?-->/g, '');
  if (!new RegExp('viewBox="0 0 ' + icon.w + ' ' + icon.h + '"').test(geom))
    fail(icon.name + '.svg: viewBox does not match its declared ' + icon.w + 'x' + icon.h);
  if (/<svg[^>]*\swidth=/.test(geom)) fail(icon.name + '.svg: hardcoded width on root svg');
  // a hex code is only legitimate as the fallback inside var(--token, #hex) —
  // strip those before checking for a genuinely bare one, per Batch 0.
  if (/#[0-9a-fA-F]{3,6}\b/.test(geom.replace(/var\([^)]*\)/g, ''))) fail(icon.name + '.svg: bare hex colour');
  if (/\bid=/.test(geom)) fail(icon.name + '.svg: unreferenced id');
  const colours = new Set([...geom.matchAll(/(?:fill|stroke)="(currentColor|var\([^)]+\))"/g)].map((m) => m[1]));
  if (colours.size > 2) fail(icon.name + '.svg: ' + colours.size + ' colours used, two-colour maximum');
  const bytes = Buffer.byteLength(icon.svg, 'utf8');
  if (bytes > 6144) fail(icon.name + '.svg: ' + bytes + 'B over the 6KB illustration budget');
}
if (!failed) ok(icons.length + ' scenes: viewBox / no width-height / no bare hex / no id / <=2 colours / size budget');

console.log('\n' + icons.length + ' illustrations -> packages/illustrations/src');
for (const i of icons) console.log('  ' + String(Buffer.byteLength(i.svg, 'utf8')).padStart(5) + 'B  ' + i.name.padEnd(28) + i.w + 'x' + i.h);

if (failed) { console.error('\n' + failed + ' check(s) failed'); process.exit(1); }
console.log('\nall checks passed');

/* ------------------------------------------------------------- manifest */

writeFileSync(join(ROOT, 'packages/illustrations/manifest.js'),
  '/* Generated by scripts/gen-batch5.mjs — do not hand-edit. */\n\n' +
  'export const ILLUSTRATIONS = ' + JSON.stringify(
    icons.map((i) => ({ name: i.name, group: i.group, file: 'src/' + i.name + '.svg', w: i.w, h: i.h })),
    null, 2
  ) + ';\n');

/* ------------------------------------------------------- contact sheet */

const innerBody = (svg) => svg
  .replace(/[\s\S]*?<svg[^>]*>\n/, '')
  .replace(/<\/svg>\s*$/, '')
  .replace(/<!--[\s\S]*?-->\n?/g, '')
  .trim();

const tag = (icon, width) =>
  '<svg viewBox="0 0 ' + icon.w + ' ' + icon.h + '" style="width:' + width + 'px" aria-hidden="true" focusable="false">' +
  innerBody(icon.svg) + '</svg>';

const card = (icon, width) =>
  '<div class="ill-card"><div class="ill-card__frame">' + tag(icon, width) +
  '</div><div class="ill-card__name">' + icon.name + '</div></div>';

const group = (label, note, names, width) =>
  '<h2>' + label + '</h2>' + (note ? '<p class="note">' + note + '</p>' : '') +
  '<div class="ill-grid">' + names.map((n2) => card(icons.find((i) => i.name === n2), width)).join('') + '</div>';

const byG = (g) => icons.filter((i) => i.group === g).map((i) => i.name);

const processSeq = (seq, label) => {
  const names = byG('process').filter((n2) => n2.includes(seq));
  return '<div class="process-seq"><h3>' + label + '</h3><div class="k-process-strip">' +
    names.map((n2) => card(icons.find((i) => i.name === n2), 96)).join('') + '</div></div>';
};

const heroTile = icons.find((i) => i.name === 'hero-backdrop-tile');

const themeSection = (theme, label) => `
<section class="theme" data-theme="${theme}">
  <header><h1>Batch 5 — illustrations and empty states</h1><span>${label}</span></header>

  ${group('1 · Empty states — real size, 200px wide', 'Each must be legible and pleasant at 200px — the size shown here, not zoomed.', byG('empty'), 200)}

  ${group('2 · Onboarding — the artisan welcome flow, wordless', '', byG('onboard'), 200)}

  ${group('3 · Feature illustrations — buyer marketplace', '', byG('feature'), 200)}

  <h2>4 · FFT comparison graphic</h2>
  <p class="note">Handloom detector explainer and slide asset. Bar heights are placeholder data for the layout — the ML track drops in the real spectrum at the same 10-bar shape.</p>
  <div class="ill-card" style="width:480px"><div class="ill-card__frame">${tag(icons.find((i) => i.name === 'fft-comparison'), 480)}</div></div>

  <h2>5 · Craft process diagrams — four steps each</h2>
  <p class="note">Step number is a dot count in the corner, not a numeral — language-independent, and avoids converting a digit glyph to a path for twelve small icons.</p>
  ${processSeq('blockprint', 'Block printing — carve, dye, stamp, wash')}
  ${processSeq('weaving', 'Weaving — warp, weft, beat, finish')}
  ${processSeq('pottery', 'Pottery — centre, throw, dry, fire')}

  <h2>6 · Hero backdrop — tiled, mask over currentColor</h2>
  <p class="note">60×60 tile, shown here at a raised opacity so the seam is visible; ships at 4-6%. Uses the CSS mask technique from Batch 3's ornament.css, never background-image — see illustrations.css.</p>
  <div class="hero-demo" style="mask-image:url('data:image/svg+xml,${encodeURIComponent(heroTile.svg.replace(/currentColor/g, '#000'))}');-webkit-mask-image:url('data:image/svg+xml,${encodeURIComponent(heroTile.svg.replace(/currentColor/g, '#000'))}');mask-repeat:repeat;-webkit-mask-repeat:repeat;mask-size:60px 60px;-webkit-mask-size:60px 60px;"></div>
</section>`;

const sheet = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Kalakriti — Batch 5 contact sheet: illustrations</title>
<link rel="stylesheet" href="../tokens/src/palette.css">
<link rel="stylesheet" href="./illustrations.css">
<style>
  body { margin: 0; font: 14px/1.5 ui-sans-serif, system-ui, sans-serif; }
  .theme { background: var(--k-surface-base); color: var(--k-text-primary); padding: 2.5rem clamp(1rem,4vw,3rem) 3.5rem; }
  .theme > header { display: flex; align-items: baseline; gap: 0.75rem; margin: 0 0 2rem; padding-bottom: 0.6rem; border-bottom: 1px solid var(--k-border-hairline); }
  .theme > header h1 { font-size: 1.1rem; margin: 0; }
  .theme > header span { color: var(--k-text-secondary); font-size: 0.8rem; }
  h2 { font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.09em; color: var(--k-text-secondary); margin: 2.5rem 0 0.9rem; font-weight: 600; }
  h2:first-of-type { margin-top: 0; }
  h3 { font-size: 0.78rem; margin: 0 0 0.5rem; }
  .note { font-size: 0.75rem; color: var(--k-text-secondary); max-width: 62ch; margin: 0.3rem 0 1rem; }
  .ill-grid { display: flex; flex-wrap: wrap; gap: 1rem; }
  .ill-card { border: 1px solid var(--k-border-hairline); background: var(--k-surface-raised); padding: 0.75rem; text-align: center; }
  .ill-card__frame svg { display: block; color: var(--k-text-secondary); }
  .ill-card__name { margin-top: 0.5rem; font-size: 0.68rem; color: var(--k-text-secondary); font-family: ui-monospace, monospace; }
  .process-seq { margin-bottom: 1.5rem; }
  .hero-demo { width: 100%; height: 60px; background-color: currentColor; color: var(--k-text-primary); }
</style>
</head>
<body>
${themeSection('light', 'light theme')}
${themeSection('dark', 'dark theme')}
${themeSection('high-contrast', 'high-contrast theme')}
</body>
</html>
`;

writeFileSync(join(ROOT, 'packages/illustrations/contact-sheet-batch5.html'), sheet);
console.log('manifest.js and contact-sheet-batch5.html written');
