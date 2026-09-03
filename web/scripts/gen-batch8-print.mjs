/*
 * Kalakriti — Batch 8 print-ready provenance tag
 *
 * Unlike every other asset in this system, this one is not consumed by a
 * browser with a CSS cascade to supply currentColor/var() — it is opened
 * directly by a printer's RIP or a person's Illustrator/Inkscape install.
 * So, deliberately and only here: literal mm dimensions on the root <svg>,
 * and explicit brand hex instead of currentColor/var(--k-token). Both are
 * violations of every other package's rules; both are correct for a file
 * whose only consumer is a print pipeline with no CSS engine at all. This
 * is documented, not a slip — see packages/print/README.md.
 *
 * Tag = 90mm x 54mm swing-tag, laid out 2x4 (8-up) on an A4 sheet with
 * corner crop marks. Seal reuses identity/seal-print.svg's geometry at
 * print scale; the QR slot reuses patterns/qr-frame.svg's quiet-zone
 * geometry and its own unverified-scan caveat verbatim. Artisan/craft
 * fields are blank rules — filled by hand or by variable-data print,
 * not by this generator.
 */

import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const OUT = join(ROOT, 'packages/print/src');
mkdirSync(OUT, { recursive: true });

const INK = { colour: '#241E1A', paper: '#FCFAF6', accent: '#963D14' };
const MONO = { colour: '#000000', paper: '#FFFFFF', accent: '#000000' };

const TAG_W = 90, TAG_H = 54; // mm, standard swing-tag size

function n(x) { return Math.round(x * 100) / 100; }

// Pull just the <path> geometry out of seal-print.svg (viewBox 0 0 200 200)
// and qr-frame.svg (viewBox 0 0 240 240), dropping their own wrapper/aria
// attrs — this file supplies its own.
function innerGeometry(file) {
  const raw = readFileSync(join(ROOT, file), 'utf8');
  const m = raw.match(/<svg[^>]*>([\s\S]*)<\/svg>/);
  return m[1].replace(/<title[^>]*>[\s\S]*?<\/title>/, '').replace(/<!--[\s\S]*?-->/g, '').trim();
}
const sealGeom = innerGeometry('packages/identity/src/seal-print.svg');
const qrGeom = innerGeometry('packages/patterns/src/qr-frame.svg');

function tag(pal) {
  const sealSize = 14, sealX = TAG_W - sealSize - 4, sealY = 4;
  const qrSize = 16, qrX = TAG_W - qrSize - 4, qrY = TAG_H - qrSize - 4;
  return `<g>
    <rect width="${TAG_W}" height="${TAG_H}" fill="${pal.paper}"/>
    <rect x="1.5" y="1.5" width="${TAG_W - 3}" height="${TAG_H - 3}" fill="none" stroke="${pal.colour}" stroke-width="0.3"/>
    <g transform="translate(${sealX} ${sealY}) scale(${n(sealSize / 200)})" color="${pal.accent}" fill="${pal.accent}">${sealGeom}</g>
    <text x="6" y="10" font-family="ui-serif, Georgia, serif" font-size="6" fill="${pal.colour}">Kalakriti</text>
    <text x="6" y="15" font-family="ui-sans-serif, system-ui, sans-serif" font-size="2.6" fill="${pal.colour}" opacity="0.75">Verified handmade provenance</text>
    <g font-family="ui-sans-serif, system-ui, sans-serif" font-size="3" fill="${pal.colour}">
      <text x="6" y="26">Artisan</text>
      <line x1="6" y1="28" x2="${TAG_W - 6}" y2="28" stroke="${pal.colour}" stroke-width="0.25"/>
      <text x="6" y="35">Craft</text>
      <line x1="6" y1="37" x2="${TAG_W - 24}" y2="37" stroke="${pal.colour}" stroke-width="0.25"/>
    </g>
    <g transform="translate(${qrX} ${qrY}) scale(${n(qrSize / 240)})" color="${pal.colour}" stroke="${pal.colour}" fill="none">${qrGeom}</g>
    <text x="6" y="${TAG_H - 4}" font-family="ui-sans-serif, system-ui, sans-serif" font-size="2.2" fill="${pal.colour}" opacity="0.6">QR filled by backend at print time — scan not yet verified on paper</text>
  </g>`;
}

// 8-up on A4 (210x297mm): 2 cols x 4 rows, margin 10mm, computed gutters.
const PAGE_W = 210, PAGE_H = 297, MARGIN = 10, COLS = 2, ROWS = 4;
const GUTTER_X = (PAGE_W - 2 * MARGIN - COLS * TAG_W) / (COLS - 1);
const GUTTER_Y = (PAGE_H - 2 * MARGIN - ROWS * TAG_H) / (ROWS - 1);

function cropMarks(x, y, colour) {
  const len = 3, off = 1.5; // mm — mark starts 1.5mm outside the trim edge, 3mm long
  const corners = [
    [x, y, -1, -1], [x + TAG_W, y, 1, -1],
    [x, y + TAG_H, -1, 1], [x + TAG_W, y + TAG_H, 1, 1],
  ];
  return corners.map(([cx, cy, dx, dy]) => `
    <path d="M${n(cx + dx * off)} ${n(cy)} L${n(cx + dx * (off + len))} ${n(cy)} M${n(cx)} ${n(cy + dy * off)} L${n(cx)} ${n(cy + dy * (off + len))}" stroke="${colour}" stroke-width="0.15"/>`).join('');
}

function sheet(pal, label) {
  let tags = '', marks = '';
  for (let r = 0; r < ROWS; r++) {
    for (let c = 0; c < COLS; c++) {
      const x = MARGIN + c * (TAG_W + GUTTER_X);
      const y = MARGIN + r * (TAG_H + GUTTER_Y);
      tags += `<g transform="translate(${n(x)} ${n(y)})">${tag(pal)}</g>`;
      marks += cropMarks(x, y, pal.colour);
    }
  }
  return `<?xml version="1.0" encoding="UTF-8"?>
<!-- Kalakriti print — Batch 8. ${label}. A4 sheet (210x297mm), 8 tags,
     90x54mm each, 10mm page margin, corner crop marks for trimming.
     Literal mm on the root svg and explicit hex fills are deliberate here
     only — this file is opened by a print pipeline, not a browser; see
     packages/print/README.md. QR slot geometry and its
     not-yet-scan-verified caveat are reused from patterns/qr-frame.svg. -->
<svg xmlns="http://www.w3.org/2000/svg" width="${PAGE_W}mm" height="${PAGE_H}mm" viewBox="0 0 ${PAGE_W} ${PAGE_H}">
  <rect width="${PAGE_W}" height="${PAGE_H}" fill="${pal.paper}"/>
  ${tags}
  <g>${marks}</g>
</svg>`;
}

writeFileSync(join(OUT, 'tag-a4-colour.svg'), sheet(INK, 'Two-colour variant (ink + terracotta seal)'));
writeFileSync(join(OUT, 'tag-a4-mono.svg'), sheet(MONO, 'Single-colour variant, pure black — for one-colour print processes'));

// ---- checks --------------------------------------------------------------
let failed = false;
const fail = (m) => { console.error('FAIL  ' + m); failed = true; };
for (const f of ['tag-a4-colour.svg', 'tag-a4-mono.svg']) {
  const svg = readFileSync(join(OUT, f), 'utf8');
  if (!/width="210mm" height="297mm"/.test(svg)) fail(f + ': not A4 at 210mm x 297mm');
  if (!/viewBox="0 0 210 297"/.test(svg)) fail(f + ': viewBox does not match the mm dimensions 1:1');
  if ((svg.match(new RegExp('<rect width="' + TAG_W + '" height="' + TAG_H + '"', 'g')) || []).length !== ROWS * COLS)
    fail(f + ': expected ' + (ROWS * COLS) + ' tags');
  if (!svg.includes('scan not yet verified on paper')) fail(f + ': missing the QR unverified-scan caveat');
}
if (MONO.colour !== '#000000' || INK.colour === INK.paper) fail('palette sanity check failed');
console.log(failed ? '\nprint generator: failures above.' : '\nprint generator: all checks passed. 2 A4 sheets written to packages/print/src/.');
if (failed) process.exit(1);
