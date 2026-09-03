/*
 * Kalakriti — Batch 8 documentation site
 *
 * Walks the same package/src directories validate-all.mjs checks and
 * builds docs/manifest.json: one row per SVG with its import line, byte
 * size, and a "blurb" — reused directly from the asset's own top XML
 * comment (every generator already writes a tradition/use comment per
 * Batch 0's own rule) rather than re-authored here.
 *
 * docs/index.html is a static, build-free page that fetches manifest.json
 * and renders a searchable, copy-to-clipboard catalogue.
 */

import { readFileSync, readdirSync, writeFileSync, mkdirSync } from 'node:fs';
import { join, extname, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const PACKAGES = ['icons', 'illustrations', 'ornament', 'patterns', 'motion', 'identity'];

function blurb(raw) {
  const m = raw.match(/<!--([\s\S]*?)-->/);
  if (!m) return '';
  return m[1].replace(/\s+/g, ' ').trim().slice(0, 220);
}

const rows = [];
for (const pkg of PACKAGES) {
  const dir = join(ROOT, 'packages', pkg, 'src');
  let files;
  try { files = readdirSync(dir).filter((f) => extname(f) === '.svg'); }
  catch { continue; }
  for (const file of files) {
    const raw = readFileSync(join(dir, file), 'utf8');
    const name = file.replace(/\.svg$/, '');
    rows.push({
      package: pkg,
      name,
      file: `src/${file}`,
      bytes: Buffer.byteLength(raw, 'utf8'),
      blurb: blurb(raw),
      importLine: `import ${toIdent(name)} from '@kalakriti/${pkg}/src/${file}';`,
    });
  }
}
rows.sort((a, b) => a.package.localeCompare(b.package) || a.name.localeCompare(b.name));

function toIdent(name) {
  return name.replace(/(^|-)([a-z])/g, (_, __, c) => c.toUpperCase());
}

const DOCS = join(ROOT, 'docs');
mkdirSync(DOCS, { recursive: true });
writeFileSync(join(DOCS, 'manifest.json'), JSON.stringify(rows, null, 2));
console.log(rows.length + ' assets written to docs/manifest.json across ' + PACKAGES.length + ' packages.');
