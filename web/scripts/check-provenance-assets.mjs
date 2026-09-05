import { readFile } from 'node:fs/promises';
import { access } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { resolve } from 'node:path';

const root = resolve(fileURLToPath(new URL('..', import.meta.url)));
const required = [
  'packages/identity/src/seal.svg',
  'packages/identity/src/heritage-crest-coarse.svg',
  'packages/patterns/src/qr-frame.svg',
  'packages/patterns/provenance.css',
  'packages/print/src/tag-a4-colour.svg',
  'packages/print/src/tag-a4-mono.svg',
];

for (const relative of required) {
  await access(resolve(root, relative));
}

const qr = await readFile(resolve(root, 'packages/patterns/src/qr-frame.svg'), 'utf8');
const quietZone = qr.match(/172\.8x172\.8 clear inner square/i);
if (!quietZone || !qr.includes('14% of the frame')) {
  throw new Error('QR frame must document the 14% quiet-zone geometry.');
}

const provenance = await readFile(resolve(root, 'packages/patterns/provenance.css'), 'utf8');
for (const asset of ['pattern-warp-weft.svg', 'pattern-jaali-fill.svg']) {
  if (!provenance.includes(asset)) {
    throw new Error(`Standalone provenance CSS does not reference ${asset}.`);
  }
}

const print = await readFile(resolve(root, 'packages/print/src/tag-a4-colour.svg'), 'utf8');
if (!print.includes('qr-frame') || !print.includes('seal')) {
  throw new Error('Colour print tag must include the QR frame and provenance seal.');
}

console.log(`Provenance asset checks passed (${required.length} required files).`);
