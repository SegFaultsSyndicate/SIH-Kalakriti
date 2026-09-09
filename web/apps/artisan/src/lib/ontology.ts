// apps/artisan/src/lib/ontology.ts
//
// Craft catalog: loadCrafts() fetches the real ontology from GET /crafts
// (services/bff/openapi.json now has it -- it did not when this file was
// first written, which is why craft ids used to be local slugs; core-svc's
// Artisan.Register rejects anything that isn't a real craft UUID). CRAFT_META
// is local UI-only metadata (icon, i18n key) keyed by the backend's `slug`,
// the same category as @kalakriti/i18n's LOCALES table -- a craft the
// backend knows about but this map doesn't still renders, with a generic
// icon and its own display_name instead of a translated one.
//
// DISTRICTS is a small placeholder seed, NOT an exhaustive list of Indian
// districts -- there is no ontology endpoint to seed it from yet, and
// hand-typing all ~780 districts for a demo build is effort spent on the
// wrong thing. It covers well-known craft-cluster districts so the search
// step is demonstrably usable, and every screen that uses it also offers
// "my district is not listed" as a free-text/voice fallback -- see
// /register/district. Each row's stateCode is what region.state_code on
// POST /artisans is actually built from -- this table *is* the lookup, not
// a separate one keyed off the full state name.

import { listCrafts } from '@kalakriti/api';
import { getPref, setPref } from '@kalakriti/offline';
import type { IconName } from '@kalakriti/icons';
import type { MessageKey } from '@kalakriti/i18n';

const CRAFTS_PREF_KEY = 'ontology.crafts';

export interface Craft {
  /** Real craft UUID from the backend -- what Artisan.Register/CreateListing expect. */
  id: string;
  slug: string;
  displayName: string;
  icon: IconName;
  nameKey: MessageKey;
}

const CRAFT_META: Readonly<Record<string, { icon: IconName; nameKey: MessageKey }>> = {
  weaving: { icon: 'weaving', nameKey: 'craft.weaving.name' },
  'block-printing': { icon: 'block-printing', nameKey: 'craft.block-printing.name' },
  pottery: { icon: 'pottery', nameKey: 'craft.pottery.name' },
  metalwork: { icon: 'metalwork', nameKey: 'craft.metalwork.name' },
  woodwork: { icon: 'woodwork', nameKey: 'craft.woodwork.name' },
  embroidery: { icon: 'embroidery', nameKey: 'craft.embroidery.name' },
  painting: { icon: 'painting', nameKey: 'craft.painting.name' },
  basketry: { icon: 'basketry', nameKey: 'craft.basketry.name' },
  jewellery: { icon: 'jewellery', nameKey: 'craft.jewellery.name' },
  leather: { icon: 'leather', nameKey: 'craft.leather.name' },
  stone: { icon: 'stone', nameKey: 'craft.stone.name' },
  bamboo: { icon: 'bamboo', nameKey: 'craft.bamboo.name' },
};
const OTHER_META = { icon: 'more-horizontal' as IconName, nameKey: 'craft.other.name' as MessageKey };

let cache: Promise<Craft[]> | undefined;

/**
 * Fetches the real craft catalog, cached for the session. A successful fetch
 * is also written to the same Dexie pref store the registration draft uses,
 * so a later offline load (registration's own step one, which must work
 * without connectivity like the rest of this PWA) still has a list to show
 * instead of rendering zero tiles and blocking the wizard -- see
 * workbox's NetworkFirst /api/v1 rule in vite.config.ts, which this backs up
 * for the case where that request was never made while online. Rejects (and
 * clears the cache, so the next call retries) only when neither the network
 * nor the pref has an answer -- callers that want an empty-list fallback do
 * `await loadCrafts().catch(() => [])` themselves, since a caller mid-wizard
 * may instead want to show a retry affordance.
 */
export function loadCrafts(): Promise<Craft[]> {
  if (!cache) {
    cache = listCrafts()
      .then((res) => {
        const crafts = (res.crafts ?? []).map((c): Craft => {
          const slug = c.slug ?? '';
          const meta = CRAFT_META[slug] ?? OTHER_META;
          return { id: c.id ?? '', slug, displayName: c.display_name ?? slug, icon: meta.icon, nameKey: meta.nameKey };
        });
        void setPref(CRAFTS_PREF_KEY, crafts);
        return crafts;
      })
      .catch(async (err) => {
        const cached = await getPref<Craft[]>(CRAFTS_PREF_KEY);
        if (cached && cached.length > 0) return cached;
        throw err;
      });
    cache.catch(() => {
      cache = undefined;
    });
  }
  return cache;
}

/** Looks up one craft by its backend UUID, for redisplaying a stored craft_id. */
export async function getCraftById(id: string): Promise<Craft | undefined> {
  const crafts = await loadCrafts().catch(() => []);
  return crafts.find((c) => c.id === id);
}

/** A craft's display label: its translated name if known locally, else its own display_name. */
export function craftLabel(craft: Craft, t: (key: MessageKey) => string): string {
  return CRAFT_META[craft.slug] ? t(craft.nameKey) : craft.displayName;
}

/** Every Indian state and union territory, ISO 3166-2:IN codes -- the free-text district picker's state dropdown needs full coverage even though DISTRICTS itself only seeds well-known craft-cluster districts. */
export const STATES: readonly { code: string; name: string }[] = [
  { code: 'IN-AP', name: 'Andhra Pradesh' },
  { code: 'IN-AR', name: 'Arunachal Pradesh' },
  { code: 'IN-AS', name: 'Assam' },
  { code: 'IN-BR', name: 'Bihar' },
  { code: 'IN-CT', name: 'Chhattisgarh' },
  { code: 'IN-GA', name: 'Goa' },
  { code: 'IN-GJ', name: 'Gujarat' },
  { code: 'IN-HR', name: 'Haryana' },
  { code: 'IN-HP', name: 'Himachal Pradesh' },
  { code: 'IN-JH', name: 'Jharkhand' },
  { code: 'IN-KA', name: 'Karnataka' },
  { code: 'IN-KL', name: 'Kerala' },
  { code: 'IN-MP', name: 'Madhya Pradesh' },
  { code: 'IN-MH', name: 'Maharashtra' },
  { code: 'IN-MN', name: 'Manipur' },
  { code: 'IN-ML', name: 'Meghalaya' },
  { code: 'IN-MZ', name: 'Mizoram' },
  { code: 'IN-NL', name: 'Nagaland' },
  { code: 'IN-OR', name: 'Odisha' },
  { code: 'IN-PB', name: 'Punjab' },
  { code: 'IN-RJ', name: 'Rajasthan' },
  { code: 'IN-SK', name: 'Sikkim' },
  { code: 'IN-TN', name: 'Tamil Nadu' },
  { code: 'IN-TG', name: 'Telangana' },
  { code: 'IN-TR', name: 'Tripura' },
  { code: 'IN-UP', name: 'Uttar Pradesh' },
  { code: 'IN-UT', name: 'Uttarakhand' },
  { code: 'IN-WB', name: 'West Bengal' },
  { code: 'IN-AN', name: 'Andaman and Nicobar Islands' },
  { code: 'IN-CH', name: 'Chandigarh' },
  { code: 'IN-DH', name: 'Dadra and Nagar Haveli and Daman and Diu' },
  { code: 'IN-DL', name: 'Delhi' },
  { code: 'IN-JK', name: 'Jammu and Kashmir' },
  { code: 'IN-LA', name: 'Ladakh' },
  { code: 'IN-LD', name: 'Lakshadweep' },
  { code: 'IN-PY', name: 'Puducherry' },
];

export interface District {
  id: string;
  name: string;
  state: string;
  stateCode: string;
}

export const DISTRICTS: readonly District[] = [
  { id: 'varanasi', name: 'Varanasi', state: 'Uttar Pradesh', stateCode: 'IN-UP' },
  { id: 'moradabad', name: 'Moradabad', state: 'Uttar Pradesh', stateCode: 'IN-UP' },
  { id: 'bhadohi', name: 'Bhadohi', state: 'Uttar Pradesh', stateCode: 'IN-UP' },
  { id: 'firozabad', name: 'Firozabad', state: 'Uttar Pradesh', stateCode: 'IN-UP' },
  { id: 'saharanpur', name: 'Saharanpur', state: 'Uttar Pradesh', stateCode: 'IN-UP' },
  { id: 'jaipur', name: 'Jaipur', state: 'Rajasthan', stateCode: 'IN-RJ' },
  { id: 'jodhpur', name: 'Jodhpur', state: 'Rajasthan', stateCode: 'IN-RJ' },
  { id: 'bikaner', name: 'Bikaner', state: 'Rajasthan', stateCode: 'IN-RJ' },
  { id: 'barmer', name: 'Barmer', state: 'Rajasthan', stateCode: 'IN-RJ' },
  { id: 'kutch', name: 'Kutch', state: 'Gujarat', stateCode: 'IN-GJ' },
  { id: 'surat', name: 'Surat', state: 'Gujarat', stateCode: 'IN-GJ' },
  { id: 'patan', name: 'Patan', state: 'Gujarat', stateCode: 'IN-GJ' },
  { id: 'chanderi', name: 'Chanderi', state: 'Madhya Pradesh', stateCode: 'IN-MP' },
  { id: 'bhopal', name: 'Bhopal', state: 'Madhya Pradesh', stateCode: 'IN-MP' },
  { id: 'channapatna', name: 'Channapatna', state: 'Karnataka', stateCode: 'IN-KA' },
  { id: 'mysuru', name: 'Mysuru', state: 'Karnataka', stateCode: 'IN-KA' },
  { id: 'kanchipuram', name: 'Kanchipuram', state: 'Tamil Nadu', stateCode: 'IN-TN' },
  { id: 'madurai', name: 'Madurai', state: 'Tamil Nadu', stateCode: 'IN-TN' },
  { id: 'pochampally', name: 'Pochampally', state: 'Telangana', stateCode: 'IN-TG' },
  { id: 'srikalahasti', name: 'Srikalahasti', state: 'Andhra Pradesh', stateCode: 'IN-AP' },
  { id: 'murshidabad', name: 'Murshidabad', state: 'West Bengal', stateCode: 'IN-WB' },
  { id: 'nadia', name: 'Nadia', state: 'West Bengal', stateCode: 'IN-WB' },
  { id: 'bhagalpur', name: 'Bhagalpur', state: 'Bihar', stateCode: 'IN-BR' },
  { id: 'madhubani', name: 'Madhubani', state: 'Bihar', stateCode: 'IN-BR' },
  { id: 'cuttack', name: 'Cuttack', state: 'Odisha', stateCode: 'IN-OR' },
  { id: 'puri', name: 'Puri', state: 'Odisha', stateCode: 'IN-OR' },
  { id: 'srinagar', name: 'Srinagar', state: 'Jammu and Kashmir', stateCode: 'IN-JK' },
];

export function matchesQuery(label: string, query: string): boolean {
  const normalised = query.trim().toLowerCase();
  if (normalised === '') return true;
  return label.toLowerCase().includes(normalised);
}
