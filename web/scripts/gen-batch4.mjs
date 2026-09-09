/*
 * Kalakriti assets — Batch 4 generator
 * Micro-icon system: core UI, domain, craft-category icons, the charkha
 * spinner, and the upload-zone graphic.
 *
 * Run:  node scripts/gen-batch4.mjs
 *
 * Every icon is defined as a short list of primitives (line, poly, circ,
 * arc, dot) rather than hand-typed path data, so ~90 icons stay geometrically
 * consistent and cheap to adjust. All icons share one convention: 24x24 grid,
 * 2px safe margin, stroke-width 1.5, round cap/join (icons are UI chrome, not
 * a geometric motif — stroke-geometry-spec.md reserves miter for jaali/
 * block-print/handloom-grid). Filled dots (more-horizontal etc.) are the only
 * fill="currentColor" elements.
 */

import { writeFileSync, readFileSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const SRC = join(ROOT, 'packages/icons/src');
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
// rounded rect, corner radius r, as a single closed stroke path
const rrect = (x, y, w, h, r) =>
  'M' + pt(x + r, y) +
  ' L' + pt(x + w - r, y) + ' A' + n(r) + ' ' + n(r) + ' 0 0 1 ' + pt(x + w, y + r) +
  ' L' + pt(x + w, y + h - r) + ' A' + n(r) + ' ' + n(r) + ' 0 0 1 ' + pt(x + w - r, y + h) +
  ' L' + pt(x + r, y + h) + ' A' + n(r) + ' ' + n(r) + ' 0 0 1 ' + pt(x, y + h - r) +
  ' L' + pt(x, y + r) + ' A' + n(r) + ' ' + n(r) + ' 0 0 1 ' + pt(x + r, y) + ' Z';

const S = (d) => ({ d, fill: 'none', stroke: true });
const F = (d) => ({ d, fill: 'currentColor', stroke: false });

/* ================================================================ CORE UI
 * 61 icons. Conventional minimal-icon silhouettes — this set is UI chrome,
 * not a cultural motif, so shapes read as globally legible iconography
 * rather than drawing on the six licensed traditions.
 */

const CORE = {
  home: () => [S(poly([[4, 12], [12, 5], [20, 12]])), S(poly([[6, 20], [6, 12], [18, 12], [18, 20]], true)), S(line(10, 20, 10, 15) + ' L' + pt(14, 15) + ' L' + pt(14, 20))],
  search: () => [S(circ(10.5, 10.5, 5.5)), S(line(14.7, 14.7, 20, 20))],
  'voice-search': () => [S(rrect(10, 4, 4, 9, 2)), S(line(7, 13, 7, 15) + ' A5 5 0 0 0 17 15 L17 13'), S(line(12, 20, 12, 17))],
  microphone: () => [S(rrect(9.5, 3, 5, 10, 2.5)), S('M6 12 A6 6 0 0 0 18 12'), S(line(12, 18, 12, 21)), S(line(8, 21, 16, 21))],
  camera: () => [S(poly([[3, 8], [8, 8], [9.5, 5], [14.5, 5], [16, 8], [21, 8], [21, 19], [3, 19]], true)), S(circ(12, 13, 3.5))],
  video: () => [S(rrect(3, 6, 12, 12, 1.5)), S(poly([[15, 10], [21, 7], [21, 17], [15, 14]], true))],
  image: () => [S(rrect(3, 4, 18, 16, 1.5)), S(circ(8.5, 9.5, 2)), S(poly([[3, 17], [9, 11], [13, 15], [17, 11], [21, 15]]))],
  upload: () => [S(line(12, 15, 12, 3) + ' L' + pt(8, 7)), S(line(12, 3, 16, 7)), S(poly([[4, 15], [4, 21], [20, 21], [20, 15]]))],
  plus: () => [S(line(12, 4, 12, 20)), S(line(4, 12, 20, 12))],
  edit: () => [S(poly([[5, 19], [5, 15], [16, 4], [20, 8], [9, 19]], true)), S(line(13.5, 6.5, 17.5, 10.5))],
  trash: () => [S(line(4, 7, 20, 7)), S(poly([[6, 7], [7, 21], [17, 21], [18, 7]])), S(line(10, 7, 10, 4) + ' L' + pt(14, 4) + ' L' + pt(14, 7)), S(line(10, 11, 10, 17)), S(line(14, 11, 14, 17))],
  check: () => [S(poly([[4, 13], [9, 18], [20, 6]]))],
  close: () => [S(line(5, 5, 19, 19)), S(line(19, 5, 5, 19))],
  'chevron-up': () => [S(poly([[5, 15], [12, 8], [19, 15]]))],
  'chevron-down': () => [S(poly([[5, 9], [12, 16], [19, 9]]))],
  'chevron-left': () => [S(poly([[15, 5], [8, 12], [15, 19]]))],
  'chevron-right': () => [S(poly([[9, 5], [16, 12], [9, 19]]))],
  'arrow-up': () => [S(line(12, 20, 12, 4)), S(poly([[6, 10], [12, 4], [18, 10]]))],
  'arrow-down': () => [S(line(12, 4, 12, 20)), S(poly([[6, 14], [12, 20], [18, 14]]))],
  'arrow-left': () => [S(line(20, 12, 4, 12)), S(poly([[10, 6], [4, 12], [10, 18]]))],
  'arrow-right': () => [S(line(4, 12, 20, 12)), S(poly([[14, 6], [20, 12], [14, 18]]))],
  filter: () => [S(poly([[4, 5], [20, 5], [14, 13], [14, 19], [10, 21], [10, 13]], true))],
  sort: () => [S(line(8, 4, 8, 20)), S(poly([[5, 7], [8, 4], [11, 7]])), S(line(16, 20, 16, 4)), S(poly([[13, 17], [16, 20], [19, 17]]))],
  share: () => [S(circ(6, 12, 2.6)), S(circ(18, 6, 2.6)), S(circ(18, 18, 2.6)), S(line(8.3, 10.7, 15.7, 7.3)), S(line(8.3, 13.3, 15.7, 16.7))],
  download: () => [S(line(12, 3, 12, 15) + ' L' + pt(8, 11)), S(line(12, 15, 16, 11)), S(poly([[4, 15], [4, 21], [20, 21], [20, 15]]))],
  print: () => [S(rrect(6, 3, 12, 6, 1)), S(poly([[4, 9], [20, 9], [20, 17], [16, 17], [16, 21], [8, 21], [8, 17], [4, 17]], true)), S(line(8, 13, 16, 13))],
  link: () => [S('M10 14 A5 5 0 0 1 10.5 6.5 L14 3 A5 5 0 0 1 21 10 L18.5 12.5'), S('M14 10 A5 5 0 0 1 13.5 17.5 L10 21 A5 5 0 0 1 3 14 L5.5 11.5')],
  'external-link': () => [S(poly([[19, 14], [19, 20], [4, 20], [4, 5], [10, 5]])), S(line(13, 4, 20, 4) + ' L' + pt(20, 11)), S(line(20, 4, 11, 13))],
  settings: () => [S(circ(12, 12, 3.4)), S('M12 4 L12 7 M12 17 L12 20 M4 12 L7 12 M17 12 L20 12 M6.3 6.3 L8.4 8.4 M15.6 15.6 L17.7 17.7 M6.3 17.7 L8.4 15.6 M15.6 8.4 L17.7 6.3')],
  user: () => [S(circ(12, 8, 3.6)), S('M5 20 A7 6.2 0 0 1 19 20')],
  users: () => [S(circ(9, 8, 3.2)), S('M3.5 20 A5.6 5 0 0 1 14.5 20'), S(circ(17, 9, 2.6)), S('M15 15.2 A5 4.4 0 0 1 20.8 19.6')],
  bell: () => [S('M6 16 L6 10 A6 6 0 0 1 18 10 L18 16 L20 19 L4 19 Z'), S('M10 19 A2 2 0 0 0 14 19')],
  message: () => [S(poly([[4, 5], [20, 5], [20, 16], [10, 16], [6, 20], [6, 16], [4, 16]], true))],
  // Delivery box for orders, shipping and delivery tracking.
  package: () => [S(poly([[3.5, 7.27], [12, 2.5], [20.5, 7.27], [20.5, 16.73], [12, 21.5], [3.5, 16.73]], true)), S(poly([[20.5, 7.27], [12, 12], [3.5, 7.27]])), S(line(12, 12, 12, 21.5)), S(line(7.5, 4.5, 16.5, 9.5))],
  // Deliberately not the WhatsApp brand mark (trademarked) — a generic
  // handset-in-bubble glyph standing in for the WhatsApp channel.
  whatsapp: () => [S(poly([[4, 5], [20, 5], [20, 16], [10, 16], [6, 20], [6, 16], [4, 16]], true)), S('M9.5 9 A0.4 0.4 0 0 0 10.3 9 A2.5 3 0 0 0 13 11.7 A0.4 0.4 0 0 0 13 10.9 L11.8 10.3 L9.5 9')],
  calendar: () => [S(rrect(4, 5, 16, 15, 1.5)), S(line(8, 3, 8, 7)), S(line(16, 3, 16, 7)), S(line(4, 10, 20, 10))],
  clock: () => [S(circ(12, 12, 8)), S(line(12, 12, 12, 7)), S(line(12, 12, 15.5, 14))],
  location: () => [S('M12 21 C12 21 5 14 5 9 A7 7 0 0 1 19 9 C19 14 12 21 12 21 Z'), S(circ(12, 9, 2.4))],
  phone: () => [S('M5 4 L9 4 L11 9 L8.5 10.5 A11 11 0 0 0 13.5 15.5 L15 13 L20 15 L20 19 A2 2 0 0 1 18 21 A16 16 0 0 1 3 6 A2 2 0 0 1 5 4 Z')],
  info: () => [S(circ(12, 12, 8)), S(dot(12, 8.2, 0.9)), S(line(12, 11, 12, 16))],
  warning: () => [S(poly([[12, 4], [21, 19], [3, 19]], true)), S(line(12, 10, 12, 14.5)), S(dot(12, 16.6, 0.85))],
  error: () => [S(circ(12, 12, 8)), S(line(9.5, 9.5, 14.5, 14.5)), S(line(14.5, 9.5, 9.5, 14.5))],
  success: () => [S(circ(12, 12, 8)), S(poly([[8.3, 12.3], [11, 15], [15.7, 9.3]]))],
  help: () => [S(circ(12, 12, 8)), S('M9.5 9.6 A2.5 2.5 0 1 1 12.7 13.9 C12.2 14.2 12 14.6 12 15.2 L12 15.6'), S(dot(12, 18, 0.85))],
  menu: () => [S(line(4, 7, 20, 7)), S(line(4, 12, 20, 12)), S(line(4, 17, 20, 17))],
  'more-horizontal': () => [F(dot(6, 12, 1.6)), F(dot(12, 12, 1.6)), F(dot(18, 12, 1.6))],
  'more-vertical': () => [F(dot(12, 6, 1.6)), F(dot(12, 12, 1.6)), F(dot(12, 18, 1.6))],
  eye: () => [S('M3 12 C5.5 6.5 9 5 12 5 C15 5 18.5 6.5 21 12 C18.5 17.5 15 19 12 19 C9 19 5.5 17.5 3 12 Z'), S(circ(12, 12, 3))],
  'eye-off': () => [S('M4 5 L20 19'), S('M9.5 6.3 A11.5 11.5 0 0 1 12 6 C15 6 18.5 7.7 21 12.5 C20.2 14 19.2 15.2 18.2 16.1'), S('M6.8 8.5 C4.9 9.9 3.5 11.7 3 12.5 C5.5 17.3 9 19 12 19 C13.2 19 14.5 18.7 15.7 18'), S(circ(12, 12.5, 3))],
  lock: () => [S(rrect(5, 11, 14, 10, 1.5)), S('M8 11 L8 7 A4 4 0 0 1 16 7 L16 11'), S(dot(12, 16, 1.1))],
  refresh: () => [S(arc(12, 12, 7, -70, 200) + ' L' + pt(9.5, 6) + ' M19 5 L19 9.5 L14.5 9.5')],
  sync: () => [S(arc(9, 9, 5, 20, 250)), S(poly([[13.5, 4], [14, 8], [10, 7.5]])), S(arc(15, 15, 5, 200, 70)), S(poly([[10.5, 20], [10, 16], [14, 16.5]]))],
  offline: () => [S(line(4, 4, 20, 20)), S('M7 9.5 A9 9 0 0 1 15.5 6.8'), S('M4.5 12.5 A11.5 11.5 0 0 1 8.5 9'), S('M9.5 15.5 A5.5 5.5 0 0 1 14 13.7'), S(dot(12, 19, 0.9))],
  'wifi-off': () => [S(line(4, 4, 20, 20)), S('M2.5 8.5 A15 15 0 0 1 9.5 4.8'), S('M14.5 5.5 A15 15 0 0 1 21.5 8.5'), S('M5.5 12 A10.5 10.5 0 0 1 9.5 9.8'), S('M14.5 9.8 A10.5 10.5 0 0 1 16 10.8'), S('M8.5 15.5 A6 6 0 0 1 13 14'), S(dot(12, 19, 0.9))],
  language: () => [S(circ(12, 12, 8)), S('M4 12 L20 12'), S('M12 4 A12 8 0 0 1 12 20 A12 8 0 0 1 12 4 Z'), S('M6 7.5 A13 5 0 0 0 18 7.5'), S('M6 16.5 A13 5 0 0 1 18 16.5')],
  accessibility: () => [S(circ(12, 12, 8)), S(dot(12, 8.3, 1)), S(line(8, 10.8, 16, 10.8)), S(line(12, 10.8, 12, 14) + ' L9.5 18.5'), S(line(12, 14, 14.5, 18.5))],
  'text-size': () => [S(poly([[4, 8], [4, 5], [14, 5], [14, 8]])), S(line(9, 5, 9, 19)), S(line(6.5, 19, 11.5, 19)), S(poly([[15, 11], [15, 9], [21, 9], [21, 11]])), S(line(18, 9, 18, 19)), S(line(16.3, 19, 19.7, 19))],
  contrast: () => [S(circ(12, 12, 8)), F('M12 4 A8 8 0 0 1 12 20 Z')],
  play: () => [S(poly([[7, 4.5], [20, 12], [7, 19.5]], true))],
  pause: () => [S(rrect(6, 4.5, 4.5, 15, 1)), S(rrect(13.5, 4.5, 4.5, 15, 1))],
  volume: () => [S(poly([[3, 9.5], [3, 14.5], [7.5, 14.5], [12.5, 19.5], [12.5, 4.5], [7.5, 9.5]], true)), S(arc(12.5, 12, 5, -45, 45))],
  speaker: () => [S(poly([[3, 9.5], [3, 14.5], [7.5, 14.5], [12.5, 19.5], [12.5, 4.5], [7.5, 9.5]], true)), S(arc(12.5, 12, 3.6, -50, 50)), S(arc(12.5, 12, 6.4, -50, 50))],
};

/* ============================================================ DOMAIN ICONS
 * 14 icons. These are ours — must not read as generic stock iconography.
 */

const DOMAIN = {
  'handmade-certified': () => [S(circ(12, 13, 7)), S(line(9, 11, 9, 15) + ' M12 11 L12 15 M15 11 L15 15 M8 13 L16 13'), S(dot(12, 5, 1))],
  'verified-artisan': () => [S('M12 3 L18.5 5.8 L18.5 12 C18.5 16.5 15.7 19.6 12 21 C8.3 19.6 5.5 16.5 5.5 12 L5.5 5.8 Z'), S(poly([[8.5, 12.2], [10.8, 14.5], [15.5, 9.5]]))],
  'gi-tagged': () => [S('M12 21 C12 21 5.5 14.3 5.5 9.3 A6.5 6.5 0 0 1 18.5 9.3 C18.5 14.3 12 21 12 21 Z'), S(dot(12, 9, 0.9)), S(poly([[8.5, 8], [10, 5], [14, 5], [15.5, 8]]))],
  'made-to-order': () => [S(circ(9.5, 14.5, 6)), S(line(9.5, 14.5, 9.5, 11) + ' L12 13'), S(line(3, 4, 9, 4) + ' M4.5 4 L4.5 8 A5 5 0 0 0 9 10.5'), S(line(21, 4, 15, 4) + ' M19.5 4 L19.5 8 A5 5 0 0 1 15 10.5')],
  'ready-stock': () => [S(poly([[4, 9], [4, 19], [20, 19], [20, 9]])), S(line(4, 9, 20, 9)), S(line(4, 13.5, 20, 13.5)), S(poly([[6, 9], [6, 5], [18, 5], [18, 9]]))],
  'process-video': () => [S(poly([[4, 5], [4, 12.5], [8, 10.5], [8, 5]], true)), S(poly([[10, 5], [10, 12.5], [14, 10.5], [14, 5]], true)), S(poly([[16, 5], [16, 12.5], [20, 10.5], [20, 5]], true)), F(poly([[9.5, 15.5], [9.5, 20.5], [15, 18]], true))],
  provenance: () => [S(poly([[6, 5], [6, 9], [10, 9], [10, 5]], true)), S(poly([[14, 5], [14, 9], [18, 9], [18, 5]], true)), S(poly([[10, 15], [10, 19], [14, 19], [14, 15]], true)), S(line(8, 9, 8, 12) + ' L16 12 L16 9'), S(line(12, 12, 12, 15))],
  'handloom-verified': () => [S(line(4, 5, 4, 19) + ' M8 5 L8 19 M12 5 L12 19'), S(line(4, 8, 12, 8) + ' M4 14 L12 14'), S(line(15, 5, 15, 19) + ' M19 6.5 L15 8.5 L19 10.5 L15 12.5 L19 14.5 L15 16.5 L19 18.5')],
  'collective-order': () => [S(circ(6, 7, 2)), S(circ(18, 7, 2)), S(circ(12, 6, 2)), S(dot(12, 17, 2.2)), S(line(6, 9, 10.5, 15.5) + ' M18 9 L13.5 15.5 M12 8 L12 14.5')],
  'fair-price': () => [S(line(12, 4, 12, 8)), S(line(5, 8, 19, 8)), S(line(5, 8, 3, 13) + ' A2.5 2.2 0 0 0 8 13 L5 8'), S(line(19, 8, 17, 13) + ' A2.5 2.2 0 0 0 22 13 L19 8'), S(line(9, 20, 15, 20)), S(line(12, 8, 12, 20))],
  'income-statement': () => [S(poly([[6, 3], [15, 3], [18, 6], [18, 21], [6, 21]], true)), S(line(6, 7, 18, 7)), S(line(9, 11, 9, 17)), S(poly([[9, 15], [12, 12], [15, 15], [15, 11]]))],
  cluster: () => [S(circ(6, 6, 2.2)), S(circ(18, 6, 2.2)), S(circ(6, 18, 2.2)), S(circ(18, 18, 2.2)), S(circ(12, 12, 2.6)), S(line(7.7, 7.7, 10.2, 10.2) + ' M16.3 7.7 L13.8 10.2 M7.7 16.3 L10.2 13.8 M16.3 16.3 L13.8 13.8')],
  shg: () => [S(circ(12, 5.5, 2.2)), S(circ(19, 12, 2.2)), S(circ(12, 18.5, 2.2)), S(circ(5, 12, 2.2)), S('M12 7.7 A6.8 6.8 0 0 1 17.4 10.3 M17.4 13.7 A6.8 6.8 0 0 1 12 16.3 M7 13.7 A6.8 6.8 0 0 0 12 16.3 M7 10.3 A6.8 6.8 0 0 1 12 7.7')],
  'dying-craft': () => [S(line(6, 5, 6, 19) + ' M10 5 L10 19'), S(line(6, 8, 10, 8) + ' M6 12 L10 12'), S('M15 5 L18 8 M18 5 L14 9'), S('M14.5 12.5 A4 4 0 1 0 20 13.5'), S(line(17, 11, 17, 15) + ' M17 17.3 L17 17.6')],
};

/* ========================================================= CRAFT CATEGORY
 * 12 icons. Each abstracts the MAKING, not the finished object.
 */

const CRAFT = {
  weaving: () => [S(poly([[5, 4], [5, 20], [19, 20], [19, 4]])), S(line(5, 4, 19, 4)), S(line(8, 4, 8, 20) + ' M16 4 L16 20'), S(line(5, 9, 19, 9) + ' M5 15 L19 15')],
  'block-printing': () => [S(poly([[6, 4], [14, 4], [14, 10], [6, 10]], true)), S(line(10, 10, 10, 14)), S(poly([[4, 14], [20, 14], [20, 20], [4, 20]])), S(dot(9, 17, 0.9)), S(dot(15, 17, 0.9))],
  pottery: () => [S(line(4, 19, 20, 19)), S('M9 19 C7 15 7 10 9 6 L15 6 C17 10 17 15 15 19'), S(line(4, 5.5, 20, 5.5))],
  metalwork: () => [S(line(6, 20, 15, 11) + ' L18 8 L20 10 L17 13'), S(line(12, 8, 4, 16) + ' A1.5 1.5 0 0 0 6 18 L12 12'), S(line(4, 4, 8, 8))],
  woodwork: () => [S(poly([[4, 9], [11, 4], [11, 20], [4, 20]], true)), S(line(6, 9, 6, 20) + ' M8.5 9 L8.5 20'), S(line(13, 6, 21, 10) + ' L21 12 L13 8 Z')],
  embroidery: () => [S(circ(12, 12, 8)), S(dot(12, 12, 0.8)), S('M12 12 L12 5 M12 12 L18 8 M12 12 L18 16 M12 12 L12 19 M12 12 L6 16 M12 12 L6 8')],
  painting: () => [S('M6 20 C4 18 4 14 7 13 C9 12.3 10 10 9 8 C11 6.5 14 7.5 14 10 C17 9 19.5 11.5 19 14 L11 20 Z'), S(line(4, 4, 4, 8) + ' M4 4 L8 4')],
  basketry: () => [S('M6 9 L18 9 L16 20 L8 20 Z'), S('M8.5 9 C8.5 9 10 5 12 5 C14 5 15.5 9 15.5 9'), S(line(7.5, 12.5, 16.5, 12.5)), S(line(8, 16, 16, 16))],
  jewellery: () => [S(circ(12, 9, 5)), S(circ(12, 9, 2)), S(poly([[8.5, 13], [6, 20], [10, 20], [12, 15], [14, 20], [18, 20], [15.5, 13]]))],
  leather: () => [S(poly([[5, 6], [19, 6], [17, 19], [7, 19]], true)), S(dot(9, 9.5, 0.6)), S(dot(15, 9.5, 0.6)), S(dot(8.3, 13, 0.6)), S(dot(15.7, 13, 0.6)), S(dot(9, 16.3, 0.6)), S(dot(15, 16.3, 0.6))],
  stone: () => [S(poly([[5, 15], [9, 5], [16, 6], [19, 14], [14, 20], [7, 19]], true)), S(line(9, 5, 12, 13) + ' L' + pt(19, 14)), S(line(12, 13, 7, 19))],
  bamboo: () => [S(line(9, 3, 9, 21)), S(line(15, 3, 15, 21)), S(line(6, 7, 18, 7)), S(line(6, 13, 18, 13)), S(line(9, 3, 15, 3)), S(line(9, 21, 15, 21))],
};

/* ---------------------------------------------------------- categories */
// Auto-generated per-icon comments come from this metadata rather than being
// hand-typed 87 times — keeps the source dense without losing the "which
// tradition, what use" documentation Batch 0 requires.
const CATMETA = {
  core: { dir: 'core', tradition: 'none — global UI iconography, not a licensed motif', use: 'general UI chrome' },
  domain: { dir: 'domain', tradition: 'handloom structure, unless noted per-icon', use: 'product-specific UI states' },
  craft: { dir: 'craft', tradition: 'none — abstracts the making process, not a motif', use: 'craft category navigation and filters' },
};
const DOMAIN_TRADITION = {
  'gi-tagged': 'none — a map-pin/seal fusion, not drawn from a licensed tradition',
  'collective-order': 'none — abstract node convergence',
  cluster: 'kolam-derived node arrangement (rotational layout, not a drawn continuous line)',
  shg: 'Warli-derived — a ring of reduced figures',
  'fair-price': 'none — balance scale with a thread',
};

/* --------------------------------------------------------------- emit */

const icons = []; // { name, svg, category }

function buildIcon(name, category, elements) {
  const body = elements.map((el) => {
    if (el.stroke) {
      return '<path fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" d="' + el.d + '"/>';
    }
    return '<path fill="currentColor" d="' + el.d + '"/>';
  }).join('\n  ');
  const meta = CATMETA[category];
  const tradition = DOMAIN_TRADITION[name] || meta.tradition;
  const comment = category === 'domain'
    ? name + ' (domain icon). Tradition: ' + tradition + '. Use: ' + meta.use + '.'
    : name + ' (' + category + ' icon). ' + tradition + '. Use: ' + meta.use + '.';
  const svg =
    '<?xml version="1.0" encoding="UTF-8"?>\n' +
    '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">\n' +
    '  <!-- ' + comment + ' -->\n' +
    '  ' + body + '\n' +
    '</svg>\n';
  icons.push({ name, svg, category });
}

for (const [name, fn] of Object.entries(CORE)) buildIcon(name, 'core', fn());
for (const [name, fn] of Object.entries(DOMAIN)) buildIcon(name, 'domain', fn());
for (const [name, fn] of Object.entries(CRAFT)) buildIcon(name, 'craft', fn());

for (const icon of icons) writeFileSync(join(SRC, icon.name + '.svg'), icon.svg);

/* ===================================================== CHARKHA SPINNER
 * Wheel-and-spindle silhouette per motifs.md: circle + radiating spokes +
 * a secondary smaller circle (spindle) offset from centre. This is the one
 * licensed use of the charkha — its literal mechanical function maps
 * directly onto a loading indicator.
 */

const wheel = () => {
  const els = [S(circ(11, 12, 8))];
  for (let a = 0; a < 360; a += 45) {
    const x1 = 11 + 3 * Math.cos(rad(a)), y1 = 12 + 3 * Math.sin(rad(a));
    const x2 = 11 + 8 * Math.cos(rad(a)), y2 = 12 + 8 * Math.sin(rad(a));
    els.push(S(line(x1, y1, x2, y2)));
  }
  els.push(S(circ(11, 12, 3)));
  els.push(S(circ(18, 15, 2.2))); // spindle, offset from centre per motifs.md
  els.push(S(line(13.3, 13.5, 16.2, 14.6)));
  return els;
};

buildIcon('charkha-spinner', 'core', wheel());
// Re-tag the auto comment: this one is licensed (motifs.md), not "none".
icons[icons.length - 1].svg = icons[icons.length - 1].svg.replace(
  '<!-- charkha-spinner (core icon). none — global UI iconography, not a licensed motif. Use: general UI chrome. -->',
  '<!-- Charkha (spinning wheel), reduced to wheel + radiating spokes + an offset spindle per motifs.md. Primary ML-processing / loading indicator. Rotation and the reduced-motion pulse are CSS in icons.css, not SMIL. -->'
);
writeFileSync(join(SRC, 'charkha-spinner.svg'), icons[icons.length - 1].svg);

/* ==================================================== UPLOAD ZONE GRAPHIC
 * One asset, four states via a `data-state` attribute the host toggles.
 * icons.css shows/hides each state's group; nothing here is state-specific
 * markup duplication beyond the four small icon groups themselves.
 */

{
  const dashedBox = rrect(3, 3, 18, 18, 2);
  const idleIcon = [S(line(12, 15, 12, 7) + ' L' + pt(9, 10)), S(line(12, 7, 15, 10))];
  const dragIcon = [S(line(12, 16, 12, 6) + ' L' + pt(8, 10)), S(line(12, 6, 16, 10))];
  const uploadingIcon = [S(arc(12, 11, 4, -70, 200))];
  const successIcon = [S(circ(12, 11, 5)), S(poly([[9.2, 11.2], [11, 13], [14.8, 8.8]]))];
  const errorIcon = [S(circ(12, 11, 5)), S(line(10, 9, 14, 13)), S(line(14, 9, 10, 13))];

  const grp = (state, els) => '<g data-upload-state="' + state + '">\n    ' +
    els.map((el) => el.stroke
      ? '<path fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" d="' + el.d + '"/>'
      : '<path fill="currentColor" d="' + el.d + '"/>').join('\n    ') +
    '\n  </g>';

  const svg =
    '<?xml version="1.0" encoding="UTF-8"?>\n' +
    '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">\n' +
    '  <!-- Upload-zone drop area, one asset with four states (idle, drag-over, uploading, success, error) toggled by the k-upload-zone[data-state] CSS in icons.css, which shows exactly one <g> at a time. The dashed border is CSS (border-style:dashed) on the host element, not drawn here. -->\n' +
    '  ' + grp('idle', idleIcon) + '\n' +
    '  ' + grp('drag', dragIcon) + '\n' +
    '  ' + grp('uploading', uploadingIcon) + '\n' +
    '  ' + grp('success', successIcon) + '\n' +
    '  ' + grp('error', errorIcon) + '\n' +
    '</svg>\n';
  writeFileSync(join(SRC, 'upload-zone.svg'), svg);
  icons.push({ name: 'upload-zone', svg, category: 'core', isMultiState: true });
}

/* ============================================================ manifest,
 * typed names, and the name->component map. Direct `.svg` imports (per
 * icon) stay the tree-shaking path; `icons.js` + <Icon name> is the
 * convenience path and is expected to pull in the whole set — Batch 8
 * proves the tree-shaking number, this batch just keeps the two paths
 * from being the same entry point.
 */

const allNames = icons.map((i) => i.name);

writeFileSync(join(ROOT, 'packages/icons/manifest.js'),
  '/* Generated by scripts/gen-batch4.mjs — do not hand-edit.\n' +
  '   Icon manifest for the documentation site and the validation script. */\n\n' +
  'export const ICONS = ' + JSON.stringify(
    icons.map((i) => ({ name: i.name, category: i.category, file: 'src/' + i.name + '.svg' })),
    null, 2
  ) + ';\n');

writeFileSync(join(ROOT, 'packages/icons/icons.d.ts'),
  '// Generated by scripts/gen-batch4.mjs — do not hand-edit.\n' +
  '// A name outside this union is a compile error at every <Icon name="..."> call site.\n' +
  'export type IconName =\n  | ' + allNames.map((n2) => "'" + n2 + "'").join('\n  | ') + ';\n');

writeFileSync(join(ROOT, 'packages/icons/icons.js'),
  '/* Generated by scripts/gen-batch4.mjs — do not hand-edit.\n' +
  '   Name -> component map for <Icon name="...">. Importing this file pulls in\n' +
  '   every icon; import a single ./src/<name>.svg directly when tree-shaking\n' +
  '   a specific icon matters (see README). */\n\n' +
  allNames.map((n2) => "import Icon_" + n2.replace(/-/g, '_') + " from './src/" + n2 + ".svg';").join('\n') + '\n\n' +
  'export const ICON_COMPONENTS = {\n' +
  allNames.map((n2) => "  '" + n2 + "': Icon_" + n2.replace(/-/g, '_') + ',').join('\n') + '\n' +
  '};\n');

/* ------------------------------------------------------------- sprite */
// <symbol> ids ARE referenced (by <use>), so they are exempt from the
// "no unreferenced id" check the same way patterns.svg was in Batch 3.

const symbolBodies = icons.filter((i) => !i.isMultiState).map((i) =>
  '    <symbol id="k-icon-' + i.name + '" viewBox="0 0 24 24">\n' +
  i.svg.replace(/[\s\S]*?<svg[^>]*>\n/, '').replace(/<\/svg>\s*$/, '').split('\n')
    .filter((l) => !l.trim().startsWith('<!--'))
    .map((l) => '      ' + l.trim()).filter(Boolean).join('\n') +
  '\n    </symbol>').join('\n');

writeFileSync(join(ROOT, 'packages/icons/icons-sprite.svg'),
  '<?xml version="1.0" encoding="UTF-8"?>\n' +
  '<!-- Kalakriti icon sprite, Batch 4. Optional — for the buyer app where icon\n' +
  '     count on one page is high enough that N inline <svg> costs more than one\n' +
  '     sprite fetch. Inline once, then <svg class="k-icon"><use href="#k-icon-home"/></svg>. -->\n' +
  '<svg xmlns="http://www.w3.org/2000/svg" width="0" height="0" style="position:absolute" aria-hidden="true" focusable="false">\n' +
  '  <defs>\n' + symbolBodies + '\n  </defs>\n</svg>\n');

/* ============================================================== CHECKS */

let failed = 0;
const fail = (m) => { console.error('FAIL  ' + m); failed++; };
const ok = (m) => console.log('ok    ' + m);

// endpoint-based path bbox: parses M/L points exactly, A endpoints only
// (arc bulge is not captured — acceptable for a margin check, since every
// arc here has a modest radius drawn from a centre already inside the
// safe area; the real arbiter is the 16px contact-sheet render).
function endpoints(d) {
  const pts = [];
  const cmds = d.match(/[MLA][^MLAZ]*/g) || [];
  for (const c of cmds) {
    const type = c[0];
    const nums = c.slice(1).trim().split(/[\s,]+/).filter(Boolean).map(Number);
    if (type === 'A') {
      for (let i = 0; i + 6 < nums.length; i += 7) pts.push([nums[i + 5], nums[i + 6]]);
    } else {
      for (let i = 0; i + 1 < nums.length; i += 2) pts.push([nums[i], nums[i + 1]]);
    }
  }
  return pts;
}

{
  const uz = icons.find((i) => i.name === 'upload-zone');
  const geom = uz.svg.replace(/<!--[\s\S]*?-->/g, '');
  if (!/viewBox="0 0 24 24"/.test(geom)) fail('upload-zone.svg: viewBox is not exactly "0 0 24 24"');
  if (/#[0-9a-fA-F]{3,6}\b/.test(geom)) fail('upload-zone.svg: bare hex colour');
  if (/<animate/.test(geom)) fail('upload-zone.svg: SMIL <animate> — CSS only');
  const states = (geom.match(/data-upload-state="(\w+)"/g) || []).length;
  if (states !== 5) fail('upload-zone.svg: expected 5 states, found ' + states);
  else ok('upload-zone.svg: 5 states, viewBox, no SMIL, no bare hex');
}

for (const icon of icons) {
  if (icon.isMultiState) continue; // checked above, separately, for its own shape
  const geom = icon.svg.replace(/<!--[\s\S]*?-->/g, '');
  if (!/viewBox="0 0 24 24"/.test(geom)) fail(icon.name + '.svg: viewBox is not exactly "0 0 24 24"');
  if (/<svg[^>]*\swidth=/.test(geom)) fail(icon.name + '.svg: hardcoded width on root svg');
  if (/#[0-9a-fA-F]{3,6}\b/.test(geom)) fail(icon.name + '.svg: bare hex colour');
  if (/\bid=/.test(geom)) fail(icon.name + '.svg: unreferenced id');
  if (/<animate/.test(geom)) fail(icon.name + '.svg: SMIL <animate> — CSS only, per Batch 4 brief');
  const strokePaths = geom.match(/<path[^>]*stroke="currentColor"[^>]*>/g) || [];
  for (const sp of strokePaths) {
    if (!/stroke-width="1\.5"/.test(sp)) fail(icon.name + '.svg: stroke-width is not the literal 1.5');
    if (!/stroke-linecap="round"/.test(sp) || !/stroke-linejoin="round"/.test(sp)) fail(icon.name + '.svg: not round cap/join');
  }
  for (const d of geom.match(/d="([^"]+)"/g) || []) {
    const dv = d.slice(3, -1);
    for (const [x, y] of endpoints(dv)) {
      if (x < 1.5 || x > 22.5 || y < 1.5 || y > 22.5)
        fail(icon.name + '.svg: point (' + x + ',' + y + ') outside the 20x20 safe area');
    }
  }
  const bytes = Buffer.byteLength(icon.svg, 'utf8');
  if (bytes > 2048) fail(icon.name + '.svg: ' + bytes + 'B over the 2KB icon budget');
}
if (!failed) ok(allNames.length + ' icons: viewBox / stroke-width / round joins / safe-area / size budget / no SMIL');

const dupCheck = new Set();
for (const n2 of allNames) { if (dupCheck.has(n2)) fail('duplicate icon name: ' + n2); dupCheck.add(n2); }
ok('no duplicate icon names');

console.log('\n' + allNames.length + ' icons + charkha-spinner + upload-zone -> packages/icons/src');
console.log('  manifest.js, icons.d.ts, icons.js, icons-sprite.svg written');

if (failed) { console.error('\n' + failed + ' check(s) failed'); process.exit(1); }
console.log('\nall checks passed');

/* ------------------------------------------------------- contact sheet
 * Fully generated (not template-injected, unlike Batch 3) — at ~90 icons,
 * a hand-maintained template with 90 injection points is worse than just
 * emitting the HTML from the same data the SVGs came from.
 */

const innerBody = (svg) => svg
  .replace(/[\s\S]*?<svg[^>]*>\n/, '')
  .replace(/<\/svg>\s*$/, '')
  .replace(/<!--[\s\S]*?-->\n?/g, '')
  .trim();

const iconSvgTag = (icon, size) =>
  '<svg viewBox="0 0 24 24" style="width:' + size + 'px;height:' + size + 'px" aria-hidden="true" focusable="false">' +
  innerBody(icon.svg) + '</svg>';

const denseGrid = icons.filter((i) => !i.isMultiState).map((i) =>
  '<div class="dense-cell" title="' + i.name + '">' + iconSvgTag(i, 16) + '</div>').join('');

const categoryBlock = (label, tradition, list) =>
  '<h2>' + label + '</h2>' +
  (tradition ? '<p class="note">' + tradition + '</p>' : '') +
  '<div class="icon-grid">' +
  list.map((i) =>
    '<div class="icon-card">' +
    '<div class="icon-card__sizes">' +
    [16, 20, 24, 32].map((s) => iconSvgTag(i, s)).join('') +
    '</div><div class="icon-card__name">' + i.name + '</div></div>'
  ).join('') + '</div>';

const coreIcons = icons.filter((i) => i.category === 'core' && !i.isMultiState && i.name !== 'charkha-spinner');
const domainIcons = icons.filter((i) => i.category === 'domain');
const craftIcons = icons.filter((i) => i.category === 'craft');

// A handful of representative icons rendered through the REAL k-icon /
// k-icon--dense / k-icon--bold classes (every other icon on this sheet is
// emitted as a bare <svg style="width:...px"> for a clean size grid, which
// never exercises icons.css's weight-swap machinery at all — this is the
// one place on the sheet that does, and it's what a Batch 8 audit caught
// wasn't actually working: fixed in icons.css, see the README).
const weightSample = ['home', 'search', 'trash', 'gi-tagged']
  .map((name) => icons.find((i) => i.name === name))
  .filter(Boolean);
const weightRow = (mod, label) => `
    <div class="weight-demo">
      <div class="weight-demo__icons">
        ${weightSample.map((i) => `<svg class="k-icon${mod ? ' k-icon--' + mod : ''}" viewBox="0 0 24 24" style="width:28px;height:28px">${innerBody(i.svg)}</svg>`).join('')}
      </div>
      <p class="tradition">${label}</p>
    </div>`;

const uploadStates = ['idle', 'drag', 'uploading', 'success', 'error'];
const uploadZoneSvg = icons.find((i) => i.name === 'upload-zone').svg;
const uploadZoneInner = innerBody(uploadZoneSvg);

const themeSection = (theme, label) => `
<section class="theme" data-theme="${theme}">
  <header><h1>Batch 4 — micro-icon system</h1><span>${label}</span></header>

  <h2>1 · Dense grid at 16px — every icon, no labels</h2>
  <p class="note">This view is where stroke-weight inconsistency becomes obvious. Scan for anything heavier, lighter, or blurrier than its neighbours.</p>
  <div class="dense-grid">${denseGrid}</div>

  <h2>1.5 · Stroke-weight variants — dense / default / bold, same paths</h2>
  <p class="note">Rendered through the real k-icon classes (icons.css), not the fixed-size grid style used everywhere else on this sheet.</p>
  <div class="weight-row">
    ${weightRow('dense', 'k-icon--dense — 1.25')}
    ${weightRow('', 'default — 1.5')}
    ${weightRow('bold', 'k-icon--bold — 2')}
  </div>

  ${categoryBlock('2 · Core UI icons — 16 / 20 / 24 / 32px', '', coreIcons)}
  ${categoryBlock('3 · Domain icons — ours, must not read as stock', '', domainIcons)}
  ${categoryBlock('4 · Craft category icons — abstracts the making, not the object', '', craftIcons)}

  <h2>5 · Charkha spinner</h2>
  <div class="spinner-row">
    <div class="spinner-demo">
      <svg class="k-icon k-spinner k-spinner--indeterminate" viewBox="0 0 24 24" style="width:48px;height:48px">${innerBody(icons.find(i=>i.name==='charkha-spinner').svg)}</svg>
      <p class="tradition">indeterminate — CSS rotation</p>
    </div>
    <div class="spinner-demo">
      <svg class="k-icon k-spinner k-spinner--determinate" viewBox="0 0 24 24" style="width:48px;height:48px;--k-spin-progress:0.35">${innerBody(icons.find(i=>i.name==='charkha-spinner').svg)}</svg>
      <p class="tradition">determinate, 35% — thread winding onto the spindle</p>
    </div>
    <div class="spinner-demo reduced-motion-demo">
      <svg class="k-icon k-spinner k-spinner--indeterminate" viewBox="0 0 24 24" style="width:48px;height:48px">${innerBody(icons.find(i=>i.name==='charkha-spinner').svg)}</svg>
      <p class="tradition">reduced-motion forced (opacity pulse, no rotation)</p>
    </div>
  </div>

  <h2>6 · Upload zone — five states, one asset</h2>
  <div class="upload-row">
    ${uploadStates.map((s) => `
    <div class="k-upload-zone" data-state="${s}">
      <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">${uploadZoneInner}</svg>
      <div class="k-upload-zone__label">${s}</div>
    </div>`).join('')}
  </div>
</section>`;

const sheet = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Kalakriti — Batch 4 contact sheet: micro-icon system</title>
<link rel="stylesheet" href="../tokens/src/palette.css">
<link rel="stylesheet" href="./icons.css">
<style>
  body { margin: 0; font: 14px/1.5 ui-sans-serif, system-ui, sans-serif; }
  .theme { background: var(--k-surface-base); color: var(--k-text-primary); padding: 2.5rem clamp(1rem,4vw,3rem) 3.5rem; }
  .theme > header { display: flex; align-items: baseline; gap: 0.75rem; margin: 0 0 2rem; padding-bottom: 0.6rem; border-bottom: 1px solid var(--k-border-hairline); }
  .theme > header h1 { font-size: 1.1rem; margin: 0; }
  .theme > header span { color: var(--k-text-secondary); font-size: 0.8rem; }
  h2 { font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.09em; color: var(--k-text-secondary); margin: 2.5rem 0 0.9rem; font-weight: 600; }
  h2:first-of-type { margin-top: 0; }
  .note { font-size: 0.75rem; color: var(--k-text-secondary); max-width: 62ch; margin: 0.3rem 0 1rem; }
  .dense-grid { display: flex; flex-wrap: wrap; gap: 10px; padding: 1rem; border: 1px solid var(--k-border-hairline); background: var(--k-surface-raised); }
  .dense-cell { width: 16px; height: 16px; color: var(--k-text-primary); }
  .icon-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(120px, 1fr)); gap: 0.75rem; }
  .icon-card { border: 1px solid var(--k-border-hairline); background: var(--k-surface-raised); padding: 0.75rem; text-align: center; }
  .icon-card__sizes { display: flex; align-items: center; justify-content: center; gap: 0.5rem; color: var(--k-text-primary); }
  .icon-card__name { margin-top: 0.5rem; font-size: 0.68rem; color: var(--k-text-secondary); font-family: ui-monospace, monospace; }
  .spinner-row, .upload-row, .weight-row { display: flex; flex-wrap: wrap; gap: 1.5rem; }
  .weight-demo { text-align: center; }
  .weight-demo__icons { display: flex; gap: 0.6rem; padding: 0.75rem; border: 1px solid var(--k-border-hairline); background: var(--k-surface-raised); color: var(--k-text-primary); }
  .spinner-demo { text-align: center; color: var(--k-accent-primary-text); }
  .tradition { font-size: 0.68rem; color: var(--k-text-secondary); font-style: italic; max-width: 12rem; }
  .reduced-motion-demo svg { animation: k-spin-pulse 1.6s ease-in-out infinite !important; transform: none !important; }
  .upload-row .k-upload-zone { width: 130px; padding: 1.25rem 0.75rem; }
</style>
</head>
<body>
${themeSection('light', 'light theme')}
${themeSection('dark', 'dark theme')}
${themeSection('high-contrast', 'high-contrast theme')}
</body>
</html>
`;

writeFileSync(join(ROOT, 'packages/icons/contact-sheet-batch4.html'), sheet);
console.log('  contact-sheet-batch4.html written');
