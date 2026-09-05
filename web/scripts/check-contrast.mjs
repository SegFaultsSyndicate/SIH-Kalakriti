// web/scripts/check-contrast.mjs
//
// palette.css writes a measured contrast ratio in a comment beside every
// text-on-surface pair. A comment cannot fail a build, so the numbers drift
// the first time a token moves. This recomputes every pair from the actual
// hex values, in all three themes, and fails if one is below its threshold or
// if the comment no longer matches what the colours really give.
//
// WCAG 2.1: 4.5:1 for body text, 3:1 for large text and non-text UI
// boundaries. Thresholds are per-pair below, because a focus ring and a body
// paragraph are not held to the same bar.

import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const css = readFileSync(
  fileURLToPath(new URL('../packages/tokens/src/palette.css', import.meta.url)),
  'utf8',
);

/** Pairs the UI actually renders, with the ratio each must clear. */
const PAIRS = [
  ['--k-text-primary', '--k-surface-base', 4.5],
  ['--k-text-secondary', '--k-surface-base', 4.5],
  ['--k-text-primary', '--k-surface-raised', 4.5],
  ['--k-text-primary', '--k-surface-sunken', 4.5],
  ['--k-accent-primary-text', '--k-surface-base', 4.5],
  ['--k-accent-secondary', '--k-surface-base', 4.5],
  ['--k-accent-danger', '--k-surface-base', 4.5],
  ['--k-accent-success', '--k-surface-base', 4.5],
  ['--k-text-on-accent', '--k-accent-primary-bg', 4.5],
  ['--k-accent-warning-text', '--k-accent-warning-bg', 4.5],
  ['--k-border-interactive', '--k-surface-base', 3],
  ['--k-focus-ring', '--k-surface-base', 3],
  ['--k-text-on-inverse', '--k-surface-inverse', 4.5],
];

const THEMES = {
  light: [':root', "[data-theme='light']", '[data-theme="light"]'],
  dark: ["[data-theme='dark']", '[data-theme="dark"]'],
  'high-contrast': ["[data-theme='high-contrast']", '[data-theme="high-contrast"]'],
};

/**
 * Collect `--name: value` declarations from every rule block whose selector
 * list contains one of `selectors`. Later blocks win, which is the cascade.
 */
function declarations(selectors) {
  const out = {};
  const blockPattern = /([^{}]+)\{([^{}]*)\}/g;
  let block;
  while ((block = blockPattern.exec(css)) !== null) {
    const selector = block[1];
    if (!selectors.some((s) => selector.includes(s))) continue;
    for (const line of block[2].split(';')) {
      const decl = line.match(/(--[\w-]+)\s*:\s*([^;]+)/);
      if (decl) out[decl[1]] = decl[2].trim();
    }
  }
  return out;
}

const raw = declarations([':root']);

/** Resolve a value through however many var() hops it takes to reach a hex. */
function resolve(value, scope, depth = 0) {
  if (depth > 10) throw new Error(`var() cycle at ${value}`);
  const ref = value.match(/var\(\s*(--[\w-]+)/);
  if (!ref) return value.trim();
  const next = scope[ref[1]] ?? raw[ref[1]];
  if (!next) throw new Error(`unresolved ${ref[1]}`);
  return resolve(next, scope, depth + 1);
}

function channel(v) {
  const c = v / 255;
  return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
}

function luminance(hex) {
  const m = hex.match(/^#([0-9a-f]{6})$/i);
  if (!m) throw new Error(`not a 6-digit hex: ${hex}`);
  const n = parseInt(m[1], 16);
  return (
    0.2126 * channel((n >> 16) & 255) +
    0.7152 * channel((n >> 8) & 255) +
    0.0722 * channel(n & 255)
  );
}

function ratio(fg, bg) {
  const a = luminance(fg);
  const b = luminance(bg);
  return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
}

/**
 * The ratio written in the comment on the line that declares `token`, but
 * ONLY when that comment names this pair's background. A token carries one
 * comment and is rendered on several grounds -- `--k-text-primary` documents
 * itself "on surface-base" and is also painted on surface-raised, and
 * comparing the two is a bug in the checker, not in the palette.
 */
function documented(token, background, selectors) {
  const blockPattern = /([^{}]+)\{([^{}]*)\}/g;
  let block;
  let found;
  while ((block = blockPattern.exec(css)) !== null) {
    if (!selectors.some((s) => block[1].includes(s))) continue;
    const line = block[2]
      .split('\n')
      .find((l) => l.includes(token + ':') || l.trimStart().startsWith(token));
    const name = background.replace('--k-', '');
    const m = line && line.match(new RegExp('on ' + name + ':\\s*([\\d.]+):1'));
    if (m) found = Number(m[1]);
  }
  return found;
}

let failed = 0;
let checked = 0;

for (const [theme, selectors] of Object.entries(THEMES)) {
  const scope = declarations(selectors);
  console.log(`\n  ${theme}`);
  for (const [fgVar, bgVar, min] of PAIRS) {
    if (!(fgVar in scope) || !(bgVar in scope)) continue;
    const fg = resolve(scope[fgVar], scope);
    const bg = resolve(scope[bgVar], scope);
    const r = ratio(fg, bg);
    checked += 1;

    const label = `${fgVar.replace('--k-', '')} on ${bgVar.replace('--k-', '')}`;
    const line = `    ${label.padEnd(44)} ${r.toFixed(2).padStart(6)}:1  needs ${min}`;

    if (r < min) {
      console.error(line + '   FAIL');
      failed += 1;
      continue;
    }

    // The comment is documentation the design review reads. Allow 0.1 for
    // rounding in whatever produced the original number.
    const claimed = documented(fgVar, bgVar, selectors);
    if (claimed !== undefined && Math.abs(claimed - r) > 0.1) {
      console.error(`${line}   COMMENT SAYS ${claimed}:1`);
      failed += 1;
      continue;
    }
    console.log(line);
  }
}

if (failed > 0) {
  console.error(`\ncontrast: ${failed} of ${checked} pairs wrong. Fix palette.css.\n`);
  process.exit(1);
}
console.log(`\ncontrast: ${checked} pairs pass, and every documented ratio is correct.\n`);
