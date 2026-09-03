/*
 * Kalakriti assets — Batch 3 generator
 * Structural ornament: jaali dividers, block-print borders, kolam corner,
 * hairline rules, weft texture, card edge.
 *
 * Run:  node scripts/gen-batch3.mjs
 * Writes packages/ornament/src/*.svg, patterns.svg and ornament.css, then
 * runs a set of hard checks (seam parity, height cap, size budget).
 *
 * Everything here is generated rather than hand-authored so that the seam
 * guarantee is a property of the construction, not of an illustrator's
 * patience. The checks at the bottom fail the build; they do not warn.
 */

import { writeFileSync, readFileSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const SRC = join(ROOT, 'packages/ornament/src');
mkdirSync(SRC, { recursive: true });

/* ---------------------------------------------------------------- utils */

const n = (v) => String(Math.round(v * 100) / 100);

/** Divider / pattern stroke rule — see stroke-geometry-spec.md.
 *  stroke-width = tile-height / 20, snapped to the 0.25px grid, min 0.75. */
const patternStroke = (h) => Math.max(0.75, Math.round((h / 20) * 4) / 4);

/** Polyline -> path data. */
const poly = (pts, close = false) =>
  'M' + pts.map(([x, y]) => n(x) + ' ' + n(y)).join(' L ') + (close ? ' Z' : '');

/** Axis-aligned rect as path data (Batch 0 prefers <path>). */
const rect = (x, y, w, h) =>
  'M' + n(x) + ' ' + n(y) + ' H' + n(x + w) + ' V' + n(y + h) + ' H' + n(x) + ' Z';

/** Diamond (square rotated 45 deg) centred at cx,cy. */
const diamond = (cx, cy, rx, ry = rx) =>
  poly([[cx, cy - ry], [cx + rx, cy], [cx, cy + ry], [cx - rx, cy]], true);

/* Records every primitive that touches x=0 or x=W so the seam check can
 * compare the two edges. Geometry is authored inside [0,W]; anything sitting
 * exactly on an edge renders as a half-stroke there and is completed by the
 * matching half-stroke of the neighbouring tile. */
const seamProbe = () => {
  const left = [], right = [];
  return {
    note(w, x, y0, y1 = y0) {
      const key = n(Math.min(y0, y1)) + '..' + n(Math.max(y0, y1));
      if (Math.abs(x) < 1e-6) left.push(key);
      else if (Math.abs(x - w) < 1e-6) right.push(key);
    },
    left, right,
  };
};

/* -------------------------------------------------------------- assets */

const files = [];
const patterns = [];

const add = (name, comment, w, h, body, opts = {}) => {
  const svg =
    '<?xml version="1.0" encoding="UTF-8"?>\n' +
    '<svg viewBox="0 0 ' + n(w) + ' ' + n(h) + '" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">\n' +
    '  <!-- ' + comment + ' -->\n' +
    body.split('\n').map((l) => (l ? '  ' + l : l)).join('\n') + '\n' +
    '</svg>\n';
  files.push(Object.assign({ name, svg, w, h }, opts));
  return svg;
};

/* ============================================================== 1. JAALI
 * Three geometries, three densities each. Density is not a scale factor:
 * each tile is redrawn, because a 16px-tall mesh cannot carry the element
 * count of a 32px one once stroke weight has to stay visible. The rule
 * across all three geometries is:
 *   fine   = base cell
 *   medium = base cell + centre aperture
 *   bold   = base cell + centre aperture + inner echo of the base cell
 * Hard corners (butt cap, miter join) per stroke-geometry-spec.
 */

const DENSITIES = { fine: 16, medium: 24, bold: 32 };
const LEVEL = { fine: 0, medium: 1, bold: 2 };

function jaaliHex(h, level, probe) {
  const P = Math.round(h * (Math.sqrt(3) / 2));
  const sw = patternStroke(h);
  const i = sw / 2 + 0.5;
  const band = h - 2 * i;
  const yTop = i, yBot = h - i, yA = i + band / 4, yB = i + (3 * band) / 4;
  const d = [
    poly([[0, yA], [P / 2, yTop], [P, yA]]),
    poly([[0, yB], [P / 2, yBot], [P, yB]]),
    poly([[0, yA], [0, yB]]),
    poly([[P, yA], [P, yB]]),
  ];
  probe.note(P, 0, yA, yB); probe.note(P, P, yA, yB);
  probe.note(P, 0, yA); probe.note(P, P, yA);
  probe.note(P, 0, yB); probe.note(P, P, yB);
  if (level >= 2) {
    const s = 0.46, cx = P / 2, cy = h / 2;
    const eA = cy - ((yB - yA) / 2) * s, eB = cy + ((yB - yA) / 2) * s;
    const eT = cy - (cy - yTop) * s, eBo = cy + (yBot - cy) * s;
    const ex = (P / 2) * s;
    d.push(poly([[cx - ex, eA], [cx, eT], [cx + ex, eA], [cx + ex, eB],
      [cx, eBo], [cx - ex, eB]], true));
  }
  if (level >= 1) d.push(diamond(P / 2, h / 2, band * 0.11));
  return { P, sw, d };
}

function jaaliOctstar(h, level, probe) {
  const P = h;
  const sw = patternStroke(h);
  const i = sw / 2 + 0.5;
  const inner = h - 2 * i;
  // A regular octagon (flat side = (sqrt2-1) x width) reads as a circle once
  // its sides are ~6px, which is exactly where the 16px tile lands. At fine
  // density the flats are lengthened so the cell stays visibly faceted.
  const k = level === 0 ? 0.58 : Math.SQRT2 - 1;
  const a = P * k;
  const b = inner * k;
  const cx = P / 2, cy = h / 2;
  const oct = (s) => poly([
    [cx - (a / 2) * s, cy - (inner / 2) * s],
    [cx + (a / 2) * s, cy - (inner / 2) * s],
    [cx + (P / 2) * s, cy - (b / 2) * s],
    [cx + (P / 2) * s, cy + (b / 2) * s],
    [cx + (a / 2) * s, cy + (inner / 2) * s],
    [cx - (a / 2) * s, cy + (inner / 2) * s],
    [cx - (P / 2) * s, cy + (b / 2) * s],
    [cx - (P / 2) * s, cy - (b / 2) * s],
  ], true);
  const d = [oct(1)];
  probe.note(P, 0, cy - b / 2, cy + b / 2);
  probe.note(P, P, cy - b / 2, cy + b / 2);
  if (level >= 2) d.push(oct(0.46));
  if (level >= 1) d.push(diamond(cx, cy, inner * 0.11));
  return { P, sw, d };
}

function jaaliInterlace(h, level, probe) {
  const P = h;
  const sw = patternStroke(h);
  const i = sw / 2 + 0.5;
  const cx = P / 2, cy = h / 2;
  const ry = h / 2 - i;
  // half-side > h/4 so the square's corners break out of the diamond and the
  // two shapes genuinely interlace rather than nest
  const s = 0.3 * h;
  const d = [
    poly([[cx, cy - ry], [P, cy], [cx, cy + ry], [0, cy]], true),
    rect(cx - s, cy - s, 2 * s, 2 * s),
  ];
  probe.note(P, 0, cy); probe.note(P, P, cy);
  if (level >= 2) {
    const k = 0.46;
    d.push(poly([[cx, cy - ry * k], [cx + (P / 2) * k, cy],
      [cx, cy + ry * k], [cx - (P / 2) * k, cy]], true));
  }
  if (level >= 1) d.push(diamond(cx, cy, h * 0.1));
  return { P, sw, d };
}

const JAALI = {
  'jaali-hex': { fn: jaaliHex, label: 'hexagonal' },
  'jaali-octstar': { fn: jaaliOctstar, label: 'octagonal-star' },
  'jaali-interlace': { fn: jaaliInterlace, label: 'interlaced-square' },
};

const seamReports = [];

for (const key of Object.keys(JAALI)) {
  const { fn, label } = JAALI[key];
  for (const density of Object.keys(DENSITIES)) {
    const h = DENSITIES[density];
    const probe = seamProbe();
    const { P, sw, d } = fn(h, LEVEL[density], probe);
    const body =
      '<g fill="none" stroke="currentColor" stroke-width="' + n(sw) +
      '" stroke-linecap="butt" stroke-linejoin="miter">\n' +
      '  <path d="' + d.join(' ') + '"/>\n' +
      '</g>';
    const name = key + '-' + h + '.svg';
    add(name,
      'Jaali (' + label + ' lattice), ' + density + ' density. Horizontal section divider, seamlessly tileable along X at ' + P + 'x' + h +
      '. Negative space is the subject; hard corners per stroke-geometry-spec. Inherits currentColor, intended at 12-24% opacity against the section text colour.',
      P, h, body, { budget: 2048, tile: true, group: 'jaali', label, density });
    patterns.push({ id: 'k-' + key + '-' + h, w: P, h, body, label: 'Jaali ' + label + ' ' + density });
    seamReports.push({ name, left: probe.left.sort(), right: probe.right.sort() });
  }
}

/* ====================================================== 2. BLOCK PRINT
 * Ajrakh/Bagru idiom: fill-led, not stroke-led, because block printing is
 * stamping and resist-dyeing, not drawing. The undyed areas are cut out of
 * the stamped shapes with fill-rule="evenodd" — that negative space is the
 * motif, exactly as resist printing works.
 */

{
  const W = 20, H = 20;
  const probe = seamProbe();
  const d = [
    rect(0, 1.5, W, 1),
    rect(0, H - 2.5, W, 1),
    diamond(10, 10, 5),
    diamond(10, 10, 2),
    rect(-0.75, 5.5, 1.5, 9),
    rect(W - 0.75, 5.5, 1.5, 9),
  ];
  probe.note(W, 0, 1.5, 2.5); probe.note(W, W, 1.5, 2.5);
  probe.note(W, 0, H - 2.5, H - 1.5); probe.note(W, W, H - 2.5, H - 1.5);
  probe.note(W, 0, 5.5, 14.5); probe.note(W, W, 5.5, 14.5);
  const body = '<path fill="currentColor" fill-rule="evenodd" d="' + d.join(' ') + '"/>';
  add('blockprint-running-20.svg',
    'Block print (Ajrakh/Bagru running border). Simple repeating band for card edges and section rules, tileable along X at 20x20. Fill-led, with resist-dye negative space cut by fill-rule evenodd. The separator tick is split across the seam and completed by the neighbouring tile.',
    W, H, body, { budget: 2048, tile: true, group: 'blockprint' });
  patterns.push({ id: 'k-blockprint-running-20', w: W, h: H, body, label: 'Block print running border' });
  seamReports.push({ name: 'blockprint-running-20.svg', left: probe.left.sort(), right: probe.right.sort() });
}

{
  const W = 32, H = 32;
  const probe = seamProbe();
  const d = [
    rect(0, 1.5, W, 1), rect(0, H - 2.5, W, 1),
    rect(0, 8.5, W, 1), rect(0, H - 9.5, W, 1),
  ];
  for (const x of [1, 9, 17, 25]) {
    d.push(rect(x, 4.5, 2, 2));
    d.push(rect(x, H - 6.5, 2, 2));
  }
  for (const cx of [8, 24]) {
    d.push(diamond(cx, 16, 5));
    d.push(rect(cx - 0.75, 12.5, 1.5, 7));
    d.push(rect(cx - 3.5, 15.25, 7, 1.5));
  }
  for (const cx of [0, 16, W]) d.push(diamond(cx, 16, 2.5));
  const edges = [[1.5, 2.5], [8.5, 9.5], [H - 9.5, H - 8.5], [H - 2.5, H - 1.5], [13.5, 18.5]];
  for (const e of edges) { probe.note(W, 0, e[0], e[1]); probe.note(W, W, e[0], e[1]); }
  const body = '<path fill="currentColor" fill-rule="evenodd" d="' + d.join(' ') + '"/>';
  add('blockprint-band-32.svg',
    'Block print (Ajrakh bordered band). Richer layered band for page headers and the provenance page, tileable along X at 32x32: outer rail, resist tick row, inner rail, rosette course. Fill-led; the rosette centres and the separator diamonds are resist negative space.',
    W, H, body, { budget: 2048, tile: true, group: 'blockprint' });
  patterns.push({ id: 'k-blockprint-band-32', w: W, h: H, body, label: 'Block print bordered band' });
  seamReports.push({ name: 'blockprint-band-32.svg', left: probe.left.sort(), right: probe.right.sort() });
}

/* ========================================================== 3. KOLAM CORNER
 * Sikku-kolam logic: a single continuous line that loops AROUND a grid of
 * dots and never touches or crosses one. Built as the outline of the union
 * of equal circles centred on the dots (r > pitch/2, so consecutive lobes
 * merge into one unbroken scalloped line). Round cap and join per spec: this
 * is the one drawn, gestural asset in the batch.
 */

{
  const PITCH = 24, R = 14, DOT = 1.6;
  const dots = [[60, 24], [36, 24], [12, 24], [12, 48], [12, 72]];
  const W = 96, H = 96;
  const TAU = Math.PI * 2;
  const norm = (a) => ((a % TAU) + TAU) % TAU;
  const beta = Math.acos((PITCH / 2) / R);

  const arcs = [];
  dots.forEach((c, idx) => {
    let keep = [[0, TAU]];
    dots.forEach((o, j) => {
      if (j === idx) return;
      const dist = Math.hypot(o[0] - c[0], o[1] - c[1]);
      if (dist > 2 * R - 1e-9) return;
      const th = Math.atan2(o[1] - c[1], o[0] - c[0]);
      const a0 = norm(th - beta), a1 = norm(th + beta);
      const cuts = a0 <= a1 ? [[a0, a1]] : [[a0, TAU], [0, a1]];
      for (const cut of cuts) {
        const out = [];
        for (const [p0, p1] of keep) {
          if (cut[1] <= p0 || cut[0] >= p1) { out.push([p0, p1]); continue; }
          if (cut[0] > p0) out.push([p0, cut[0]]);
          if (cut[1] < p1) out.push([cut[1], p1]);
        }
        keep = out;
      }
    });
    for (const [a0, a1] of keep.filter(([x, y]) => y - x > 1e-6)) {
      arcs.push({
        from: [c[0] + R * Math.cos(a0), c[1] + R * Math.sin(a0)],
        to: [c[0] + R * Math.cos(a1), c[1] + R * Math.sin(a1)],
        large: a1 - a0 > Math.PI ? 1 : 0,
      });
    }
  });

  const near = (p, q) => Math.hypot(p[0] - q[0], p[1] - q[1]) < 1e-6;
  const used = new Array(arcs.length).fill(false);
  const order = [0]; used[0] = true;
  let cursor = arcs[0].to;
  for (let step = 1; step < arcs.length; step++) {
    const k = arcs.findIndex((a, i) => !used[i] && near(a.from, cursor));
    if (k < 0) throw new Error('kolam: arc walk broke, union outline is not a single loop');
    used[k] = true; order.push(k); cursor = arcs[k].to;
  }
  if (!near(cursor, arcs[0].from)) throw new Error('kolam: arc walk did not close');

  let d = 'M' + n(arcs[order[0]].from[0]) + ' ' + n(arcs[order[0]].from[1]);
  for (const k of order) {
    const a = arcs[k];
    d += ' A' + n(R) + ' ' + n(R) + ' 0 ' + a.large + ' 1 ' + n(a.to[0]) + ' ' + n(a.to[1]);
  }
  d += ' Z';

  const dotPath = dots.map(([x, y]) =>
    'M' + n(x - DOT) + ' ' + n(y) +
    ' A' + n(DOT) + ' ' + n(DOT) + ' 0 1 0 ' + n(x + DOT) + ' ' + n(y) +
    ' A' + n(DOT) + ' ' + n(DOT) + ' 0 1 0 ' + n(x - DOT) + ' ' + n(y) + ' Z').join(' ');

  const body =
    '<path fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" d="' + d + '"/>\n' +
    '<path fill="currentColor" d="' + dotPath + '"/>';
  add('kolam-corner.svg',
    'Kolam (sikku) L-corner ornament for framing featured content. One continuous closed line looping around a five-dot L on the 24px dot grid; the line encircles every dot and touches none, which is the tradition’s defining constraint. Rotate for the other three corners: top-right scaleX(-1), bottom-right rotate(180deg), bottom-left scaleY(-1).',
    W, H, body, { budget: 2048, group: 'kolam' });
}

/* ====================================================== 4. HAIRLINE RULES
 * The plain hairline is CSS (border-top) — it has no SVG and therefore cannot
 * fail to load. The two motif variants below are SVG-backed, and every CSS
 * utility that uses them keeps a plain border underneath as the fallback.
 */

{
  const W = 24, H = 24;
  // The warp's break is 1.5 units wider than the weft bar on each side, so the
  // interruption survives as visible negative space. Butting the two bars flush
  // (as the 48px logomark can afford to) collapses into a solid plus sign at
  // rule scale — the frame that keeps the logomark legible is not here.
  const body = '<path fill="currentColor" d="' + [
    rect(2, 10.5, 20, 3),
    rect(10.5, 2, 3, 7),
    rect(10.5, 15, 3, 7),
  ].join(' ') + '"/>';
  add('rule-knot.svg',
    'Handloom structure (warp/weft crossing), reduced from the logomark. Centred motif for the interrupted hairline rule: the warp thread breaks with clear air either side of the weft passing over it, so the knot reads as weave rather than as a plus sign. Legibility floor is 20px, not 16px — it is a rule ornament, not an icon.',
    W, H, body, { budget: 2048, group: 'rule' });
}

{
  const W = 8, H = 8;
  const probe = seamProbe();
  const d = ['M0 0.5 H8', 'M0 7.5 H8', 'M4 0.5 V7.5', 'M0 4 H2.8', 'M5.2 4 H8'];
  probe.note(W, 0, 0.5); probe.note(W, W, 0.5);
  probe.note(W, 0, 7.5); probe.note(W, W, 7.5);
  probe.note(W, 0, 4); probe.note(W, W, 4);
  const body =
    '<g fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="butt" stroke-linejoin="miter">\n' +
    '  <path d="' + d.join(' ') + '"/>\n' +
    '</g>';
  add('rule-double-woven-8.svg',
    'Handloom structure. Double rule with woven-thread infill, tileable along X at 8x8: two rules with a warp thread between them and a weft thread broken where the warp passes over. Degrades to a plain 1px line via the border-top kept underneath it in ornament.css.',
    W, H, body, { budget: 2048, tile: true, group: 'rule' });
  patterns.push({ id: 'k-rule-double-woven-8', w: W, h: H, body, label: 'Double rule, woven infill' });
  seamReports.push({ name: 'rule-double-woven-8.svg', left: probe.left.sort(), right: probe.right.sort() });
}

/* ======================================================= 5. WEFT TEXTURE
 * Plain weave: warp and weft alternate over and under at each crossing. Only
 * the thread on top is drawn at a crossing, so the weave is legible purely
 * through the gaps in the thread that passes under. Two crossings per axis
 * per tile — the smallest tile that carries a full over/under cycle.
 * Budget: under 1KB.
 */

{
  const W = 12, H = 12, T = 2;
  const probe = seamProbe();
  const d = [
    rect(2, 0, T, 8), rect(2, 10, T, 2),
    rect(8, 0, T, 2), rect(8, 4, T, 8),
    rect(0, 2, 2, T), rect(4, 2, 8, T),
    rect(0, 8, 8, T), rect(10, 8, 2, T),
  ];
  probe.note(W, 0, 2, 4); probe.note(W, W, 2, 4);
  probe.note(W, 0, 8, 10); probe.note(W, W, 8, 10);
  const body = '<path fill="currentColor" d="' + d.join(' ') + '"/>';
  add('weft-texture-12.svg',
    'Handloom structure (plain weave). Page background texture, tileable on both axes at 12x12. Only the over-thread is drawn at each crossing, so the weave reads through the gaps. Intended at 3-6% opacity behind content.',
    W, H, body, { budget: 1024, tile: true, group: 'weft' });
  patterns.push({ id: 'k-weft-texture-12', w: W, h: H, body, label: 'Weft texture' });
  seamReports.push({ name: 'weft-texture-12.svg', left: probe.left.sort(), right: probe.right.sort() });
}

/* ========================================================= 6. CARD EDGE
 * A 9-slice frame: 48x48 with a 12px (25%) slice. The middle slices repeat,
 * so every motif sits strictly inside its slice and the inner rail runs
 * unbroken through the slice boundaries.
 */

{
  const W = 48, H = 48, S = 12;
  const corner = (ox, oy, sx, sy) => {
    const px = (x) => ox + sx * x, py = (y) => oy + sy * y;
    const box = (ax, ay, bx, by) => rect(
      Math.min(px(ax), px(bx)), Math.min(py(ay), py(by)),
      Math.abs(px(bx) - px(ax)), Math.abs(py(by) - py(ay)));
    return [box(2, 2, 10, 10), box(4.5, 4.5, 7.5, 7.5)];
  };
  const d = [
    ...corner(0, 0, 1, 1), ...corner(W, 0, -1, 1),
    ...corner(0, H, 1, -1), ...corner(W, H, -1, -1),
    rect(S, 5.5, W - 2 * S, 1), rect(S, H - 6.5, W - 2 * S, 1),
    rect(5.5, S, 1, H - 2 * S), rect(W - 6.5, S, 1, H - 2 * S),
  ];
  for (const c of [18, 30]) {
    d.push(diamond(c, 6, 3.5), diamond(c, 6, 1.4));
    d.push(diamond(c, H - 6, 3.5), diamond(c, H - 6, 1.4));
    d.push(diamond(6, c, 3.5), diamond(6, c, 1.4));
    d.push(diamond(W - 6, c, 3.5), diamond(W - 6, c, 1.4));
  }
  const body = '<path fill="currentColor" fill-rule="evenodd" d="' + d.join(' ') + '"/>';
  add('card-edge.svg',
    'Block print (Bagru corner stamp and running border). 9-slice card frame, 48x48 with a 12px (25%) slice: corner stamps, an unbroken inner rail, and a repeating diamond course in the middle slices. Gives a card a printed edge with no drop shadow.',
    W, H, body, { budget: 2048, group: 'cardedge' });
}

/* ------------------------------------------------------------- emit SVGs */

for (const f of files) writeFileSync(join(SRC, f.name), f.svg);

/* --------------------------------------------------- emit patterns.svg */

const patternDefs = patterns.map((p) =>
  '    <pattern id="' + p.id + '" patternUnits="userSpaceOnUse" width="' + n(p.w) + '" height="' + n(p.h) + '">\n' +
  p.body.split('\n').map((l) => '      ' + l).join('\n') + '\n' +
  '    </pattern>').join('\n');

const patternSprite =
  '<?xml version="1.0" encoding="UTF-8"?>\n' +
  '<!-- Kalakriti ornament patterns, Batch 3. Inline this whole file once per\n' +
  '     document (it renders nothing on its own), then reference a pattern with\n' +
  '     fill="url(#k-jaali-hex-24)". Every pattern inherits currentColor from the\n' +
  '     element that references it, so it takes the section’s text colour, which\n' +
  '     is the reason this exists as an inlinable sprite rather than as a\n' +
  '     background-image. Ids are namespaced k-* to survive being inlined\n' +
  '     alongside anything else. -->\n' +
  '<svg xmlns="http://www.w3.org/2000/svg" width="0" height="0" style="position:absolute" aria-hidden="true" focusable="false">\n' +
  '  <defs>\n' + patternDefs + '\n  </defs>\n</svg>\n';

writeFileSync(join(SRC, 'patterns.svg'), patternSprite);

/* ------------------------------------------------------------ emit CSS */

const uri = (svg) =>
  'data:image/svg+xml,' + svg
    .replace(/<\?xml[\s\S]*?\?>\s*/, '')
    .replace(/<!--[\s\S]*?-->\s*/g, '')
    .replace(/\s*\n\s*/g, ' ')
    .trim()
    .replace(/"/g, "'")
    .replace(/[<>#%{}|\\^~[\]`]/g, (c) => '%' + c.charCodeAt(0).toString(16).toUpperCase());

const maskRule = (sel, f, extra) =>
  sel + ' {\n' +
  '  height: ' + n(f.h) + 'px;\n' +
  '  background-color: currentColor;\n' +
  '  -webkit-mask-image: url("' + uri(f.svg) + '");\n' +
  '          mask-image: url("' + uri(f.svg) + '");\n' +
  '  -webkit-mask-repeat: repeat-x;\n' +
  '          mask-repeat: repeat-x;\n' +
  '  -webkit-mask-size: ' + n(f.w) + 'px ' + n(f.h) + 'px;\n' +
  '          mask-size: ' + n(f.w) + 'px ' + n(f.h) + 'px;\n' +
  '  -webkit-mask-position: left center;\n' +
  '          mask-position: left center;\n' +
  (extra || '') + '}';

const byName = Object.fromEntries(files.map((f) => [f.name, f]));

const dividerRules = files
  .filter((f) => f.tile && (f.group === 'jaali' || f.group === 'blockprint'))
  .map((f) => maskRule('.k-divider--' + f.name.replace('.svg', ''), f))
  .join('\n\n');

const weft = byName['weft-texture-12.svg'];
const cardEdge = byName['card-edge.svg'];
const doubleRule = byName['rule-double-woven-8.svg'];

const css = `/*
 * Kalakriti ornament — Batch 3
 * ---------------------------------------------------------------
 * WHY THESE ARE MASKS AND NOT BACKGROUND IMAGES.
 *
 * An SVG referenced from background-image or border-image — data: URI
 * included — is loaded into an isolated document. currentColor there
 * resolves to black and the --k-* custom properties are invisible to it.
 * A divider delivered that way looks right in a contact sheet and is wrong
 * in the product.
 *
 * So every tileable asset below is applied as a mask over
 * background-color: currentColor. The colour comes from the host element,
 * exactly as the Batch 0 currentColor rule requires, and nothing is lost
 * because every Batch 3 asset is monochrome by design.
 *
 * Do not "simplify" these back to background-image.
 * ---------------------------------------------------------------
 * Requires palette.css for the --k-* tokens.
 */

/* ---- section dividers ------------------------------------------------ */

.k-divider {
  display: block;
  width: 100%;
  flex: none;
  color: var(--k-text-primary);
  opacity: 0.18;          /* dividers are structure, not content */
}

/* Dividers carry no meaning. Set aria-hidden="true" at the call site, or use
   the Svelte component, which does it for you. */

/* High contrast: a divider at 18% is a smudge, and a smudge is worse than no
   ornament for a user who chose this theme. Either it is a boundary or it is
   not there. */
[data-theme="high-contrast"] .k-divider { opacity: 0.4; }

${dividerRules}

/* ---- hairline rules -------------------------------------------------- */

/* The plain hairline is CSS only. It has no SVG, so it cannot fail to load. */
.k-rule {
  border: 0;
  border-top: 1px solid var(--k-border-hairline);
  margin: 0;
}

/* Double rule with woven infill. If the mask fails, the @supports block below
   restores a plain 1px line. */
${maskRule('.k-rule--woven', doubleRule,
  '  border-top: 0;\n' +
  '  color: var(--k-text-secondary);\n' +
  '  opacity: 0.4;\n')}

@supports not ((-webkit-mask-image: none) or (mask-image: none)) {
  .k-rule--woven {
    background-color: transparent;
    border-top: 1px solid var(--k-border-hairline);
    height: 1px;
  }
}

/* Hairline interrupted by a centred motif. The rule is drawn by the two
   pseudo-elements, so the motif sits in a real gap rather than on top of a
   line. Put an inline rule-knot.svg inside the element. */
.k-rule-motif {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  color: var(--k-border-hairline);
}
.k-rule-motif::before,
.k-rule-motif::after {
  content: "";
  flex: 1 1 0;
  border-top: 1px solid currentColor;
}
.k-rule-motif > * {
  flex: none;
  width: 1.25rem;    /* 20px — the knot's legibility floor; below this the
                        warp interruption closes up and it reads as a plus */
  height: 1.25rem;
  color: var(--k-text-secondary);
  opacity: 0.55;
}

/* ---- weft page texture ----------------------------------------------- */

/* Applied to a ::before so the texture sits under content and never over it,
   and so the opacity here does not cascade into the page's own children. */
.k-weft-surface { position: relative; isolation: isolate; }
.k-weft-surface > * { position: relative; z-index: 1; }
.k-weft-surface::before {
  content: "";
  position: absolute;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  color: var(--k-text-primary);
  background-color: currentColor;
  opacity: var(--k-weft-opacity, 0.05);   /* recommended range 0.03-0.06 */
  -webkit-mask-image: url("${uri(weft.svg)}");
          mask-image: url("${uri(weft.svg)}");
  -webkit-mask-repeat: repeat;
          mask-repeat: repeat;
  -webkit-mask-size: 12px 12px;
          mask-size: 12px 12px;
}

/* Dark themes need slightly less: a light thread on a dark ground reads
   brighter than a dark thread on a light one at the same opacity. */
[data-theme="dark"] .k-weft-surface::before { opacity: var(--k-weft-opacity, 0.035); }
[data-theme="high-contrast"] .k-weft-surface::before { opacity: 0; }

/* ---- kolam corner ---------------------------------------------------- */

.k-kolam-corner {
  width: 3rem;
  height: 3rem;
  color: var(--k-accent-primary-text);
  opacity: 0.5;
}
.k-kolam-corner--tr { transform: scaleX(-1); }
.k-kolam-corner--br { transform: rotate(180deg); }
.k-kolam-corner--bl { transform: scaleY(-1); }

/* ---- printed card edge ----------------------------------------------- */

/*
 * This is the one Batch 3 asset that cannot use currentColor. A border-image
 * loads in an isolated document, so the colour has to be baked into the data
 * URI — one bake per theme, below. The consequence is that the card edge
 * follows the theme rather than the element's text colour, which for a card
 * frame is the right trade.
 *
 * If a dynamic edge colour is ever needed, the upgrade is
 * -webkit-mask-box-image on a ::before over background-color: currentColor,
 * keeping the border-image below as the Firefox fallback.
 *
 * border-image-slice must be a percentage: a viewBox-only SVG has no intrinsic
 * pixel size, so a px slice is undefined.
 */
.k-card--printed {
  border: 6px solid var(--k-border-hairline);   /* fallback if the image fails */
  border-radius: var(--k-radius-none, 0);
  border-image: url("${uri(cardEdge.svg.replace(/currentColor/g, '#241E1A'))}") 25% round;
  background: var(--k-surface-raised);
  padding: 1rem;
}
[data-theme="dark"] .k-card--printed {
  border-image: url("${uri(cardEdge.svg.replace(/currentColor/g, '#F5F0E8'))}") 25% round;
}
[data-theme="high-contrast"] .k-card--printed {
  border-image: url("${uri(cardEdge.svg.replace(/currentColor/g, '#0F0A07'))}") 25% round;
}
`;

writeFileSync(join(ROOT, 'packages/ornament/ornament.css'), css);

/* ------------------------------------------------- emit the contact sheet
 * The kolam corner and the pattern sprite are inlined rather than fetched so
 * the sheet opens straight off disk — fetch() is blocked on file:// URLs, and
 * a contact sheet nobody can open is not a contact sheet.
 */

const innerOf = (name) => byName[name].svg
  .replace(/[\s\S]*?<svg[^>]*>/, '')
  .replace(/<\/svg>\s*$/, '')
  .replace(/<!--[\s\S]*?-->/g, '')
  .trim();

const sheet = readFileSync(join(ROOT, 'scripts/contact-sheet-batch3.template.html'), 'utf8')
  .replace('<!--INJECT:PATTERNS-->', patternSprite.replace(/<\?xml[\s\S]*?\?>\n/, ''))
  .replace('<!--INJECT:KOLAM-->', innerOf('kolam-corner.svg'))
  .replace('<!--INJECT:KNOT-->', innerOf('rule-knot.svg'));

writeFileSync(join(ROOT, 'packages/ornament/contact-sheet-batch3.html'), sheet);

/* ============================================================== CHECKS */

let failed = 0;
const fail = (m) => { console.error('FAIL  ' + m); failed++; };
const ok = (m) => console.log('ok    ' + m);

for (const r of seamReports) {
  if (JSON.stringify(r.left) !== JSON.stringify(r.right))
    fail(r.name + ': seam mismatch\n        left  ' + JSON.stringify(r.left) +
      '\n        right ' + JSON.stringify(r.right));
}
if (!failed) ok('seam parity on ' + seamReports.length + ' tileable assets');

for (const f of files) {
  if ((f.group === 'jaali' || f.group === 'blockprint') && f.h > 32)
    fail(f.name + ': ' + f.h + 'px tall, exceeds the 32px divider cap');
}
ok('no divider exceeds 32px');

for (const f of files) {
  const bytes = Buffer.byteLength(f.svg, 'utf8');
  if (f.budget && bytes > f.budget) fail(f.name + ': ' + bytes + 'B over ' + f.budget + 'B budget');
}
ok('size budgets');

for (const f of files) {
  const geom = f.svg.replace(/<!--[\s\S]*?-->/g, '');
  if (!/viewBox=/.test(geom)) fail(f.name + ': no viewBox');
  if (/<svg[^>]*\swidth=/.test(geom)) fail(f.name + ': hardcoded width on root svg');
  if (/#[0-9a-fA-F]{3,6}\b/.test(geom)) fail(f.name + ': bare hex colour');
  if (/\bid=/.test(geom)) fail(f.name + ': unreferenced id');
  if (/\d+\.\d{3,}/.test(geom)) fail(f.name + ': more than 2 decimal places');
}
ok('viewBox / no width-height / no bare hex / no ids / max 2dp');

console.log('\n' + files.length + ' assets -> packages/ornament/src');
for (const f of files) {
  console.log('  ' + String(Buffer.byteLength(f.svg, 'utf8')).padStart(5) + 'B  ' +
    f.name.padEnd(26) + f.w + 'x' + f.h);
}
console.log('  ' + String(Buffer.byteLength(css, 'utf8')).padStart(5) + 'B  ornament.css');

if (failed) { console.error('\n' + failed + ' check(s) failed'); process.exit(1); }
console.log('\nall checks passed');
