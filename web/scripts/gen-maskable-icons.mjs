// web/scripts/gen-maskable-icons.mjs
//
// Rasterise the maskable app icon to the PNG sizes the manifest declares.
//
// Chrome's installability audit wants a PNG of at least 192x192, and Android's
// adaptive-icon mask wants a `purpose: "maskable"` entry -- an SVG satisfies
// neither reliably. packages/identity ships the maskable artwork as SVG with
// the safe zone already drawn in, so this is a pure rasterisation step, run on
// demand rather than at build time; the PNGs are committed.

import { mkdirSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { join } from 'node:path';
import sharp from 'sharp';

const root = fileURLToPath(new URL('../', import.meta.url));
const source = join(root, 'packages/identity/src/app-icon-maskable.svg');
const sizes = [192, 512];
const apps = ['artisan', 'buyer', 'admin'];

const rasterDir = join(root, 'packages/identity/raster');
mkdirSync(rasterDir, { recursive: true });

for (const size of sizes) {
  const png = await sharp(source, { density: 384 })
    .resize(size, size, { fit: 'cover' })
    .png({ compressionLevel: 9, palette: true })
    .toBuffer();

  const name = `app-icon-maskable-${size}.png`;
  writeFileSync(join(rasterDir, name), png);
  for (const app of apps) {
    const dir = join(root, 'apps', app, 'static', 'icons');
    mkdirSync(dir, { recursive: true });
    writeFileSync(join(dir, name), png);
  }
  console.log(`${name}: ${png.length} bytes`);
}
