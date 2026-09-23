/**
 * apps/buyer/src/lib/craft-categories.ts
 *
 * The 12 authentic National Artisan Craft Categories of Kalakriti.
 * Aligned 1:1 with the artisan onboarding ontology and @kalakriti/icons.
 *
 * `name` stays a plain, stable English string -- it is the identifier
 * matched against `/search?category=` and stub-listings.ts's `category`
 * field, never rendered directly. Display text goes through `nameKey`/
 * `nativeNameKey`/`subtitleKey`/`taglineKey` (see I18N_PLAN.md §4.1(c) for
 * why `nativeNameKey` replaced the old hardcoded-Hindi `hindiName` field).
 */

import type { IconName } from '@kalakriti/icons';
import type { MessageKey } from '@kalakriti/i18n';

export interface CraftCategory {
  id: string;
  name: string;
  nameKey: MessageKey;
  nativeNameKey: MessageKey;
  icon: IconName;
  subtitleKey: MessageKey;
  taglineKey: MessageKey;
  funFactKey: MessageKey;
  query: string;
  sampleImage: string;
  regions: string[];
  giCount: number;
}

export const ARTISAN_CRAFT_CATEGORIES: readonly CraftCategory[] = [
  {
    id: 'weaving',
    name: 'Weaving',
    nameKey: 'craft.weaving.name',
    nativeNameKey: 'craft.weaving.nativeName',
    icon: 'weaving',
    subtitleKey: 'craft.weaving.subtitle',
    taglineKey: 'craft.weaving.tagline',
    funFactKey: 'craft.weaving.funFact',
    query: 'weaving',
    sampleImage: '/craft-images/weaving_and_looms/category_cover.jpg',
    regions: ['Varanasi (UP)', 'Chanderi (MP)', 'Kanchipuram (TN)', 'Patan (Gujarat)'],
    giCount: 38,
  },
  {
    id: 'block-printing',
    name: 'Block printing',
    nameKey: 'craft.block-printing.name',
    nativeNameKey: 'craft.block-printing.nativeName',
    icon: 'block-printing',
    subtitleKey: 'craft.block-printing.subtitle',
    taglineKey: 'craft.block-printing.tagline',
    funFactKey: 'craft.block-printing.funFact',
    query: 'block printing',
    sampleImage: '/craft-images/block_printing/category_cover.jpg',
    regions: ['Kutch (Gujarat)', 'Bagru (Rajasthan)', 'Machilipatnam (AP)'],
    giCount: 14,
  },
  {
    id: 'pottery',
    name: 'Pottery',
    nameKey: 'craft.pottery.name',
    nativeNameKey: 'craft.pottery.nativeName',
    icon: 'pottery',
    subtitleKey: 'craft.pottery.subtitle',
    taglineKey: 'craft.pottery.tagline',
    funFactKey: 'craft.pottery.funFact',
    query: 'pottery',
    sampleImage: '/craft-images/pottery/category_cover.jpg',
    regions: ['Khurja (UP)', 'Nizamabad (UP)', 'Jaipur (Rajasthan)', 'Bankura (WB)'],
    giCount: 11,
  },
  {
    id: 'metalwork',
    name: 'Metalwork',
    nameKey: 'craft.metalwork.name',
    nativeNameKey: 'craft.metalwork.nativeName',
    icon: 'metalwork',
    subtitleKey: 'craft.metalwork.subtitle',
    taglineKey: 'craft.metalwork.tagline',
    funFactKey: 'craft.metalwork.funFact',
    query: 'metalwork',
    sampleImage: '/craft-images/metalwork/category_cover.jpg',
    regions: ['Bastar (Chhattisgarh)', 'Moradabad (UP)', 'Bidar (Karnataka)', 'Thanjavur (TN)'],
    giCount: 19,
  },
  {
    id: 'woodwork',
    name: 'Woodwork',
    nameKey: 'craft.woodwork.name',
    nativeNameKey: 'craft.woodwork.nativeName',
    icon: 'woodwork',
    subtitleKey: 'craft.woodwork.subtitle',
    taglineKey: 'craft.woodwork.tagline',
    funFactKey: 'craft.woodwork.funFact',
    query: 'woodwork',
    sampleImage: '/craft-images/woodwork/category_cover.jpg',
    regions: ['Saharanpur (UP)', 'Srinagar (J&K)', 'Channapatna (Karnataka)', 'Jodhpur (RJ)'],
    giCount: 16,
  },
  {
    id: 'embroidery',
    name: 'Embroidery',
    nameKey: 'craft.embroidery.name',
    nativeNameKey: 'craft.embroidery.nativeName',
    icon: 'embroidery',
    subtitleKey: 'craft.embroidery.subtitle',
    taglineKey: 'craft.embroidery.tagline',
    funFactKey: 'craft.embroidery.funFact',
    query: 'embroidery',
    sampleImage: '/craft-images/embroidery/category_cover.jpg',
    regions: ['Lucknow (UP)', 'Shantiniketan (WB)', 'Kashmir', 'Kutch (Gujarat)'],
    giCount: 21,
  },
  {
    id: 'painting',
    name: 'Painting',
    nameKey: 'craft.painting.name',
    nativeNameKey: 'craft.painting.nativeName',
    icon: 'painting',
    subtitleKey: 'craft.painting.subtitle',
    taglineKey: 'craft.painting.tagline',
    funFactKey: 'craft.painting.funFact',
    query: 'painting',
    sampleImage: '/craft-images/paintings/category_cover.jpg',
    regions: ['Madhubani (Bihar)', 'Raghurajpur (Odisha)', 'Nathdwara (RJ)', 'Warli (MH)'],
    giCount: 27,
  },
  {
    id: 'basketry',
    name: 'Basketry',
    nameKey: 'craft.basketry.name',
    nativeNameKey: 'craft.basketry.nativeName',
    icon: 'basketry',
    subtitleKey: 'craft.basketry.subtitle',
    taglineKey: 'craft.basketry.tagline',
    funFactKey: 'craft.basketry.funFact',
    query: 'basketry',
    sampleImage: '/craft-images/basketry/category_cover.jpg',
    regions: ['Mayurbhanj (Odisha)', 'Madhubani (Bihar)', 'Prayagraj (UP)'],
    giCount: 8,
  },
  {
    id: 'jewellery',
    name: 'Jewellery',
    nameKey: 'craft.jewellery.name',
    nativeNameKey: 'craft.jewellery.nativeName',
    icon: 'jewellery',
    subtitleKey: 'craft.jewellery.subtitle',
    taglineKey: 'craft.jewellery.tagline',
    funFactKey: 'craft.jewellery.funFact',
    query: 'jewellery',
    sampleImage: '/craft-images/jewellery/category_cover.jpg',
    regions: ['Cuttack (Odisha)', 'Karimnagar (Telangana)', 'Jaipur (Rajasthan)'],
    giCount: 12,
  },
  {
    id: 'leather',
    name: 'Leatherwork',
    nameKey: 'craft.leather.name',
    nativeNameKey: 'craft.leather.nativeName',
    icon: 'leather',
    subtitleKey: 'craft.leather.subtitle',
    taglineKey: 'craft.leather.tagline',
    funFactKey: 'craft.leather.funFact',
    query: 'leatherwork',
    sampleImage: '/craft-images/leatherwork/category_cover.jpg',
    regions: ['Shantiniketan (WB)', 'Kolhapur (MH)', 'Indore (MP)'],
    giCount: 7,
  },
  {
    id: 'stone',
    name: 'Stone carving',
    nameKey: 'craft.stone.name',
    nativeNameKey: 'craft.stone.nativeName',
    icon: 'stone',
    subtitleKey: 'craft.stone.subtitle',
    taglineKey: 'craft.stone.tagline',
    funFactKey: 'craft.stone.funFact',
    query: 'stone carving',
    sampleImage: '/craft-images/stone_carving/category_cover.jpg',
    regions: ['Agra (UP)', 'Puri (Odisha)', 'Varanasi (UP)', 'Mamallapuram (TN)'],
    giCount: 9,
  },
  {
    id: 'bamboo',
    name: 'Bamboo craft',
    nameKey: 'craft.bamboo.name',
    nativeNameKey: 'craft.bamboo.nativeName',
    icon: 'bamboo',
    subtitleKey: 'craft.bamboo.subtitle',
    taglineKey: 'craft.bamboo.tagline',
    funFactKey: 'craft.bamboo.funFact',
    query: 'bamboo craft',
    sampleImage: '/craft-images/bamboo_craft/category_cover.jpg',
    regions: ['Assam', 'Tripura', 'Nagaland', 'Kerala'],
    giCount: 15,
  },
];
