// web/scripts/hex-to-token.mjs
//
// Batch 2/3 say a component may not carry a raw colour: not a hex literal, and
// not a raw palette step either -- only a semantic alias, so the light / dark /
// high-contrast themes can swap underneath it. App components accumulated ~1000
// hex literals anyway.
//
// This rewrites hex literals INSIDE <style> blocks only. Markup, strings, i18n
// catalogues and inline style: props are never touched -- a hex in those is
// either data or a case that needs a human.
//
// Each literal maps to its nearest palette step by CIE76 distance in Lab, then
// to the semantic alias that points at that step in the light theme. Anything
// further away than MAX_DELTA from every step is left alone and reported: that
// is a colour this palette has no answer for, and guessing would move it.
//
// Usage: node scripts/hex-to-token.mjs [--write] [glob-root]

import { readFileSync, writeFileSync } from 'node:fs';
import { readdirSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

const WRITE = process.argv.includes('--write');
const ROOT = process.argv.find((a) => !a.startsWith('--') && a.endsWith('apps')) ?? 'apps';
// CIE76. The default is "same colour, different rounding". --loose also pulls
// in colours that belong to a palette family but were authored off it (the
// Tailwind/Material ambers and greens that leaked in), accepting a visible
// shift onto the nearest craft dye. Run the contrast gate after --loose.
const MAX_DELTA = process.argv.includes('--loose') ? 55 : 12;

// The national flag's saffron and green are not palette colours and must not be
// pulled onto one -- they are the tricolour, reproduced exactly or not at all.
const KEEP = new Set(['#FF9933', '#138808']);

// ---------------------------------------------------------------- palette

const paletteSrc = readFileSync('packages/tokens/src/palette.css', 'utf8');

const rootBlock = paletteSrc.slice(paletteSrc.indexOf(':root'), paletteSrc.indexOf('\n}'));
/** @type {Map<string, string>} step name -> hex */
const steps = new Map();
for (const m of rootBlock.matchAll(/(--k-[a-z]+-\d+):\s*(#[0-9A-Fa-f]{6})/g)) {
  steps.set(m[1], m[2].toUpperCase());
}

// Light theme is the canonical mapping: an alias defined there tells us which
// semantic role a given raw step plays.
const lightStart = paletteSrc.indexOf('[data-theme="light"]');
const lightBlock = paletteSrc.slice(lightStart, paletteSrc.indexOf('\n}', lightStart));
/** @type {Map<string, string>} step name -> first semantic alias using it */
const aliasForStep = new Map();
for (const m of lightBlock.matchAll(/(--k-[a-z-]+):\s*var\((--k-[a-z]+-\d+)\)/g)) {
  if (!aliasForStep.has(m[2])) aliasForStep.set(m[2], m[1]);
}

// ---------------------------------------------------------------- colour math

function hexToRgb(hex) {
  const h = hex.length === 4 ? hex[1] + hex[1] + hex[2] + hex[2] + hex[3] + hex[3] : hex.slice(1);
  return [
    parseInt(h.slice(0, 2), 16),
    parseInt(h.slice(2, 4), 16),
    parseInt(h.slice(4, 6), 16),
  ];
}

function rgbToLab([r, g, b]) {
  const f = (v) => {
    v /= 255;
    return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
  };
  const [R, G, B] = [f(r), f(g), f(b)];
  // sRGB D65 -> XYZ, then XYZ -> Lab
  const x = (R * 0.4124 + G * 0.3576 + B * 0.1805) / 0.9505;
  const y = R * 0.2126 + G * 0.7152 + B * 0.0722;
  const z = (R * 0.0193 + G * 0.1192 + B * 0.9505) / 1.089;
  const k = (t) => (t > 0.008856 ? Math.cbrt(t) : 7.787 * t + 16 / 116);
  const [fx, fy, fz] = [k(x), k(y), k(z)];
  return [116 * fy - 16, 500 * (fx - fy), 200 * (fy - fz)];
}

const stepLabs = [...steps].map(([name, hex]) => ({ name, hex, lab: rgbToLab(hexToRgb(hex)) }));

function nearestStep(hex) {
  const lab = rgbToLab(hexToRgb(hex));
  let best = null;
  let bestD = Infinity;
  for (const s of stepLabs) {
    const d = Math.hypot(lab[0] - s.lab[0], lab[1] - s.lab[1], lab[2] - s.lab[2]);
    if (d < bestD) {
      bestD = d;
      best = s;
    }
  }
  return { step: best, delta: bestD };
}

// A step can back several aliases (ink-900 is both --k-text-primary and, in
// another theme, a surface). Picking by name alone would turn a dark panel
// background into --k-text-primary, which reads identically in light mode and
// inverts to near-white in dark mode. So the CSS property decides the role.
/** @type {Map<string, string[]>} step -> every light-theme alias pointing at it */
const aliasesForStep = new Map();
for (const m of lightBlock.matchAll(/(--k-[a-z-]+):\s*var\((--k-[a-z]+-\d+)\)/g)) {
  if (!aliasesForStep.has(m[2])) aliasesForStep.set(m[2], []);
  aliasesForStep.get(m[2]).push(m[1]);
}

function roleOf(prop) {
  if (/background/.test(prop)) return 'surface';
  if (/^border|outline|^box-shadow|^text-decoration-color|fill|stroke/.test(prop)) return 'border';
  if (/color$/.test(prop) || prop === 'color') return 'text';
  return 'any';
}

const ROLE_MATCH = {
  surface: (a) => /^--k-(surface|accent-[a-z]+-bg)/.test(a),
  // An accent alias is a text colour unless its name says otherwise:
  // --k-accent-danger is what you paint danger text with. Only the -bg
  // names are surfaces, and --k-border-* is matched by the border role.
  text: (a) => /^--k-text/.test(a) || (/^--k-accent-/.test(a) && !/-bg$/.test(a)),
  border: (a) => /^--k-border/.test(a),
  any: () => true,
};

/** The token a hex should become, given the property it sits on. */
function tokenFor(hex, prop = '') {
  if (KEEP.has(hex.toUpperCase())) return { token: null, delta: 0, step: 'kept' };
  const { step, delta } = nearestStep(hex);
  if (delta > MAX_DELTA) return { token: null, delta, step: step.name };
  const candidates = aliasesForStep.get(step.name) ?? [];
  const role = roleOf(prop);
  // No cross-role fallback. An alias from the wrong role reads correctly in the
  // light theme and then inverts: a `color` given --k-surface-raised turns dark
  // text light exactly where the background also went dark. Better a raw step,
  // which is at least theme-static, and counted honestly against gap G1.
  const alias = candidates.find(ROLE_MATCH[role]);
  return { token: `var(${alias ?? step.name})`, delta, step: step.name, aliased: Boolean(alias) };
}

// ---------------------------------------------------------------- walk

function* svelteFiles(dir) {
  for (const entry of readdirSync(dir)) {
    if (entry === 'node_modules' || entry === '.svelte-kit' || entry === 'build') continue;
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) yield* svelteFiles(full);
    else if (entry.endsWith('.svelte')) yield full;
  }
}

const HEX = /#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})\b/g;

let converted = 0;
const unmapped = new Map(); // hex -> count
const unaliasedSteps = new Map(); // raw step -> count, no semantic alias for its role
let stepCount = 0;
const perFile = [];

for (const file of svelteFiles(ROOT)) {
  const src = readFileSync(file, 'utf8');
  const open = src.lastIndexOf('<style');
  if (open === -1) continue;
  const bodyStart = src.indexOf('>', open) + 1;
  const close = src.lastIndexOf('</style>');
  if (close <= bodyStart) continue;

  const head = src.slice(0, bodyStart);
  const style = src.slice(bodyStart, close);
  const tail = src.slice(close);

  let fileCount = 0;
  // Rewrite per declaration so the property name is in hand when choosing the
  // alias. Anything outside a `prop: value;` shape keeps its literal.
  const next = style.replace(/([a-z-]+)\s*:\s*([^;{}]*)/g, (decl, prop, value) => {
    if (!HEX.test(value)) return decl;
    HEX.lastIndex = 0;
    const rewritten = value.replace(HEX, (hex) => {
      const { token } = tokenFor(hex, prop);
      if (!token) {
        unmapped.set(hex.toUpperCase(), (unmapped.get(hex.toUpperCase()) ?? 0) + 1);
        return hex;
      }
      fileCount++;
      return token;
    });
    return decl.replace(value, rewritten);
  });

  // Second pass: a raw palette step is as banned as a hex -- it pins the
  // component to one theme's value. Swap it for the alias that plays its role.
  // A step with no alias (illustration mid-tones, chart series) has no semantic
  // equivalent and is left as-is.
  const next2 = next.replace(/([a-z-]+)\s*:\s*([^;{}]*)/g, (decl, prop, value) => {
    if (!/var\(--k-[a-z]+-\d+\)/.test(value)) return decl;
    const rewritten = value.replace(/var\((--k-[a-z]+-\d+)\)/g, (whole, step) => {
      // Only colour families. --k-space-4 and friends are scale tokens and
      // are exactly what a component is supposed to reference.
      if (!steps.has(step)) return whole;
      const candidates = aliasesForStep.get(step) ?? [];
      const alias = candidates.find(ROLE_MATCH[roleOf(prop)]);
      if (!alias) {
        unaliasedSteps.set(step, (unaliasedSteps.get(step) ?? 0) + 1);
        return whole;
      }
      stepCount++;
      return `var(${alias})`;
    });
    return decl.replace(value, rewritten);
  });

  if (fileCount > 0 || next2 !== next) {
    converted += fileCount;
    perFile.push([relative('.', file).replace(/\\/g, '/'), fileCount]);
    if (WRITE) writeFileSync(file, head + next2 + tail);
  }
}

perFile.sort((a, b) => b[1] - a[1]);
for (const [f, n] of perFile) console.log(`${String(n).padStart(4)}  ${f}`);
console.log(`\n${converted} hex literals ${WRITE ? 'converted' : 'convertible'} across ${perFile.length} files`);
console.log(`${stepCount} raw palette steps ${WRITE ? 'converted' : 'convertible'} to semantic aliases`);
const noAlias = [...unaliasedSteps].sort((a, b) => b[1] - a[1]);
console.log(
  `${noAlias.reduce((s, [, n]) => s + n, 0)} raw steps left: no alias plays their role in that property`,
);
for (const [step, n] of noAlias.slice(0, 20)) console.log(`  ${step} x${n}`);

const left = [...unmapped].sort((a, b) => b[1] - a[1]);
const leftTotal = left.reduce((s, [, n]) => s + n, 0);
console.log(`${leftTotal} left alone (no palette step within ΔE ${MAX_DELTA}) across ${left.length} distinct colours:`);
for (const [hex, n] of left.slice(0, 40)) {
  const { step, delta } = nearestStep(hex);
  console.log(`  ${hex} x${n}  nearest ${step.name} (${step.hex}) ΔE ${delta.toFixed(1)}`);
}
