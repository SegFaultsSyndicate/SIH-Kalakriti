// web/scripts/check-brand-tokens.mjs
//
// The PWA manifest needs colours as literal hex (JSON cannot resolve a CSS
// custom property), so packages/tokens/src/brand.js duplicates three values
// from palette.css. Duplication is fine as long as it cannot drift silently.
// This asserts each brand.js value still equals the token it claims to mirror.
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const paletteUrl = new URL('../packages/tokens/src/palette.css', import.meta.url);
const palette = readFileSync(fileURLToPath(paletteUrl), 'utf8');
const brand = await import('../packages/tokens/src/brand.js');

// Only the :root block; later theme blocks redefine semantic tokens, not the
// raw scale values these three point at, but scoping keeps the match honest.
const rootBlock = palette.slice(palette.indexOf(':root'), palette.indexOf('\n}'));

let failed = 0;
for (const [key, token] of Object.entries(brand.BRAND_TOKEN_SOURCES)) {
  const match = rootBlock.match(new RegExp(token + ':[ \\t]*(#[0-9A-Fa-f]{6})'));
  if (!match) {
    console.error(`brand-tokens: ${token} not found in palette.css :root`);
    failed++;
    continue;
  }
  const expected = match[1].toUpperCase();
  const actual = String(brand[key]).toUpperCase();
  if (expected !== actual) {
    console.error(`brand-tokens: ${key} is ${actual}, but ${token} is ${expected}`);
    failed++;
  }
}

if (failed > 0) {
  console.error(`brand-tokens: ${failed} mismatch(es). Update packages/tokens/src/brand.js.`);
  process.exit(1);
}
console.log(
  `brand-tokens: ${Object.keys(brand.BRAND_TOKEN_SOURCES).length} values match palette.css`,
);
