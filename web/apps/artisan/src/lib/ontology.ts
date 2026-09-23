// apps/artisan/src/lib/ontology.ts
//
// Reference data for the registration wizard's craft and district pickers.
//
// CRAFTS used to be a static local list because, at the time this was
// written, services/bff/openapi.json had no GET /crafts endpoint to generate
// a client from. That's no longer true -- /crafts is a real, public (no
// auth) bff route backed by core-svc's craft ontology, and POST /artisans
// requires craft_ids to be real ontology UUIDs from it, not an arbitrary
// local slug. loadCrafts() below fetches the live list via @kalakriti/api's
// generated listCrafts() (per the ABSOLUTE RULE -- this now IS a typed
// request/response, so it goes through the generated client, not hand-rolled
// fetch), and caches it via @kalakriti/offline's getCached/setCached so the
// picker still works offline after the first successful load. Each entry's
// icon is guessed from its slug/display name against ICON_KEYWORDS below --
// the ontology has no icon field of its own -- falling back to a generic
// icon for anything unmatched.
//
// DISTRICTS is still a small placeholder seed, NOT an exhaustive list of
// Indian districts -- there is no ontology endpoint for districts, and
// hand-typing all ~780 for a demo build is effort spent on the wrong thing.
// It covers well-known craft-cluster districts so the search step is
// demonstrably usable, and every screen that uses it also offers "my
// district is not listed" as a free-text/voice fallback -- see
// /register/district. STATE_CODES maps each state name appearing here to its
// ISO 3166-2:IN code, because POST /artisans' region.state_code is required
// and must be in that form (see proto/common/v1/common.proto's GeoRegion).

import type { IconName } from '@kalakriti/icons';
import { listCrafts } from '@kalakriti/api';
import { getCached, setCached } from '@kalakriti/offline';

export interface Craft {
  /** Real craft ontology UUID -- what actually goes into POST /artisans' craft_ids. */
  id: string;
  slug: string;
  displayName: string;
  icon: IconName;
}

const CRAFTS_CACHE_KEY = 'ontology.crafts';

// Fallback shown when the backend is unreachable and there's no cached
// fetch yet (fresh install, or first run offline) -- this is the original
// static list from before /crafts existed. Its ids are slugs, not real
// ontology UUIDs, so a submission made against this list will fail
// POST /artisans' craft_ids validation once it reaches a live backend --
// it exists purely so the picker isn't empty during local frontend work
// with no backend running. Network data always wins once available.
const OFFLINE_FALLBACK_CRAFTS: readonly Craft[] = [
  { id: 'weaving', slug: 'weaving', displayName: 'Weaving', icon: 'weaving' },
  { id: 'block-printing', slug: 'block-printing', displayName: 'Block Printing', icon: 'block-printing' },
  { id: 'pottery', slug: 'pottery', displayName: 'Pottery', icon: 'pottery' },
  { id: 'metalwork', slug: 'metalwork', displayName: 'Metalwork', icon: 'metalwork' },
  { id: 'woodwork', slug: 'woodwork', displayName: 'Woodwork', icon: 'woodwork' },
  { id: 'embroidery', slug: 'embroidery', displayName: 'Embroidery', icon: 'embroidery' },
  { id: 'painting', slug: 'painting', displayName: 'Painting', icon: 'painting' },
  { id: 'basketry', slug: 'basketry', displayName: 'Basketry', icon: 'basketry' },
  { id: 'jewellery', slug: 'jewellery', displayName: 'Jewellery', icon: 'jewellery' },
  { id: 'leather', slug: 'leather', displayName: 'Leather', icon: 'leather' },
  { id: 'stone', slug: 'stone', displayName: 'Stone Carving', icon: 'stone' },
  { id: 'bamboo', slug: 'bamboo', displayName: 'Bamboo Craft', icon: 'bamboo' },
  { id: 'other', slug: 'other', displayName: 'Something else', icon: 'more-horizontal' },
];

/** slug/display-name substring -> icon, checked in order; first match wins. */
const ICON_KEYWORDS: readonly (readonly [string, IconName])[] = [
  ['weav', 'weaving'],
  ['block-print', 'block-printing'],
  ['print', 'block-printing'],
  ['potter', 'pottery'],
  ['terracotta', 'pottery'],
  ['ceramic', 'pottery'],
  ['metal', 'metalwork'],
  ['brass', 'metalwork'],
  ['copper', 'metalwork'],
  ['bronze', 'metalwork'],
  ['bidri', 'metalwork'],
  ['dhokra', 'metalwork'],
  ['wood', 'woodwork'],
  ['carving', 'woodwork'],
  ['embroider', 'embroidery'],
  ['zari', 'embroidery'],
  ['phulkari', 'embroidery'],
  ['chikankari', 'embroidery'],
  ['paint', 'painting'],
  ['pattachitra', 'painting'],
  ['madhubani', 'painting'],
  ['warli', 'painting'],
  ['basket', 'basketry'],
  ['cane', 'basketry'],
  ['jewel', 'jewellery'],
  ['silver', 'jewellery'],
  ['leather', 'leather'],
  ['stone', 'stone'],
  ['marble', 'stone'],
  ['bamboo', 'bamboo'],
  ['bandhani', 'weaving'],
  ['tie-dye', 'weaving'],
  ['tie and dye', 'weaving'],
  ['toy', 'woodwork'],
  ['channapatna', 'woodwork'],
  ['lacquer', 'woodwork'],
];

function iconFor(slug: string, displayName: string): IconName {
  const haystack = `${slug} ${displayName}`.toLowerCase();
  for (const [keyword, icon] of ICON_KEYWORDS) {
    if (haystack.includes(keyword)) return icon;
  }
  return 'more-horizontal';
}

interface RawCraft {
  id: string;
  slug: string;
  display_name: string;
}

function toCraft(raw: RawCraft): Craft {
  return { id: raw.id, slug: raw.slug, displayName: raw.display_name, icon: iconFor(raw.slug, raw.display_name) };
}

/**
 * Live craft ontology for the /register/craft picker. Network first (the
 * ontology can grow/change), falling back to the last successful fetch when
 * offline, then to an empty list -- the caller shows an empty/error state
 * rather than letting an artisan pick a craft this build has never seen a
 * real id for.
 */
export async function loadCrafts(): Promise<Craft[]> {
  try {
    const response = await listCrafts();
    const crafts = (response.crafts ?? []).map((c) => toCraft(c as RawCraft));
    await setCached(CRAFTS_CACHE_KEY, crafts);
    return crafts;
  } catch {
    const cached = await getCached<Craft[]>(CRAFTS_CACHE_KEY);
    return cached ?? [...OFFLINE_FALLBACK_CRAFTS];
  }
}

export interface District {
  id: string;
  name: string;
  state: string;
}

/** ISO 3166-2:IN code for every state name used in DISTRICTS below. */
export const STATE_CODES: Readonly<Record<string, string>> = {
  'Uttar Pradesh': 'IN-UP',
  Rajasthan: 'IN-RJ',
  Gujarat: 'IN-GJ',
  'Madhya Pradesh': 'IN-MP',
  Karnataka: 'IN-KA',
  'Tamil Nadu': 'IN-TN',
  Telangana: 'IN-TG',
  'Andhra Pradesh': 'IN-AP',
  'West Bengal': 'IN-WB',
  Bihar: 'IN-BR',
  Odisha: 'IN-OR',
  'Jammu and Kashmir': 'IN-JK',
};

/** Sorted state names for the free-text-district "which state" fallback picker. */
export const STATES: readonly string[] = Object.keys(STATE_CODES).sort();

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
