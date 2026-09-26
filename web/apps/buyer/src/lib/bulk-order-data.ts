/**
 * apps/buyer/src/lib/bulk-order-data.ts
 *
 * Rich craft definitions and fallback listing resolvers for the B2B
 * Institutional Bulk Order & RFQ Wizard (/bulk-order). Ensures the wizard
 * displays authentic Indian GI craft heritage and real photographed pieces
 * with full pricing and lead-time calculations, even when the backend is
 * in mock mode or offline.
 */

import type { components } from '@kalakriti/api';
import type { MessageKey, MessageValues } from '@kalakriti/i18n';
import { allStubListings } from './stub-listings';
import { RAW_FALLBACK_GI_LISTINGS, RAW_FALLBACK_NEW_ARRIVALS, toFallbackListing } from './demo-home-listings';
import { locale } from '@kalakriti/i18n';

type Craft = components['schemas']['Craft'];
type ListingSummary = components['schemas']['ListingSummary'];
type TFn = (key: MessageKey, values?: MessageValues) => string;

export interface BulkCraftItem extends Craft {
  category: string;
  sample_image: string;
  artisan_count: number;
  highlight?: string;
  lead_weeks?: number;
}

export const FALLBACK_BULK_CRAFTS: BulkCraftItem[] = [
  {
    id: 'craft-ajrakh',
    slug: 'ajrakh-block-printing',
    display_name: 'Ajrakh Block Printing',
    category: 'Block printing',
    gi_registration_no: 'GI-405',
    gi_certified: true,
    regions: ['Kutch, Gujarat'],
    sample_image: '/craft-images/block_printing/ajrakh_dabu_monsoon_indigo_01.jpeg',
    artisan_count: 14,
    highlight: 'Natural Indigo & Madder 16-stage resist dyeing',
    lead_weeks: 3,
  },
  {
    id: 'craft-banarasi',
    slug: 'banarasi-brocade-weaving',
    display_name: 'Banarasi Brocade Weaving',
    category: 'Weaving',
    gi_registration_no: 'GI-99',
    gi_certified: true,
    regions: ['Varanasi, Uttar Pradesh'],
    sample_image: '/craft-images/weaving_and_looms/banarasi_brocade_weaving_02.jpeg',
    artisan_count: 22,
    highlight: 'Pure Mulberry Silk with tested Zari Kadwa weaving',
    lead_weeks: 4,
  },
  {
    id: 'craft-blue-pottery',
    slug: 'blue-pottery',
    display_name: 'Jaipur Blue Pottery',
    category: 'Pottery',
    gi_registration_no: 'GI-426',
    gi_certified: true,
    regions: ['Jaipur, Rajasthan'],
    sample_image: '/craft-images/pottery/terracotta-jar-floral-blue.jpeg',
    artisan_count: 9,
    highlight: 'Egyptian paste quartz body with cobalt oxide glaze',
    lead_weeks: 2,
  },
  {
    id: 'craft-pashmina',
    slug: 'kashmir-pashmina-sozni',
    display_name: 'Kashmir Pashmina Sozni Embroidery',
    category: 'Embroidery',
    gi_registration_no: 'GI-409',
    gi_certified: true,
    regions: ['Srinagar, Jammu & Kashmir'],
    sample_image: '/craft-images/embroidery/kashmir_pashmina_sozni_01.jpeg',
    artisan_count: 18,
    highlight: 'Hand-spun Changthangi Pashmina with micro-needle needlework',
    lead_weeks: 6,
  },
  {
    id: 'craft-bamboo-assam',
    slug: 'assam-bamboo-craft',
    display_name: 'Assam Bamboo & Cane Craft',
    category: 'Bamboo craft',
    gi_registration_no: 'GI-401',
    gi_certified: true,
    regions: ['Barpeta, Assam'],
    sample_image: '/craft-images/bamboo_craft/assam-bamboo-craft.jpg',
    artisan_count: 16,
    highlight: 'Sustainable treated riverine bamboo & structural jaapi weave',
    lead_weeks: 2,
  },
  {
    id: 'craft-jodhpur',
    slug: 'jodhpur-furniture',
    display_name: 'Jodhpur Handcrafted Furniture',
    category: 'Woodwork',
    gi_registration_no: 'GI-410',
    gi_certified: true,
    regions: ['Jodhpur, Rajasthan'],
    sample_image: '/craft-images/furniture/jodhpur-furniture.jpg',
    artisan_count: 12,
    highlight: 'Kiln-seasoned Sheesham & Acacia with hand-lathed detailing',
    lead_weeks: 5,
  },
  {
    id: 'craft-bhadohi',
    slug: 'bhadohi-carpet',
    display_name: 'Bhadohi Hand-Knotted Carpet',
    category: 'Weaving',
    gi_registration_no: 'GI-411',
    gi_certified: true,
    regions: ['Bhadohi, Uttar Pradesh'],
    sample_image: '/craft-images/home_and_living/bhadohi-carpet.jpg',
    artisan_count: 24,
    highlight: 'High knot-density wool & silk blends for luxury hospitality',
    lead_weeks: 6,
  },
  {
    id: 'craft-moradabad',
    slug: 'moradabad-metal',
    display_name: 'Moradabad Metal Craft',
    category: 'Metalwork',
    gi_registration_no: 'GI-412',
    gi_certified: true,
    regions: ['Moradabad, Uttar Pradesh'],
    sample_image: '/craft-images/home_and_living/moradabad-metal.jpg',
    artisan_count: 15,
    highlight: 'Sand-cast brass & hand-engraved Bidri patina',
    lead_weeks: 3,
  },
  {
    id: 'craft-meenakari',
    slug: 'jaipur-meenakari',
    display_name: 'Jaipur Meenakari Jewellery',
    category: 'Jewellery',
    gi_registration_no: 'GI-414',
    gi_certified: true,
    regions: ['Jaipur, Rajasthan'],
    sample_image: '/craft-images/jewellery/meenakari_enamelwork_01.jpeg',
    artisan_count: 11,
    highlight: 'Vitreous enamel fusing on fine brass & silver alloy',
    lead_weeks: 3,
  },
  {
    id: 'craft-shantiniketan',
    slug: 'shantiniketan-leather',
    display_name: 'Shantiniketan Leather Goods',
    category: 'Leatherwork',
    gi_registration_no: 'GI-415',
    gi_certified: true,
    regions: ['Bolpur Shantiniketan, West Bengal'],
    sample_image: '/craft-images/leatherwork/shantiniketan_embossed_leather_01.jpeg',
    artisan_count: 17,
    highlight: 'Vegetable-tanned sheepskin with embossed batik touch',
    lead_weeks: 2,
  },
  {
    id: 'craft-kolhapuri',
    slug: 'kolhapuri-chappal',
    display_name: 'Kolhapuri Chappal',
    category: 'Leatherwork',
    gi_registration_no: 'GI-416',
    gi_certified: true,
    regions: ['Kolhapur, Maharashtra'],
    sample_image: '/craft-images/leatherwork/kolhapuri_chappals_01.jpeg',
    artisan_count: 20,
    highlight: 'Hand-corded vegetable-tanned leather footwear',
    lead_weeks: 2,
  },
  {
    id: 'craft-kutch-leather',
    slug: 'kutch-leathercraft',
    display_name: 'Kutch Leathercraft (Marwada Style)',
    category: 'Leatherwork',
    gi_registration_no: 'GI-417',
    gi_certified: true,
    regions: ['Kutch, Gujarat'],
    sample_image: '/craft-images/leatherwork/jawaja_tanned_leatherwork_01.jpeg',
    artisan_count: 8,
    highlight: 'Hand-punched copper rivets and mirror-accented mojari',
    lead_weeks: 3,
  },
  {
    id: 'craft-sikki',
    slug: 'sikki-grass-basketry',
    display_name: 'Sikki Grass Basketry',
    category: 'Basketry',
    gi_registration_no: 'GI-403',
    gi_certified: true,
    regions: ['Madhubani, Bihar'],
    sample_image: '/craft-images/basketry/sikki-grass-basketry.jpg',
    artisan_count: 19,
    highlight: 'Golden grass coiled basketry dyed with natural mineral colours',
    lead_weeks: 2,
  },
  {
    id: 'craft-sabai',
    slug: 'sabai-grass-basketry',
    display_name: 'Mayurbhanj Sabai Grass Craft',
    category: 'Basketry',
    gi_registration_no: 'GI-404',
    gi_certified: true,
    regions: ['Mayurbhanj, Odisha'],
    sample_image: '/craft-images/basketry/assamese_cane_bamboo_basketry_01.jpeg',
    artisan_count: 25,
    highlight: 'Tribal women-led rope-braided eco tableware & planters',
    lead_weeks: 2,
  },
  {
    id: 'craft-nizamabad',
    slug: 'nizamabad-black-pottery',
    display_name: 'Nizamabad Black Clay Pottery',
    category: 'Pottery',
    gi_registration_no: 'GI-425',
    gi_certified: true,
    regions: ['Azamgarh, Uttar Pradesh'],
    sample_image: '/craft-images/pottery/nizamabad-black-pottery.jpg',
    artisan_count: 10,
    highlight: 'Smoked reduction firing with pure silver-zinc wire etching',
    lead_weeks: 3,
  },
  {
    id: 'craft-tussar',
    slug: 'tussar-silk-weaving',
    display_name: 'Tussar Silk Weaving',
    category: 'Weaving',
    gi_registration_no: 'GI-351',
    gi_certified: true,
    regions: ['Bhagalpur, Bihar'],
    sample_image: '/craft-images/weaving_and_looms/tussar-silk-saree-navy-floral.jpeg',
    artisan_count: 21,
    highlight: 'Wild forest Antheraea silk with organic golden sheen',
    lead_weeks: 4,
  },
  {
    id: 'craft-zari',
    slug: 'zari-work',
    display_name: 'Surat Zari Work',
    category: 'Weaving',
    gi_registration_no: 'GI-433',
    gi_certified: true,
    regions: ['Surat, Gujarat'],
    sample_image: '/craft-images/weaving_and_looms/zari-work-sari-beige.jpeg',
    artisan_count: 18,
    highlight: 'Real electroplated metallic thread embroidery & bordering',
    lead_weeks: 3,
  },
  {
    id: 'craft-sanganeri',
    slug: 'sanganeri-block-printing',
    display_name: 'Sanganeri Block Printing',
    category: 'Block printing',
    gi_registration_no: 'GI-354',
    gi_certified: true,
    regions: ['Sanganer, Jaipur, Rajasthan'],
    sample_image: '/craft-images/block_printing/sanganeri-block-print-kurta-white.jpeg',
    artisan_count: 16,
    highlight: 'Fine floral buti hand prints on bleached cambric cotton',
    lead_weeks: 2,
  },
  {
    id: 'craft-filigree',
    slug: 'cuttack-silver-filigree',
    display_name: 'Cuttack Silver Filigree (Tarakasi)',
    category: 'Jewellery',
    gi_registration_no: 'GI-413',
    gi_certified: true,
    regions: ['Cuttack, Odisha'],
    sample_image: '/craft-images/jewellery/tarakashi_silver_filigree_02.jpeg',
    artisan_count: 13,
    highlight: '0.925 sterling silver wire drawn down to hair-thin curls',
    lead_weeks: 4,
  },
  {
    id: 'craft-tripura',
    slug: 'tripura-bamboo-craft',
    display_name: 'Tripura Bamboo & Cane Craft',
    category: 'Bamboo craft',
    gi_registration_no: 'GI-402',
    gi_certified: true,
    regions: ['West Tripura, Tripura'],
    sample_image: '/craft-images/bamboo_craft/tripura-bamboo-craft.jpg',
    artisan_count: 14,
    highlight: 'Muli bamboo screen weaving and modern modular furniture',
    lead_weeks: 3,
  },
];

/**
 * Returns authentic photographed listings matching a given craft slug.
 * Always returns at least 1-6 pieces so no craft shows an empty state.
 */
export function getStubListingsForCraft(slug: string, t: TFn): ListingSummary[] {
  const all = allStubListings(t);

  // Exact slug match
  let matches = all.filter((l) => l.craft_slug === slug);
  if (matches.length > 0) return matches;

  // Partial or normalized slug match
  const normalized = slug.toLowerCase().replace(/-(weaving|craft|printing|sozni|pottery|goods)$/, '');
  matches = all.filter(
    (l) =>
      l.craft_slug?.includes(normalized) ||
      l.craft_name?.toLowerCase().includes(normalized) ||
      slug.includes(l.craft_slug ?? ''),
  );
  if (matches.length > 0) return matches;

  // Match from home fallback items
  const homeMatches = [...RAW_FALLBACK_GI_LISTINGS, ...RAW_FALLBACK_NEW_ARRIVALS]
    .filter((r) => r.craft_slug === slug || r.craft_slug.includes(normalized))
    .map((r) => toFallbackListing(r, t, locale.code));
  if (homeMatches.length > 0) return homeMatches;

  // Broad category match
  const craftDef = FALLBACK_BULK_CRAFTS.find((c) => c.slug === slug);
  if (craftDef?.category) {
    const catMatches = all.filter((l) => (l as any).category?.toLowerCase() === craftDef.category.toLowerCase());
    if (catMatches.length > 0) return catMatches.slice(0, 6);
  }

  // Graceful fallback to first 4 listings so buyer can always complete order
  return all.slice(0, 4);
}

/**
 * Return realistic artisan headcount for the feasibility banner.
 */
export function getArtisanCountForCraft(slug: string): number {
  const craft = FALLBACK_BULK_CRAFTS.find((c) => c.slug === slug);
  return craft?.artisan_count ?? 12;
}
