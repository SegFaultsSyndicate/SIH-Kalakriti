/**
 * apps/buyer/src/lib/sih-demo-injector.ts
 *
 * SIH Demo Video - Buyer Portal Side of the cross-portal state bridge.
 *
 * Reads from the shared localStorage key written by the artisan app
 * (sih-demo-store.ts on localhost:5173) and converts SihListing objects
 * into ListingSummary objects that the buyer app's existing ListingCard
 * and search fallback can render without any template changes.
 *
 * Also installs a "storage" event listener so switching tabs from the
 * artisan portal to the buyer portal after publishing the Paithani saree
 * causes the feed to refresh reactively, with no page reload needed.
 */

import type { components } from '@kalakriti/api';

type ListingSummary = components['schemas']['ListingSummary'];

const STORE_KEY = 'kalakriti.sih.demo.v4';
const COOKIE_PAITHANI = 'kalakriti_sih_paithani';

export interface SihListing {
  id: string;
  title: string;
  category: string;
  craftSlug: string;
  craftName: string;
  artisanName: string;
  district: string;
  stateCode: string;
  giNo: string;
  imageUrl: string;
  pricePaise: number;
  madeToOrder: boolean;
  state: string;
  publishedAt: string;
}

export interface SihDemoState {
  artisan: {
    id: string;
    name: string;
    craftName: string;
    location: string;
    pehchanId: string;
    clusterName: string;
  };
  listings: SihListing[];
  orders: unknown[];
  paithaniPublished: boolean;
}

export const ESHAAN_BASE_LISTINGS: SihListing[] = [
  {
    id: 'eshaan-1',
    title: 'Kashi Kadwa Pure Silver-Gilt Pit-Loom Mulberry Silk Saree',
    category: 'Weaving',
    craftSlug: 'banarasi-brocade-weaving',
    craftName: 'Banarasi Brocade Weaving',
    artisanName: 'Eshaan',
    district: 'Varanasi, Uttar Pradesh',
    stateCode: 'UP',
    giNo: 'GI-99',
    imageUrl: '/craft-images/weaving_and_looms/banarasi-brocade-weaving.jpg',
    pricePaise: 2450000,
    madeToOrder: true,
    state: 'PUBLISHED',
    publishedAt: '2026-06-10T09:00:00+05:30',
  },
  {
    id: 'eshaan-2',
    title: 'Zari Brocades and Kadwa Weaves of Varanasi',
    category: 'Weaving',
    craftSlug: 'banarasi-brocade-weaving',
    craftName: 'Banarasi Brocade Weaving',
    artisanName: 'Eshaan',
    district: 'Varanasi, Uttar Pradesh',
    stateCode: 'UP',
    giNo: 'GI-99',
    imageUrl: '/craft-images/weaving_and_looms/banarasi_brocade_weaving_01.jpeg',
    pricePaise: 1200000,
    madeToOrder: true,
    state: 'PUBLISHED',
    publishedAt: '2026-07-15T11:30:00+05:30',
  },
  {
    id: 'eshaan-3',
    title: 'Banarasi Brocade Weaving \u2014 Kadwa Detail',
    category: 'Weaving',
    craftSlug: 'banarasi-brocade-weaving',
    craftName: 'Banarasi Brocade Weaving',
    artisanName: 'Eshaan',
    district: 'Varanasi, Uttar Pradesh',
    stateCode: 'UP',
    giNo: 'GI-99',
    imageUrl: '/craft-images/weaving_and_looms/banarasi_brocade_weaving_03.jpeg',
    pricePaise: 1350000,
    madeToOrder: true,
    state: 'PUBLISHED',
    publishedAt: '2026-08-20T10:00:00+05:30',
  },
];

export const PAITHANI_LISTING: SihListing = {
  id: 'eshaan-paithani',
  title: 'White Saree with Maroon Border',
  category: 'Weaving',
  craftSlug: 'banarasi-brocade-weaving',
  craftName: 'Banarasi Brocade Weaving',
  artisanName: 'Eshaan',
  district: 'Varanasi, Uttar Pradesh',
  stateCode: 'UP',
  giNo: 'GI-99',
  imageUrl: '/craft-images/weaving_and_looms/white-saree-maroon-border.jpeg',
  pricePaise: 750000,
  madeToOrder: false,
  state: 'PUBLISHED',
  publishedAt: new Date().toISOString(),
};

function hasPaithaniCookie(): boolean {
  try {
    if (typeof document !== 'undefined') {
      return document.cookie.split(';').some((c) => c.trim().startsWith(`${COOKIE_PAITHANI}=1`));
    }
  } catch { /* ignore */ }
  return false;
}

function readState(): SihDemoState {
  try {
    if (typeof localStorage !== 'undefined') {
      const raw = localStorage.getItem(STORE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw) as SihDemoState;
        if (hasPaithaniCookie() && !parsed.paithaniPublished) {
          parsed.paithaniPublished = true;
          parsed.listings.unshift(PAITHANI_LISTING);
        }
        return parsed;
      }
    }
  } catch { /* ignore */ }

  const paithaniActive = hasPaithaniCookie();
  const listings = paithaniActive
    ? [PAITHANI_LISTING, ...ESHAAN_BASE_LISTINGS]
    : [...ESHAAN_BASE_LISTINGS];

  return {
    artisan: {
      id: 'sih-artisan-eshaan',
      name: 'Eshaan',
      craftName: 'Weaving & Looms',
      location: 'Varanasi, Uttar Pradesh',
      pehchanId: 'UP-VNS-2024-0982',
      clusterName: 'Varanasi Silk Weaver Facility Centre',
    },
    listings,
    orders: [],
    paithaniPublished: paithaniActive,
  };
}

function sihToListingSummary(item: SihListing): ListingSummary {
  return {
    id: item.id,
    product_id: `prod-${item.id}`,
    artisan_id: 'sih-artisan-eshaan',
    artisan_name: item.artisanName,
    craft_name: item.craftName,
    craft_slug: item.craftSlug,
    craft_gi_registration_no: item.giNo,
    gi_certified: true,
    artisan_verified: true,
    artisan_district: item.district,
    artisan_state_code: item.stateCode,
    type: item.madeToOrder ? 'MADE_TO_ORDER' : 'READY_STOCK',
    price: { amount_paise: item.pricePaise, currency_code: 'INR' },
    image_url: item.imageUrl,
    // Single-photo listing: stops the buyer page borrowing same-craft extras for the gallery.
    media: [{ kind: 'IMAGE' as const, url: item.imageUrl }],
    translations: [
      {
        language: 'en',
        title: item.title,
        description: `${item.title} — handcrafted by ${item.artisanName}, master weaver from ${item.district}. GI-certified ${item.craftName}.`,
      },
    ],
  } as ListingSummary;
}

/**
 * Returns all Eshaan listings from the shared store, newest-first,
 * as fully-typed ListingSummary objects ready for ListingCard.
 *
 * If the Paithani listing was published on either localhost:5173 or localhost:5174,
 * it is dynamically prepended at index 0.
 */
export function getDemoListingsForBuyer(categoryFilter?: string): ListingSummary[] {
  const state = readState();
  const listings = state.listings.filter(
    (l) => l.state === 'PUBLISHED' && (!categoryFilter || l.category === categoryFilter) && !l.title.includes('Kashi Kadwa Pure Silver-Gilt Pit-Loom Mulberry Silk Saree'),
  );
  listings.sort((a, b) => new Date(b.publishedAt).getTime() - new Date(a.publishedAt).getTime());
  return listings.map(sihToListingSummary);
}

/** All Eshaan weaving listings (the ones that should appear in search for "Weaving"). */
export function getEshaanWeavingListings(): ListingSummary[] {
  return getDemoListingsForBuyer('Weaving');
}

/** Whether the Paithani listing has been published by the artisan tab. */
export function isPaithaniLive(): boolean {
  return hasPaithaniCookie() || (readState()?.paithaniPublished ?? false);
}

/**
 * Registers a live listener across storage, focus, visibility, and lightweight polling.
 * Returns an unsubscribe function.
 */
export function onDemoStateChange(callback: () => void): () => void {
  let lastPaithani = isPaithaniLive();

  function check(): void {
    const current = isPaithaniLive();
    if (current !== lastPaithani) {
      lastPaithani = current;
      callback();
    }
  }

  function handleStorage(e: StorageEvent): void {
    if (e.key === STORE_KEY) {
      lastPaithani = isPaithaniLive();
      callback();
    }
  }

  let intervalId: ReturnType<typeof setInterval> | undefined;

  if (typeof window !== 'undefined') {
    window.addEventListener('storage', handleStorage);
    window.addEventListener('focus', check);
    document.addEventListener('visibilitychange', check);
    intervalId = setInterval(check, 600);
  }

  return () => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('storage', handleStorage);
      window.removeEventListener('focus', check);
      document.removeEventListener('visibilitychange', check);
      if (intervalId) clearInterval(intervalId);
    }
  };
}

export function resetDemoState(): void {
  try {
    if (typeof localStorage !== 'undefined') {
      localStorage.removeItem(STORE_KEY);
    }
    if (typeof document !== 'undefined') {
      document.cookie = 'kalakriti_sih_paithani=; path=/; max-age=0';
    }
    window.dispatchEvent(new StorageEvent('storage', { key: STORE_KEY, newValue: null }));
  } catch { /* ignore */ }
}

if (typeof window !== 'undefined') {
  (window as unknown as Record<string, unknown>).__resetSihDemo = resetDemoState;
}
