/**
 * apps/buyer/src/lib/demo-catalog.ts
 *
 * One place that turns every demo listing id the buyer app links to into a
 * full ListingSummary, so /listing/{id} renders a product page instead of
 * "not found" while the backend has no published stock:
 *
 *   listing-gi-N / listing-arr-N   home page fallbacks (demo-home-listings.ts)
 *   gi-N                           GI directory (demo-gi-products.ts)
 *   stub-<craft>-N                 search fallbacks (stub-listings.ts)
 *   VoicesReelCarousel slugs       best token match against the stub pieces
 *                                  (explicit REEL_SLUGS list, nothing else)
 *
 * Also generates the Amazon-style extras (ratings, reviews, Q&A, MRP) for
 * those demo pieces. Everything is seeded from the id, never Math.random,
 * so a piece looks the same on every visit and SSR matches hydration.
 *
 * All copy (reviews, Q&A, bullets, spec labels) is `pdp.*` catalogue keys,
 * resolved through the caller's reactive `t`, so it follows the active
 * language like any other UI text.
 */
import type { components } from '@kalakriti/api';
import { locale, type MessageKey, type MessageValues } from '@kalakriti/i18n';
import { RAW_FALLBACK_GI_LISTINGS, RAW_FALLBACK_NEW_ARRIVALS, toFallbackListing } from './demo-home-listings';
import { GI_PRODUCTS, type GIProduct } from './demo-gi-products';
import { allStubListings, stubListingById } from './stub-listings';

type ListingSummary = components['schemas']['ListingSummary'];
type TFn = (key: MessageKey, values?: MessageValues) => string;

// ---------- deterministic randomness ----------

function hash(s: string): number {
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) h = Math.imul(h ^ s.charCodeAt(i), 16777619);
  return h >>> 0;
}

/** mulberry32 -- small seeded PRNG, returns floats in [0, 1). */
function rng(seed: string): () => number {
  let a = hash(seed);
  return () => {
    a = (a + 0x6d2b79f5) | 0;
    let x = Math.imul(a ^ (a >>> 15), 1 | a);
    x = (x + Math.imul(x ^ (x >>> 7), 61 | x)) ^ x;
    return ((x ^ (x >>> 14)) >>> 0) / 4294967296;
  };
}

/** Seeded Fisher-Yates -- sort(() => r() - 0.5) is biased and engine-dependent. */
function shuffle<T>(r: () => number, arr: readonly T[]): T[] {
  const out = [...arr];
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(r() * (i + 1));
    [out[i], out[j]] = [out[j]!, out[i]!];
  }
  return out;
}

function pick<T>(r: () => number, arr: readonly T[]): T {
  return arr[Math.floor(r() * arr.length)]!;
}

// ---------- id resolution ----------

const STATE_NAME_TO_CODE: Record<string, string> = {
  'Uttarakhand': 'UK', 'Bihar': 'BR', 'Gujarat': 'GJ', 'Uttar Pradesh': 'UP', 'Rajasthan': 'RJ',
  'Jammu & Kashmir': 'JK', 'Tamil Nadu': 'TN', 'Karnataka': 'KA', 'Odisha': 'OD', 'West Bengal': 'WB',
  'Madhya Pradesh': 'MP', 'Chhattisgarh': 'CT', 'Assam': 'AS', 'Telangana': 'TG', 'Kerala': 'KL',
};

function slugify(s: string): string {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
}

function giToListing(p: GIProduct, t: TFn): ListingSummary {
  return {
    id: p.id,
    product_id: `prod-${p.id}`,
    artisan_id: `artisan-${p.id}`,
    artisan_name: t(p.artisanNameKey),
    craft_name: t(p.craftNameKey),
    craft_slug: slugify(p.craftName),
    craft_gi_registration_no: p.giRegNo,
    gi_certified: true,
    artisan_verified: true,
    artisan_district: p.state,
    artisan_state_code: STATE_NAME_TO_CODE[p.state] ?? '',
    type: 'READY_STOCK',
    price: { amount_paise: p.price, currency_code: 'INR' },
    image_url: p.image,
    translations: [
      {
        language: locale.code,
        title: t(p.titleKey),
        description: t('stub.listing.descriptionTemplate', {
          craft: t(p.craftNameKey),
          artisan: t(p.artisanNameKey),
          district: p.state,
        }),
      },
    ],
  } as ListingSummary;
}

/**
 * Fills the fields the product page reads but the demo fixtures lack: MTO
 * terms (so the made-to-order panel renders) and a provenance seal (so the
 * page's "sealed" claims match its provenance section). No qr_code -- there
 * is no real record for /verify to look up.
 */
function enrich(l: ListingSummary): ListingSummary {
  const r = rng(`enrich:${l.id}`);
  const woven = /weav|loom|brocade|patola|pashmina|ikat|carpet|chanderi/.test(`${l.craft_slug} ${l.craft_name}`.toLowerCase());
  return {
    ...l,
    made_to_order_terms:
      l.type === 'MADE_TO_ORDER' && !l.made_to_order_terms
        ? {
            lead_time_days: 21 + Math.floor(r() * 25),
            capacity_per_month: 2 + Math.floor(r() * 8),
            advance_pct: pick(r, [25, 30, 40, 50]),
            accepting_orders: true,
          }
        : l.made_to_order_terms,
    provenance: l.provenance ?? {
      sealed_at: new Date(Date.UTC(2026, 0, 1) + Math.floor(r() * 200) * 86400000).toISOString(),
      technique_verdict: { matches: true, claimed: l.craft_name, observed: l.craft_name },
      loom_verdict: woven ? { is_handloom: true } : undefined,
    },
  } as ListingSummary;
}

const REEL_SLUGS = new Set([
  'ajrakh-indigo-stole',
  'banarasi-kadwa-silk',
  'kashmir-pashmina-shawl',
  'dhokra-brass-figurine',
  'pochampally-double-ikat',
  'nizamabad-black-pottery',
]);

export function demoListingById(id: string, t: TFn): ListingSummary | undefined {
  if (!id) return undefined;
  const home = [...RAW_FALLBACK_GI_LISTINGS, ...RAW_FALLBACK_NEW_ARRIVALS].find((r) => r.id === id);
  if (home) return enrich(toFallbackListing(home, t, locale.code));
  const gi = GI_PRODUCTS.find((p) => p.id === id);
  if (gi) return enrich(giToListing(gi, t));
  const stub = stubListingById(id, t);
  if (stub) return enrich(stub);

  // VoicesReelCarousel links by craft slug, not a listing id: map those to the
  // closest stub piece. Kept to this explicit list so an arbitrary worded URL
  // still 404s instead of inventing a product (and its reviews).
  if (!REEL_SLUGS.has(id)) return undefined;
  const tokens = id.toLowerCase().split(/[^a-z0-9]+/).filter((w) => w.length > 3);
  if (tokens.length === 0) return undefined;
  let best: ListingSummary | undefined;
  let bestScore = 0;
  for (const l of allStubListings(t)) {
    const hay = `${l.craft_slug} ${l.craft_name} ${l.translations?.[0]?.title ?? ''}`.toLowerCase();
    const score = tokens.filter((w) => hay.includes(w)).length;
    if (score > bestScore) {
      best = l;
      bestScore = score;
    }
  }
  return best ? enrich({ ...best, id }) : undefined;
}

/** Every demo piece a product page can link to, for the related-items rows. */
export function allDemoListings(t: TFn): ListingSummary[] {
  return [
    ...[...RAW_FALLBACK_GI_LISTINGS, ...RAW_FALLBACK_NEW_ARRIVALS].map((r) => toFallbackListing(r, t, locale.code)),
    ...GI_PRODUCTS.map((p) => giToListing(p, t)),
    ...allStubListings(t),
  ];
}

// ---------- product-page content ----------

export interface Review {
  name: string;
  location: string;
  rating: number;
  title: string;
  body: string;
  date: string;
  helpful: number;
}

export interface DemoSocial {
  rating: number;
  ratingCount: number;
  /** Share of ratings for 5,4,3,2,1 stars, summing to 100. */
  histogram: [number, number, number, number, number];
  mrpPaise: number;
  /** Units left for ready stock; undefined = don't show a scarcity line. */
  stockLeft?: number;
  reviews: Review[];
  qa: { q: string; a: string; by: string }[];
}

const MATERIALS: [RegExp, MessageKey][] = [
  [/pashmina|sozni|shawl/, 'pdp.material.pashmina'],
  [/brocade|banarasi|patola|chanderi|ikat|silk|kanchi/, 'pdp.material.silk'],
  [/carpet|durrie|rug/, 'pdp.material.carpet'],
  [/weav|loom|khadi/, 'pdp.material.cotton'],
  [/ajrakh|block|print|dabu|bagru/, 'pdp.material.dyed'],
  [/embroider|zardozi|chikan|phulkari/, 'pdp.material.embroidery'],
  [/pottery|terracotta|clay|ceramic/, 'pdp.material.clay'],
  [/dhokra|bidri|brass|metal|bell/, 'pdp.material.brass'],
  [/wood|furniture|sheesham|lacquer/, 'pdp.material.wood'],
  [/bamboo|cane|basket|sikki|sabai|grass|moonj/, 'pdp.material.fibre'],
  [/marble|stone|granite|sandstone|soapstone/, 'pdp.material.stone'],
  [/aipan|madhubani|pattachitra|warli|paint/, 'pdp.material.paper'],
  [/jewel|silver|filigree/, 'pdp.material.silver'],
];

const CARE: [RegExp, MessageKey][] = [
  [/silk|brocade|pashmina|patola|embroider|shawl/, 'pdp.care.dryClean'],
  [/cotton|print|weav|khadi|carpet/, 'pdp.care.handWash'],
  [/pottery|clay|stone|marble/, 'pdp.care.wipe'],
  [/brass|metal|dhokra|bidri/, 'pdp.care.polish'],
  [/wood|bamboo|cane|basket|grass/, 'pdp.care.dry'],
  [/paint|paper/, 'pdp.care.frame'],
];

function firstMatch(table: [RegExp, MessageKey][], hay: string, fallback: MessageKey): MessageKey {
  return table.find(([re]) => re.test(hay))?.[1] ?? fallback;
}

/** "About this item" bullets + spec table, built from the listing's real fields -- safe for real listings too. */
export function productDetails(l: ListingSummary, t: TFn): { bullets: string[]; specs: [string, string][] } {
  const hay = `${l.craft_slug ?? ''} ${l.craft_name ?? ''} ${l.translations?.[0]?.title ?? ''}`.toLowerCase();
  const material = t(firstMatch(MATERIALS, hay, 'pdp.material.default'));
  const care = t(firstMatch(CARE, hay, 'pdp.care.default'));
  const origin = [l.artisan_district, l.artisan_state_code].filter(Boolean).join(', ') || t('pdp.spec.india');
  const artisan = l.artisan_name ?? t('pdp.theArtisan');
  const r = rng(`dims:${l.id}`);
  const madeToOrder = l.type === 'MADE_TO_ORDER';
  const gi = l.gi_certified && l.craft_gi_registration_no;

  const bullets = [
    t('pdp.bullet.authentic', { craft: l.craft_name ?? '', origin }),
    gi ? t('pdp.bullet.gi', { giNo: l.craft_gi_registration_no ?? '' }) : t('pdp.bullet.cluster'),
    t('pdp.bullet.material', { material }),
    t(madeToOrder ? 'pdp.bullet.mto' : 'pdp.bullet.ready'),
    t('pdp.bullet.direct', { artisan }),
    ...(l.provenance ? [t('pdp.bullet.sealed')] : []),
  ];

  const specs: [string, string][] = [
    [t('pdp.spec.craft'), l.craft_name ?? '—'],
    [t('pdp.spec.gi'), gi ? (l.craft_gi_registration_no ?? '') : t('pdp.spec.notGi')],
    [t('pdp.spec.origin'), origin],
    [t('pdp.spec.artisan'), artisan],
    [t('pdp.spec.material'), material],
    [t('pdp.spec.dimensions'), t('pdp.spec.dimensionsValue', { w: 20 + Math.floor(r() * 60), h: 15 + Math.floor(r() * 45) })],
    [t('pdp.spec.care'), care],
    [t('pdp.spec.availability'), t(madeToOrder ? 'pdp.spec.mto' : 'pdp.spec.ready')],
    [t('pdp.spec.country'), t('pdp.spec.india')],
  ];
  return { bullets, specs };
}

// Reviewer names and cities are proper nouns -- left as-is in every language.
const NAMES = ['Ananya S.', 'Rohit M.', 'Priya K.', 'Farhan A.', 'Meenakshi R.', 'Arjun P.', 'Sneha D.', 'Kavita J.', 'Vikram N.', 'Lakshmi V.', 'Imran H.', 'Neha G.'];
const CITIES = ['Mumbai', 'Bengaluru', 'New Delhi', 'Pune', 'Hyderabad', 'Kolkata', 'Chennai', 'Jaipur', 'Ahmedabad', 'Lucknow'];
const REVIEW_BANK: { rating: number; title: MessageKey; body: MessageKey }[] = [
  { rating: 5, title: 'pdp.review.1.title', body: 'pdp.review.1.body' },
  { rating: 5, title: 'pdp.review.2.title', body: 'pdp.review.2.body' },
  { rating: 4, title: 'pdp.review.3.title', body: 'pdp.review.3.body' },
  { rating: 5, title: 'pdp.review.4.title', body: 'pdp.review.4.body' },
  { rating: 3, title: 'pdp.review.5.title', body: 'pdp.review.5.body' },
];
const QA_BANK: { q: MessageKey; a: MessageKey; aMto?: MessageKey }[] = [
  { q: 'pdp.qa.1.q', a: 'pdp.qa.1.a' },
  { q: 'pdp.qa.2.q', a: 'pdp.qa.2.aReady', aMto: 'pdp.qa.2.aMto' },
  { q: 'pdp.qa.3.q', a: 'pdp.qa.3.a' },
  { q: 'pdp.qa.4.q', a: 'pdp.qa.4.a' },
];

/** Invented ratings/reviews/Q&A -- only ever call this for demo pieces (see demoListingById). */
export function demoSocial(l: ListingSummary, t: TFn): DemoSocial {
  const r = rng(`social:${l.id}`);
  const price = l.price?.amount_paise ?? 0;
  const gi = GI_PRODUCTS.find((p) => p.id === l.id);
  const mrpPaise = gi?.mrp ?? Math.round((price * (1.12 + r() * 0.25)) / 10000) * 10000;

  const reviewCount = 3 + Math.floor(r() * 2);
  const craft = l.craft_name ?? '';
  const artisan = l.artisan_name ?? t('pdp.theArtisan');
  const pool = shuffle(r, REVIEW_BANK).slice(0, reviewCount);
  const reviews: Review[] = pool.map((rv) => {
    const d = new Date(Date.UTC(2026, 0, 1) + Math.floor(r() * 250) * 86400000);
    return {
      name: pick(r, NAMES),
      location: pick(r, CITIES),
      rating: rv.rating,
      title: t(rv.title),
      body: t(rv.body, { craft, artisan }),
      date: d.toISOString().slice(0, 10),
      helpful: Math.floor(r() * 40),
    };
  });

  const five = 55 + Math.floor(r() * 25);
  const four = Math.floor((100 - five) * 0.6);
  const three = Math.floor((100 - five - four) * 0.6);
  const two = Math.floor((100 - five - four - three) * 0.6);
  const histogram: DemoSocial['histogram'] = [five, four, three, two, 100 - five - four - three - two];
  const rating = Math.round(((5 * five + 4 * four + 3 * three + 2 * two + histogram[4]) / 100) * 10) / 10;

  const mto = l.type === 'MADE_TO_ORDER';
  const qa = shuffle(r, QA_BANK)
    .slice(0, 3)
    .map((x) => ({ q: t(x.q), a: t(mto && x.aMto ? x.aMto : x.a), by: artisan }));

  return {
    rating,
    ratingCount: 12 + Math.floor(r() * 480),
    histogram,
    mrpPaise: Math.max(mrpPaise, price),
    stockLeft: mto ? undefined : 1 + Math.floor(r() * 9),
    reviews,
    qa,
  };
}

/** Same craft first, then a seeded spread of everything else -- excludes the current piece. */
export function relatedListings(l: ListingSummary, t: TFn): { sameCraft: ListingSummary[]; alsoViewed: ListingSummary[] } {
  const all = allDemoListings(t).filter((x) => x.id !== l.id && x.image_url !== l.image_url);
  const craftWord = (l.craft_slug ?? '').split('-')[0] ?? '';
  const sameCraft = all.filter((x) => l.craft_slug && (x.craft_slug === l.craft_slug || (craftWord.length > 3 && x.craft_slug?.includes(craftWord)))).slice(0, 8);
  const r = rng(`related:${l.id}`);
  const rest = shuffle(r, all.filter((x) => !sameCraft.includes(x))).slice(0, 8);
  return { sameCraft, alsoViewed: rest };
}

type ArtisanStorefront = components['schemas']['ArtisanStorefront'];

/**
 * Storefront for a demo artisan, matched the way the product page links to it
 * (/artisan/{lowercased artisan_name}) -- so "sold by" and the artisan badge
 * land on a real page instead of "not found".
 */
export function demoStorefront(slug: string, t: TFn): { artisan: ArtisanStorefront; listings: ListingSummary[] } | undefined {
  const name = decodeURIComponent(slug).toLowerCase();
  const listings = allDemoListings(t).filter((l) => l.artisan_name?.toLowerCase() === name);
  const first = listings[0];
  if (!first) return undefined;
  return {
    artisan: {
      id: first.artisan_id,
      slug,
      display_name: first.artisan_name,
      craft_name: first.craft_name,
      craft_slug: first.craft_slug,
      district: first.artisan_district,
      state_code: first.artisan_state_code,
      verified: true,
      bio: t('pdp.storefront.bio', {
        name: first.artisan_name ?? '',
        craft: first.craft_name ?? '',
        district: first.artisan_district ?? '',
      }),
    },
    listings,
  };
}
