// apps/artisan/src/lib/ontology.ts
//
// Static local reference data for the registration wizard's craft and
// district pickers. Neither is generated: services/bff/openapi.json has no
// GET /crafts or /districts endpoint (only the opaque `craft_id` string
// param on /listings and /search), so there is nothing to generate a client
// from. Per the ABSOLUTE RULE this governs request/response *types*, not
// local UI reference data -- the same category as @kalakriti/i18n's
// LOCALES table.
//
// CRAFTS' ids match @kalakriti/icons' craft icon names 1:1 (weaving,
// block-printing, ...) so the picker's icon and its craft_id come from the
// same source rather than a second mapping that could drift. `other` has no
// craft icon and is the escape hatch for a craft not in this list.
//
// DISTRICTS is a small placeholder seed, NOT an exhaustive list of Indian
// districts -- there is no ontology endpoint to seed it from yet, and
// hand-typing all ~780 districts for a demo build is effort spent on the
// wrong thing. It covers well-known craft-cluster districts so the search
// step is demonstrably usable, and every screen that uses it also offers
// "my district is not listed" as a free-text/voice fallback -- see
// /register/district.

import type { IconName } from '@kalakriti/icons';
import type { MessageKey } from '@kalakriti/i18n';

export interface Craft {
  id: string;
  icon: IconName;
  nameKey: MessageKey;
}

export const CRAFTS: readonly Craft[] = [
  { id: 'weaving', icon: 'weaving', nameKey: 'craft.weaving.name' },
  { id: 'block-printing', icon: 'block-printing', nameKey: 'craft.block-printing.name' },
  { id: 'pottery', icon: 'pottery', nameKey: 'craft.pottery.name' },
  { id: 'metalwork', icon: 'metalwork', nameKey: 'craft.metalwork.name' },
  { id: 'woodwork', icon: 'woodwork', nameKey: 'craft.woodwork.name' },
  { id: 'embroidery', icon: 'embroidery', nameKey: 'craft.embroidery.name' },
  { id: 'painting', icon: 'painting', nameKey: 'craft.painting.name' },
  { id: 'basketry', icon: 'basketry', nameKey: 'craft.basketry.name' },
  { id: 'jewellery', icon: 'jewellery', nameKey: 'craft.jewellery.name' },
  { id: 'leather', icon: 'leather', nameKey: 'craft.leather.name' },
  { id: 'stone', icon: 'stone', nameKey: 'craft.stone.name' },
  { id: 'bamboo', icon: 'bamboo', nameKey: 'craft.bamboo.name' },
  { id: 'other', icon: 'more-horizontal', nameKey: 'craft.other.name' },
];

export interface District {
  id: string;
  name: string;
  state: string;
}

export const DISTRICTS: readonly District[] = [
  { id: 'varanasi', name: 'Varanasi', state: 'Uttar Pradesh' },
  { id: 'moradabad', name: 'Moradabad', state: 'Uttar Pradesh' },
  { id: 'bhadohi', name: 'Bhadohi', state: 'Uttar Pradesh' },
  { id: 'firozabad', name: 'Firozabad', state: 'Uttar Pradesh' },
  { id: 'saharanpur', name: 'Saharanpur', state: 'Uttar Pradesh' },
  { id: 'jaipur', name: 'Jaipur', state: 'Rajasthan' },
  { id: 'jodhpur', name: 'Jodhpur', state: 'Rajasthan' },
  { id: 'bikaner', name: 'Bikaner', state: 'Rajasthan' },
  { id: 'barmer', name: 'Barmer', state: 'Rajasthan' },
  { id: 'kutch', name: 'Kutch', state: 'Gujarat' },
  { id: 'surat', name: 'Surat', state: 'Gujarat' },
  { id: 'patan', name: 'Patan', state: 'Gujarat' },
  { id: 'chanderi', name: 'Chanderi', state: 'Madhya Pradesh' },
  { id: 'bhopal', name: 'Bhopal', state: 'Madhya Pradesh' },
  { id: 'channapatna', name: 'Channapatna', state: 'Karnataka' },
  { id: 'mysuru', name: 'Mysuru', state: 'Karnataka' },
  { id: 'kanchipuram', name: 'Kanchipuram', state: 'Tamil Nadu' },
  { id: 'madurai', name: 'Madurai', state: 'Tamil Nadu' },
  { id: 'pochampally', name: 'Pochampally', state: 'Telangana' },
  { id: 'srikalahasti', name: 'Srikalahasti', state: 'Andhra Pradesh' },
  { id: 'murshidabad', name: 'Murshidabad', state: 'West Bengal' },
  { id: 'nadia', name: 'Nadia', state: 'West Bengal' },
  { id: 'bhagalpur', name: 'Bhagalpur', state: 'Bihar' },
  { id: 'madhubani', name: 'Madhubani', state: 'Bihar' },
  { id: 'cuttack', name: 'Cuttack', state: 'Odisha' },
  { id: 'puri', name: 'Puri', state: 'Odisha' },
  { id: 'srinagar', name: 'Srinagar', state: 'Jammu and Kashmir' },
];

export function matchesQuery(label: string, query: string): boolean {
  const normalised = query.trim().toLowerCase();
  if (normalised === '') return true;
  return label.toLowerCase().includes(normalised);
}
