// apps/buyer/src/lib/craft-icon.ts
//
// Maps a craft slug/display name to one of the twelve craft-family icons the
// icon pack ships (packages/icons/src: bamboo, basketry, block-printing,
// embroidery, jewellery, leather, metalwork, painting, pottery, stone,
// weaving, woodwork). Substring match against the slug, ordered so a more
// specific term (e.g. "block-printing") wins over a broader one it contains
// no part of; falls back to a generic craft glyph.

import type { IconName } from '@kalakriti/icons';

const FAMILIES: readonly IconName[] = [
  'block-printing',
  'embroidery',
  'weaving',
  'pottery',
  'metalwork',
  'woodwork',
  'jewellery',
  'basketry',
  'bamboo',
  'leather',
  'painting',
  'stone',
];

export function craftIcon(slugOrName: string): IconName {
  const key = slugOrName.toLowerCase();
  for (const family of FAMILIES) {
    if (key.includes(family)) return family;
  }
  if (key.includes('weav') || key.includes('loom') || key.includes('handloom')) return 'weaving';
  if (key.includes('pot') || key.includes('terracotta') || key.includes('ceramic')) return 'pottery';
  if (key.includes('embroid') || key.includes('zari') || key.includes('kantha')) return 'embroidery';
  if (key.includes('print') || key.includes('ajrakh') || key.includes('batik')) return 'block-printing';
  if (key.includes('wood') || key.includes('carv')) return 'woodwork';
  if (key.includes('metal') || key.includes('brass') || key.includes('bronze')) return 'metalwork';
  if (key.includes('jewel')) return 'jewellery';
  if (key.includes('leather')) return 'leather';
  if (key.includes('paint') || key.includes('madhubani') || key.includes('warli')) return 'painting';
  if (key.includes('bamboo') || key.includes('cane')) return 'bamboo';
  if (key.includes('basket')) return 'basketry';
  if (key.includes('stone') || key.includes('marble')) return 'stone';
  return 'handmade-certified';
}
