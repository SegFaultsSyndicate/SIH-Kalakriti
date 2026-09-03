/*
 * Kalakriti assets — Batch 6 generator
 * Patterns, textures, and surface treatments: six seamless pattern tiles,
 * paper grain, section backgrounds, card surfaces, skeleton shimmer, the
 * provenance page backdrop, and the QR frame.
 *
 * Run:  node scripts/gen-batch6.mjs
 *
 * Same discipline as Batches 3-5: primitives, not hand-typed path data;
 * a seam-parity check on every tileable asset; a real render before
 * shipping. Batch 3 already built the jaali DIVIDER (a horizontal band)
 * and the weft LINE texture — this batch's jaali/weft tiles are 2D FILL
 * patterns for panels and page backgrounds, a different job, so they are
 * new assets here rather than re-exports.
 */

import { writeFileSync, readFileSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import zlib from 'node:zlib';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const SRC = join(ROOT, 'packages/patterns/src');
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
const diamond = (cx, cy, r) => poly([[cx, cy - r], [cx + r, cy], [cx, cy + r], [cx - r, cy]], true);

const seamProbe = () => {
  const left = [], right = [], top = [], bottom = [];
  return {
    left, right, top, bottom,
    noteX(w, x, y0, y1 = y0) {
      const key = n(Math.min(y0, y1)) + '..' + n(Math.max(y0, y1));
      if (Math.abs(x) < 1e-6) left.push(key); else if (Math.abs(x - w) < 1e-6) right.push(key);
    },
    noteY(h, y, x0, x1 = x0) {
      const key = n(Math.min(x0, x1)) + '..' + n(Math.max(x0, x1));
      if (Math.abs(y) < 1e-6) top.push(key); else if (Math.abs(y - h) < 1e-6) bottom.push(key);
    },
  };
};

const tiles = []; // { name, svg, w, h, budget, opacity, tradition }

function addTile(name, w, h, body, opts) {
  const svg =
    '<?xml version="1.0" encoding="UTF-8"?>\n' +
    '<svg viewBox="0 0 ' + n(w) + ' ' + n(h) + '" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">\n' +
    '  <!-- ' + opts.comment + ' Recommended opacity: ' + opts.opacity + '. -->\n' +
    '  ' + body + '\n' +
    '</svg>\n';
  tiles.push({ name, svg, w, h, ...opts });
  return svg;
}

/* =========================================================== 1. AJRAKH
 * Dense geometric repeat. Ajrakh layered resist-dye passes read as a
 * tessellated eight-pointed star (octagram) with a resist-cut centre —
 * fill-led, hard corners, evenodd negative space.
 */
{
  const P = 24;
  const probe = seamProbe();
  const star = (cx, cy, r1, r2) => {
    const pts = [];
    for (let i = 0; i < 8; i++) {
      const a1 = (Math.PI / 4) * i - Math.PI / 8;
      const a2 = (Math.PI / 4) * i;
      pts.push([cx + r1 * Math.cos(a1), cy + r1 * Math.sin(a1)]);
      pts.push([cx + r2 * Math.cos(a2), cy + r2 * Math.sin(a2)]);
    }
    return poly(pts, true);
  };
  const d = [star(0, 0, 5, 11), star(P, 0, 5, 11), star(0, P, 5, 11), star(P, P, 5, 11), star(P / 2, P / 2, 5, 11), diamond(P / 2, P / 2, 2.2)];
  probe.noteX(P, 0, 0); probe.noteX(P, P, 0);
  probe.noteX(P, 0, P); probe.noteX(P, P, P);
  probe.noteY(P, 0, 0); probe.noteY(P, P, 0);
  probe.noteY(P, 0, P); probe.noteY(P, P, P);
  const body = '<path fill="currentColor" fill-rule="evenodd" d="' + d.join(' ') + '"/>';
  addTile('pattern-ajrakh', P, P, body, {
    comment: 'Block print (Ajrakh). Dense tessellated eight-pointed star, resist-cut centre, fill-led with evenodd negative space. Corner stars combine at the tile edge into one star per corner, in the Ajrakh layered-stamp idiom.',
    opacity: '5-8%, dense surfaces only (panels, not body-text backgrounds)',
    tradition: 'block print (Ajrakh)', budget: 2048, tile2d: true, probe,
  });
}

/* ============================================================ 2. BAGRU
 * Medium floral-geometric repeat: a four-petal rosette on sprigs, the
 * Bagru idiom -- softer, more floral than Ajrakh hard geometry.
 */
{
  const P = 40;
  const probe = seamProbe();
  const petal = (cx, cy, ang, len) => {
    const x1 = cx + len * Math.cos(ang), y1 = cy + len * Math.sin(ang);
    return 'M' + pt(cx, cy) + ' Q' + pt(x1 + 4 * Math.cos(ang + Math.PI / 2), y1 + 4 * Math.sin(ang + Math.PI / 2)) + ' ' + pt(x1, y1) +
      ' Q' + pt(x1 + 4 * Math.cos(ang - Math.PI / 2), y1 + 4 * Math.sin(ang - Math.PI / 2)) + ' ' + pt(cx, cy) + ' Z';
  };
  const rosette = (cx, cy) => {
    const parts = [dot(cx, cy, 1.6)];
    for (let i = 0; i < 4; i++) parts.push(petal(cx, cy, (Math.PI / 2) * i, 6.5));
    return parts.join(' ');
  };
  const sprig = (x0, y0, x1, y1) => line(x0, y0, x1, y1);
  const d = [rosette(0, 0), rosette(P, 0), rosette(0, P), rosette(P, P), rosette(P / 2, P / 2)];
  const strokes = [sprig(0, 0, P / 2, P / 2), sprig(P, 0, P / 2, P / 2), sprig(0, P, P / 2, P / 2), sprig(P, P, P / 2, P / 2)];
  probe.noteX(P, 0, 0); probe.noteX(P, P, 0);
  probe.noteX(P, 0, P); probe.noteX(P, P, P);
  probe.noteY(P, 0, 0); probe.noteY(P, P, 0);
  probe.noteY(P, 0, P); probe.noteY(P, P, P);
  const body =
    '<path fill="currentColor" d="' + d.join(' ') + '"/>\n' +
    '  <path fill="none" stroke="currentColor" stroke-width="0.6" stroke-linecap="round" d="' + strokes.join(' ') + '"/>';
  addTile('pattern-bagru', P, P, body, {
    comment: 'Block print (Bagru). Medium floral-geometric repeat: a four-petal rosette at each grid point, joined by hairline sprigs, softer and more floral than Ajrakh hard star geometry, per motifs.md per-tradition distinction.',
    opacity: '6-10%, medium surfaces (section backgrounds, wide panels)',
    tradition: 'block print (Bagru)', budget: 2048, tile2d: true, probe,
  });
}

/* ============================================================= 3. DABU
 * Fine dotted resist: Dabu mud-resist printing reads as a dense, slightly
 * irregular scatter of small resist dots -- fine density, near-textile.
 */
{
  const P = 16;
  const dots = [[2, 2], [10, 4], [4, 9], [13, 11], [8, 14]];
  const d = dots.map(([x, y]) => dot(x, y, 1)).join(' ');
  const wrapped = [];
  for (const [x, y] of dots) {
    if (x < 1.5) wrapped.push(dot(x + P, y, 1));
    if (y < 1.5) wrapped.push(dot(x, y + P, 1));
  }
  const body = '<path fill="currentColor" d="' + d + ' ' + wrapped.join(' ') + '"/>';
  addTile('pattern-dabu', P, P, body, {
    comment: 'Block print (Dabu, mud-resist). Fine scattered resist dots, deliberately irregular spacing within one repeat unit rather than a perfect grid, per Dabu hand-applied resist paste.',
    opacity: '8-14%, fine surfaces (cards, small panels)',
    tradition: 'block print (Dabu)', budget: 1024, tile2d: false,
  });
}

/* ===================================================== 4. KOLAM DOT-GRID
 * Very fine, near-invisible -- just the dot grid a kolam is drawn around,
 * with no connecting line. This is the substrate, not the drawn motif;
 * kolam-corner.svg (Batch 3) is the drawn line.
 */
{
  const P = 24; // matches the 24px kolam pitch in stroke-geometry-spec.md
  const body = '<path fill="currentColor" d="' + dot(P / 2, P / 2, 0.9) + '"/>';
  addTile('pattern-kolam-dotgrid', P, P, body, {
    comment: 'Kolam. The dot grid a kolam line is drawn around (motifs.md), with no line — the substrate pattern for large surfaces, at 24px pitch matching stroke-geometry-spec.md kolam grid unit.',
    opacity: '4-6%, large surfaces (page backgrounds)',
    tradition: 'kolam / rangoli', budget: 512, tile2d: false,
  });
}

/* ===================================================== 5. WARP-AND-WEFT
 * The workhorse. A fresh 2D plain-weave fill tile for panels and page
 * backgrounds -- the same over/under logic as Batch 3 weft-texture-12.svg,
 * which is a divider-scale line texture; this is the panel-fill version
 * Batch 6 asks for as its own numbered deliverable.
 */
{
  const P = 12;
  const probe = seamProbe();
  const d = [
    'M' + pt(2, 0) + ' L' + pt(4, 0) + ' L' + pt(4, 8) + ' L' + pt(2, 8) + ' Z',
    'M' + pt(2, 10) + ' L' + pt(4, 10) + ' L' + pt(4, 12) + ' L' + pt(2, 12) + ' Z',
    'M' + pt(8, 0) + ' L' + pt(10, 0) + ' L' + pt(10, 2) + ' L' + pt(8, 2) + ' Z',
    'M' + pt(8, 4) + ' L' + pt(10, 4) + ' L' + pt(10, 12) + ' L' + pt(8, 12) + ' Z',
    'M' + pt(0, 2) + ' L' + pt(2, 2) + ' L' + pt(2, 4) + ' L' + pt(0, 4) + ' Z',
    'M' + pt(4, 2) + ' L' + pt(12, 2) + ' L' + pt(12, 4) + ' L' + pt(4, 4) + ' Z',
    'M' + pt(0, 8) + ' L' + pt(8, 8) + ' L' + pt(8, 10) + ' L' + pt(0, 10) + ' Z',
    'M' + pt(10, 8) + ' L' + pt(12, 8) + ' L' + pt(12, 10) + ' L' + pt(10, 10) + ' Z',
  ];
  probe.noteX(P, 0, 2, 4); probe.noteX(P, P, 2, 4);
  probe.noteX(P, 0, 8, 10); probe.noteX(P, P, 8, 10);
  probe.noteY(P, 0, 2, 4); probe.noteY(P, P, 2, 4);
  probe.noteY(P, 0, 8, 10); probe.noteY(P, P, 8, 10);
  const body = '<path fill="currentColor" d="' + d.join(' ') + '"/>';
  addTile('pattern-warp-weft', P, P, body, {
    comment: 'Handloom structure (plain weave), the workhorse pattern. Panel/page-background fill version of the over/under weave, the same construction as Batch 3 weft-texture-12.svg divider tile, delivered here as its own Batch 6 asset per the brief.',
    opacity: '3-6%, any surface, the most broadly licensed pattern in the set',
    tradition: 'handloom structure', budget: 1024, tile2d: true, probe,
  });
}

/* ======================================================= 6. JAALI FILL
 * A 2D lattice FILL for panels and cards -- solid ground with apertures cut
 * out (evenodd), which is closer to real pierced-stone jaali (solid
 * material, holes cut in it) than Batch 3 stroke-drawn divider bands.
 * Hard corners (miter) per stroke-geometry-spec.md.
 */
{
  const P = 28;
  const probe = seamProbe();
  const aperture = (cx, cy) => diamond(cx, cy, 8);
  const d = [
    'M0 0 H' + P + ' V' + P + ' H0 Z',
    aperture(0, 0), aperture(P, 0), aperture(0, P), aperture(P, P), aperture(P / 2, P / 2),
  ];
  probe.noteX(P, 0, 0); probe.noteX(P, P, 0);
  probe.noteX(P, 0, P); probe.noteX(P, P, P);
  probe.noteY(P, 0, 0); probe.noteY(P, P, 0);
  probe.noteY(P, 0, P); probe.noteY(P, P, P);
  const body = '<path fill="currentColor" fill-rule="evenodd" stroke="currentColor" stroke-width="0.75" stroke-linejoin="miter" d="' + d.join(' ') + '"/>';
  addTile('pattern-jaali-fill', P, P, body, {
    comment: 'Jaali. Solid ground with diamond apertures cut by fill-rule evenodd, a filled panel with holes, closer to real pierced stone than Batch 3 stroke-drawn jaali dividers (a different job: horizontal bands, not a 2D panel fill). Hard corners per stroke-geometry-spec.md.',
    opacity: '4-8%, panels and cards (this is the jaali lattice fill the brief asks for)',
    tradition: 'jaali', budget: 2048, tile2d: true, probe,
  });
}

/* --------------------------------------------------------------- emit */

for (const t of tiles) writeFileSync(join(SRC, t.name + '.svg'), t.svg);

/* ========================================================= PAPER GRAIN
 * SVG feTurbulence, tuned low so it reads as handmade paper, not noise.
 * Plus a real PNG fallback (Node's built-in zlib only, no new dependency,
 * per ponytail: stdlib before a library) for the throttled-device case
 * where a filter primitive costs frames.
 */

{
  const svg =
    '<?xml version="1.0" encoding="UTF-8"?>\n' +
    '<svg viewBox="0 0 200 200" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">\n' +
    '  <!-- Paper grain, feTurbulence-based. Tuned low (baseFrequency 0.9, 2 octaves) so it reads as handmade paper texture, not visual noise. Tile at 200x200; recommended opacity 4-7%. Verify on a throttled low-end profile before shipping — if it costs frames, use grain.png (below) instead; both are delivered so the choice can be made per-surface, not guessed. -->\n' +
    '  <filter id="k-grain" x="0" y="0" width="100%" height="100%">\n' +
    '    <feTurbulence type="fractalNoise" baseFrequency="0.9" numOctaves="2" stitchTiles="stitch" seed="7" result="noise"/>\n' +
    '    <feColorMatrix in="noise" type="matrix" values="0 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0.9 0 0 0 0"/>\n' +
    '  </filter>\n' +
    '  <rect width="200" height="200" fill="currentColor" filter="url(#k-grain)"/>\n' +
    '</svg>\n';
  writeFileSync(join(SRC, 'paper-grain.svg'), svg);
  tiles.push({ name: 'paper-grain', svg, w: 200, h: 200, budget: 1024, tradition: 'none — paper texture, not a motif', opacity: '4-7%', isFilter: true });
}

// --- tiny stdlib PNG encoder: an 8x8 greyscale noise tile, deflate-only ---
function crc32(buf) {
  let c, crcTable = crc32.table || (crc32.table = (() => {
    const t = [];
    for (let n = 0; n < 256; n++) {
      c = n;
      for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
      t[n] = c;
    }
    return t;
  })());
  let crc = 0xffffffff;
  for (let i = 0; i < buf.length; i++) crc = crcTable[(crc ^ buf[i]) & 0xff] ^ (crc >>> 8);
  return (crc ^ 0xffffffff) >>> 0;
}
function pngChunk(type, data) {
  const len = Buffer.alloc(4); len.writeUInt32BE(data.length);
  const typeData = Buffer.concat([Buffer.from(type, 'ascii'), data]);
  const crc = Buffer.alloc(4); crc.writeUInt32BE(crc32(typeData));
  return Buffer.concat([len, typeData, crc]);
}
function makeGrainPng(size, seedFn) {
  const sig = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
  const ihdrData = Buffer.alloc(13);
  ihdrData.writeUInt32BE(size, 0); ihdrData.writeUInt32BE(size, 4);
  ihdrData[8] = 8; ihdrData[9] = 0; ihdrData[10] = 0; ihdrData[11] = 0; ihdrData[12] = 0; // 8-bit greyscale
  const ihdr = pngChunk('IHDR', ihdrData);
  const raw = Buffer.alloc(size * (size + 1));
  let seed = 1;
  const rnd = () => { seed = (seed * 1103515245 + 12345) & 0x7fffffff; return seed / 0x7fffffff; };
  for (let y = 0; y < size; y++) {
    raw[y * (size + 1)] = 0; // filter type: none
    for (let x = 0; x < size; x++) {
      const v = 200 + Math.floor(rnd() * 55); // light greys, paper-like
      raw[y * (size + 1) + 1 + x] = v;
    }
  }
  const idat = pngChunk('IDAT', zlib.deflateSync(raw));
  const iend = pngChunk('IEND', Buffer.alloc(0));
  return Buffer.concat([sig, ihdr, idat, iend]);
}
const grainPng = makeGrainPng(48);
writeFileSync(join(SRC, 'paper-grain.png'), grainPng);

/* ============================================== SECTION BACKGROUNDS,
 * card surfaces, skeleton shimmer, provenance backdrop, QR frame
 * — all CSS (plus one QR-frame SVG), written directly below.
 */

const bg = readFileSync(join(ROOT, 'packages/tokens/src/palette.css'), 'utf8');
const hasIndigo900 = /--k-indigo-900/.test(bg);

const qrProbe = (() => {
  // QR quiet zone = 4 modules minimum per ISO/IEC 18004. We don't know the
  // consumer's module size, so the frame's clear inner square is expressed
  // as a percentage of the frame, and the component contract requires the
  // caller to size the QR itself to exactly that inner square — the frame
  // cannot enforce module count, only reserve the space for it.
  const OUT = 240, QUIET_PCT = 0.14; // 14% each side reserved, well over 4 modules at typical QR module counts
  const inner = OUT * (1 - QUIET_PCT * 2);
  return { OUT, QUIET_PCT, inner };
})();

{
  const OUT = qrProbe.OUT, ringR = OUT / 2 - 6, teeth = 24;
  const cx = OUT / 2, cy = OUT / 2;
  const parts = [];
  for (let i = 0; i < teeth; i++) {
    const a = (Math.PI * 2 * i) / teeth;
    const r1 = ringR - 5, r2 = ringR + 5;
    parts.push(line(cx + r1 * Math.cos(a), cy + r1 * Math.sin(a), cx + r2 * Math.cos(a), cy + r2 * Math.sin(a)));
  }
  const svg =
    '<?xml version="1.0" encoding="UTF-8"?>\n' +
    '<svg viewBox="0 0 ' + OUT + ' ' + OUT + '" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">\n' +
    '  <!-- Jaali. Decorative surround for the provenance QR. The QR itself is NOT drawn here — the consuming page places its own QR image inside the ' + n(qrProbe.inner) + 'x' + n(qrProbe.inner) + ' clear inner square (out of a ' + OUT + 'x' + OUT + ' frame), which reserves ' + n(qrProbe.QUIET_PCT * 100) + '% of the frame on every side as quiet zone. Per ISO/IEC 18004 a QR needs a quiet zone of at least 4 modules; at the smallest module count this system\'s QR uses (~25 modules across for a typical provenance URL), 14% of the frame is several modules, comfortably over the minimum, but the exact margin depends on the QR\'s own module count and must be confirmed for the final printed size. THIS HAS NOT BEEN VERIFIED WITH A REAL PHONE SCAN — the prompt pack\'s own warning: test with a printed tag before week three, not on demo morning. -->\n' +
    '  <g fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">\n' +
    '    <path d="' + circ(cx, cy, ringR) + '"/>\n' +
    '    <path d="' + parts.join(' ') + '"/>\n' +
    '  </g>\n' +
    '</svg>\n';
  writeFileSync(join(SRC, 'qr-frame.svg'), svg);
  tiles.push({ name: 'qr-frame', svg, w: OUT, h: OUT, budget: 4096, tradition: 'jaali', opacity: '100% (foreground device, not a background pattern)' });
}

/* ============================================================== CHECKS */

let failed = 0;
const fail = (m) => { console.error('FAIL  ' + m); failed++; };
const ok = (m) => console.log('ok    ' + m);

for (const t of tiles) {
  const geom = t.svg.replace(/<!--[\s\S]*?-->/g, '');
  if (!/viewBox=/.test(geom)) fail(t.name + '.svg: no viewBox');
  if (/<svg[^>]*\swidth=/.test(geom)) fail(t.name + '.svg: hardcoded width on root svg');
  if (/#[0-9a-fA-F]{3,6}\b/.test(geom.replace(/var\([^)]*\)/g, ''))) fail(t.name + '.svg: bare hex colour outside a var() fallback');
  // ids are allowed only when referenced (paper-grain's filter id, used by its own url(#...))
  const ids = [...geom.matchAll(/\bid="([^"]+)"/g)].map((m) => m[1]);
  for (const id of ids) if (!new RegExp('url\\(#' + id + '\\)').test(geom)) fail(t.name + '.svg: unreferenced id "' + id + '"');
  const bytes = Buffer.byteLength(t.svg, 'utf8');
  if (t.budget && bytes > t.budget) fail(t.name + '.svg: ' + bytes + 'B over ' + t.budget + 'B budget');
  if (t.probe) {
    const p = t.probe;
    if (JSON.stringify(p.left.sort()) !== JSON.stringify(p.right.sort()))
      fail(t.name + '.svg: X-axis seam mismatch');
    if (JSON.stringify(p.top.sort()) !== JSON.stringify(p.bottom.sort()))
      fail(t.name + '.svg: Y-axis seam mismatch');
  }
}
if (!failed) ok(tiles.length + ' tiles/devices: viewBox / no width-height / no bare hex / referenced-only ids / size budget / 2D seam parity');

const pngBytes = grainPng.length;
if (pngBytes > 3072) fail('paper-grain.png: ' + pngBytes + 'B over the 3KB budget');
else ok('paper-grain.png: ' + pngBytes + 'B, under the 3KB budget');

console.log('\n' + tiles.length + ' pattern/texture assets -> packages/patterns/src (+ paper-grain.png)');
for (const t of tiles) console.log('  ' + String(Buffer.byteLength(t.svg, 'utf8')).padStart(5) + 'B  ' + t.name.padEnd(24) + t.w + 'x' + t.h + '  opacity ' + (t.opacity || ''));
console.log('  ' + String(pngBytes).padStart(5) + 'B  paper-grain.png');

if (failed) { console.error('\n' + failed + ' check(s) failed'); process.exit(1); }
console.log('\nall checks passed');

/* ------------------------------------------------------- contact sheet */

const innerBody = (svg) => svg
  .replace(/[\s\S]*?<svg[^>]*>\n/, '')
  .replace(/<\/svg>\s*$/, '')
  .replace(/<!--[\s\S]*?-->\n?/g, '')
  .trim();

const patternTiles = tiles.filter((t) => !t.isFilter && t.name !== 'qr-frame');

const seamStrip = (t) => {
  const w4 = t.w * 4;
  return '<div class="seam-row"><svg viewBox="0 0 ' + w4 + ' ' + t.h + '" style="width:' + Math.min(w4, 400) + 'px;height:' + t.h + 'px;opacity:.6" aria-hidden="true">' +
    '<defs><pattern id="seam-' + t.name + '" width="' + t.w + '" height="' + t.h + '" patternUnits="userSpaceOnUse">' + innerBody(t.svg) + '</pattern></defs>' +
    '<rect width="' + w4 + '" height="' + t.h + '" fill="url(#seam-' + t.name + ')"/></svg>' +
    '<small>' + t.name + ' — 4 tiles, ' + t.w + 'x' + t.h + ' each</small></div>';
};

const opacityStudy = (t, levels) => '<div class="op-study"><h4>' + t.name + '</h4><div class="op-row">' +
  levels.map((op) => '<div class="op-cell" style="mask-image:url(\'data:image/svg+xml,' + encodeURIComponent(t.svg.replace(/currentColor/g, '#000')) + '\');-webkit-mask-image:url(\'data:image/svg+xml,' + encodeURIComponent(t.svg.replace(/currentColor/g, '#000')) + '\');mask-repeat:repeat;-webkit-mask-repeat:repeat;mask-size:' + t.w + 'px ' + t.h + 'px;-webkit-mask-size:' + t.w + 'px ' + t.h + 'px;opacity:' + op + '"><span>' + Math.round(op * 100) + '%</span></div>').join('') +
  '</div></div>';

const themeSection = (theme, label) => `
<section class="theme" data-theme="${theme}">
  <header><h1>Batch 6 — patterns, textures, backgrounds</h1><span>${label}</span></header>

  <h2>1 · Six pattern tiles — 4-tile seam test</h2>
  <p class="note">Each strip is exactly four tile-widths, at raised opacity. A doubled line, gap or kink at 1/4, 1/2 or 3/4 across is a seam failure.</p>
  ${patternTiles.map(seamStrip).join('')}

  <h2>2 · Opacity studies</h2>
  <p class="note">Recommended range in each tile's own comment; shown here at 3 points across that range against real body text.</p>
  <div class="op-text">Khadi cotton, handspun. Warp and weft, plain weave, the workhorse pattern of this system.</div>
  ${opacityStudy(tiles.find(t=>t.name==='pattern-ajrakh'), [0.03, 0.06, 0.1])}
  ${opacityStudy(tiles.find(t=>t.name==='pattern-bagru'), [0.04, 0.08, 0.12])}
  ${opacityStudy(tiles.find(t=>t.name==='pattern-dabu'), [0.06, 0.11, 0.16])}
  ${opacityStudy(tiles.find(t=>t.name==='pattern-kolam-dotgrid'), [0.03, 0.05, 0.08])}
  ${opacityStudy(tiles.find(t=>t.name==='pattern-warp-weft'), [0.03, 0.045, 0.06])}
  ${opacityStudy(tiles.find(t=>t.name==='pattern-jaali-fill'), [0.03, 0.06, 0.09])}

  <h2>3 · Paper grain</h2>
  <div class="grain-row">
    <div class="grain-cell k-grain"><div class="grain-label">SVG feTurbulence, 5%</div></div>
    <div class="grain-cell k-grain k-grain--raster"><div class="grain-label">PNG fallback, ${pngBytes}B</div></div>
  </div>

  <h2>4 · Section backgrounds</h2>
  <div class="section-demo k-section--khadi-plain"><strong>khadi-plain</strong><p>Base surface, no texture.</p></div>
  <div class="section-demo k-section--khadi-weft k-pattern"><strong>khadi-with-weft</strong><p>Weft mask at 4.5%.</p></div>
  <div class="section-demo k-section--indigo-panel"><strong>indigo-panel</strong><p>Khadi-50 text, 11.50:1 measured. <a href="#">A link</a> at 7.73:1.</p></div>

  <h2>5 · Card surfaces</h2>
  <div class="card-row">
    <div class="k-card k-card--flat">flat</div>
    <div class="k-card k-card--hairline">hairline-bordered</div>
    <div class="k-card--printed">block-print-edged (Batch 3's k-card--printed, referenced not duplicated)</div>
  </div>

  <h2>6 · Loading skeleton shimmer</h2>
  <div class="skeleton-row">
    <div class="k-skeleton" style="width:220px;height:90px"></div>
    <div class="k-skeleton k-skeleton--reduced-demo" style="width:220px;height:90px"></div>
  </div>
  <p class="note">Right: reduced-motion variant (static, no sweep) — force-simulated with a demo-only class since the sheet can't toggle the OS setting; ships as the real <code>prefers-reduced-motion</code> media query in patterns.css.</p>

  <h2>7 · QR frame — quiet zone</h2>
  <div class="k-qr-frame">
    <img src="./src/qr-frame.svg" alt="" aria-hidden="true">
    <div class="k-qr-frame__slot"><div class="qr-placeholder">QR<br>goes here</div></div>
  </div>
  <p class="note">Inner slot is ${n(qrProbe.inner)}×${n(qrProbe.inner)} out of a ${qrProbe.OUT}×${qrProbe.OUT} frame — ${n(qrProbe.QUIET_PCT*100)}% reserved per side. NOT verified against a real phone scan; see the SVG's own comment and provenance.css.</p>
</section>`;

const sheet = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Kalakriti — Batch 6 contact sheet: patterns and surfaces</title>
<link rel="stylesheet" href="../tokens/src/palette.css">
<link rel="stylesheet" href="../ornament/ornament.css">
<link rel="stylesheet" href="./patterns.css">
<style>
  body { margin: 0; font: 14px/1.5 ui-sans-serif, system-ui, sans-serif; }
  .theme { background: var(--k-surface-base); color: var(--k-text-primary); padding: 2.5rem clamp(1rem,4vw,3rem) 3.5rem; }
  .theme > header { display: flex; align-items: baseline; gap: 0.75rem; margin: 0 0 2rem; padding-bottom: 0.6rem; border-bottom: 1px solid var(--k-border-hairline); }
  .theme > header h1 { font-size: 1.1rem; margin: 0; }
  .theme > header span { color: var(--k-text-secondary); font-size: 0.8rem; }
  h2 { font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.09em; color: var(--k-text-secondary); margin: 2.5rem 0 0.9rem; font-weight: 600; }
  h2:first-of-type { margin-top: 0; }
  h4 { font-size: 0.75rem; margin: 0 0 0.4rem; }
  .note { font-size: 0.75rem; color: var(--k-text-secondary); max-width: 62ch; margin: 0.3rem 0 1rem; }
  .seam-row { display: flex; align-items: center; gap: 1rem; margin-bottom: 0.6rem; }
  .seam-row svg { outline: 1px dashed var(--k-border-hairline); outline-offset: 3px; }
  .seam-row small { color: var(--k-text-secondary); font-size: 0.68rem; }
  .op-text { padding: 1rem; border: 1px solid var(--k-border-hairline); background: var(--k-surface-raised); margin-bottom: 1rem; max-width: 40rem; }
  .op-study { margin-bottom: 1.25rem; }
  .op-row { display: flex; gap: 0.75rem; }
  .op-cell { width: 90px; height: 60px; background-color: currentColor; color: var(--k-text-primary); position: relative; border: 1px solid var(--k-border-hairline); }
  .op-cell span { position: absolute; bottom: 2px; right: 4px; font-size: 0.6rem; color: var(--k-text-secondary); mix-blend-mode: normal; }
  .grain-row { display: flex; gap: 1rem; }
  .grain-cell { width: 200px; height: 100px; border: 1px solid var(--k-border-hairline); background: var(--k-surface-raised); display: flex; align-items: flex-end; padding: 0.5rem; }
  .grain-label { font-size: 0.68rem; color: var(--k-text-secondary); position: relative; z-index: 1; }
  .section-demo { padding: 1.25rem; margin-bottom: 0.75rem; max-width: 32rem; }
  .card-row { display: flex; gap: 1rem; flex-wrap: wrap; }
  .card-row > * { padding: 1rem; width: 200px; font-size: 0.8rem; }
  .skeleton-row { display: flex; gap: 1rem; }
  /* demo-only: simulates prefers-reduced-motion without needing the OS setting toggled */
  .k-skeleton--reduced-demo::after { animation: none !important; transform: none !important; opacity: 0.28 !important; }
  .qr-placeholder { font-size: 0.65rem; text-align: center; color: var(--k-text-secondary); border: 1px dashed var(--k-border-hairline); width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; }
</style>
</head>
<body>
${themeSection('light', 'light theme')}
${themeSection('dark', 'dark theme')}
${themeSection('high-contrast', 'high-contrast theme')}
</body>
</html>
`;

writeFileSync(join(ROOT, 'packages/patterns/contact-sheet-batch6.html'), sheet);
console.log('contact-sheet-batch6.html written');
