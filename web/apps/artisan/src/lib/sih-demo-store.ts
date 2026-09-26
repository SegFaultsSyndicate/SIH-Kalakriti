/**
 * apps/artisan/src/lib/sih-demo-store.ts
 *
 * SIH Demo Video - Cross-portal shared state bridge.
 *
 * Both the Artisan (localhost:5173) and Buyer (localhost:5174) apps run in the
 * same browser profile, so they share the same localStorage namespace.
 * This module owns:
 *
 *   1. The canonical Eshaan artisan profile fixture.
 *   2. The pre-existing catalog (3 Eshaan listings shown in "My Works" and the
 *      Buyer marketplace from the very first frame of the recording).
 *   3. The stub orders/earnings that make the financial dashboard add up.
 *   4. A publish() function called from the listing wizard that appends a new
 *      listing to the store and fires a "storage" event so the buyer tab,
 *      which is listening via window.addEventListener("storage"), refreshes
 *      its feed without a page reload.
 *
 * localStorage key: "kalakriti.sih.demo"
 */

import { session } from '@kalakriti/api';

// ---------- types ----------

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
  state: 'PUBLISHED' | 'DRAFT' | 'PENDING_ARTISAN_APPROVAL';
  publishedAt: string;
}

export interface SihOrder {
  id: string;
  listingId: string;
  listingTitle: string;
  buyerName: string;
  quantity: number;
  unitPricePaise: number;
  totalPaise: number;
  state: 'COMPLETED' | 'IN_PROGRESS' | 'OFFERED';
  placedAt: string;
  shipping?: {
    phone: string;
    addressLines: string[];
    carrier: string;
    trackingCode: string;
    deliveredAt: string;
    milestones: { label: string; at: string }[];
  };
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
  orders: SihOrder[];
  paithaniPublished: boolean;
}

// ---------- storage key ----------

const STORE_KEY = 'kalakriti.sih.demo.v4';

export function isEshaanDemoAccount(): boolean {
  return session.claims?.phone === '+918779279060';
}

// ---------- canonical fixtures ----------

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

export const ESHAAN_MOCK_ORDERS: SihOrder[] = [
  {
    id: 'sih-order-001',
    listingId: 'eshaan-1',
    listingTitle: 'Kashi Kadwa Pure Silver-Gilt Pit-Loom Mulberry Silk Saree',
    buyerName: 'Priya Mehta',
    quantity: 1,
    unitPricePaise: 2450000,
    totalPaise: 2450000,
    state: 'COMPLETED',
    placedAt: '2026-06-20T14:00:00+05:30',
  },
  {
    id: 'sih-order-002',
    listingId: 'eshaan-1',
    listingTitle: 'Kashi Kadwa Pure Silver-Gilt Pit-Loom Mulberry Silk Saree',
    buyerName: 'Artisans India Store',
    quantity: 1,
    unitPricePaise: 2450000,
    totalPaise: 2450000,
    state: 'IN_PROGRESS',
    placedAt: '2026-09-01T10:00:00+05:30',
  },
  {
    id: 'sih-order-003',
    listingId: 'eshaan-2',
    listingTitle: 'Zari Brocades and Kadwa Weaves of Varanasi',
    buyerName: 'Anita Desai',
    quantity: 1,
    unitPricePaise: 1200000,
    totalPaise: 1200000,
    state: 'COMPLETED',
    placedAt: '2026-09-10T11:00:00+05:30',
    shipping: {
      phone: '+91 98765 43210',
      addressLines: ['House 12, Silk Lane, Sigra', 'Varanasi, Uttar Pradesh 221010'],
      carrier: 'Kalakriti Demo Logistics',
      trackingCode: 'KLT-DEMO-003',
      deliveredAt: '2026-09-16T16:30:00+05:30',
      milestones: [
        { label: 'Order placed', at: '2026-09-10T11:00:00+05:30' },
        { label: 'Handed to courier', at: '2026-09-13T09:15:00+05:30' },
        { label: 'Delivered to Anita', at: '2026-09-16T16:30:00+05:30' },
      ],
    },
  },
  {
    id: 'sih-order-004',
    listingId: 'eshaan-3',
    listingTitle: 'Banarasi Brocade Weaving \u2014 Kadwa Detail',
    buyerName: 'Rahul Verma',
    quantity: 1,
    unitPricePaise: 1350000,
    totalPaise: 1350000,
    state: 'COMPLETED',
    placedAt: '2026-09-15T10:00:00+05:30',
  },
];

// ---------- derived financial figures (all in paise) ----------

/** COMPLETED orders only, toward realised revenue. ₹1,04,000 */
export const MOCK_TOTAL_SALES_PAISE = ESHAAN_MOCK_ORDERS
  .filter((o) => o.state === 'COMPLETED')
  .reduce((acc, o) => acc + o.totalPaise, 0);

export const MOCK_ACTIVE_ORDERS_COUNT = ESHAAN_MOCK_ORDERS
  .filter((o) => o.state === 'IN_PROGRESS' || o.state === 'OFFERED').length;

export const MOCK_GROWTH_PCT = 142;

// ---------- state accessors ----------

export function loadDemoState(): SihDemoState {
  try {
    if (typeof localStorage !== 'undefined') {
      const raw = localStorage.getItem(STORE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw) as Partial<SihDemoState>;
        const sanitized = {
          artisan: parsed.artisan ?? buildDefaultState().artisan,
          listings: Array.isArray(parsed.listings) ? parsed.listings : [],
          orders: Array.isArray(parsed.orders) ? parsed.orders : [],
          paithaniPublished: Boolean(parsed.paithaniPublished),
        } satisfies SihDemoState;

        // Migrate the temporary empty store created by the earlier cleanup so
        // Eshaan's legitimate published catalog comes back immediately.
        if (sanitized.listings.length === 0 && sanitized.orders.length === 0 && !sanitized.paithaniPublished) {
          const defaults = buildDefaultState();
          localStorage.setItem(STORE_KEY, JSON.stringify(defaults));
          return defaults;
        }

        return sanitized;
      }
    }
  } catch { /* ignore */ }
  return buildDefaultState();
}

function buildDefaultState(): SihDemoState {
  return {
    artisan: {
      id: 'sih-artisan-eshaan',
      name: 'Eshaan',
      craftName: 'Weaving & Looms',
      location: 'Varanasi, Uttar Pradesh',
      pehchanId: 'UP-VNS-2024-0982',
      clusterName: 'Varanasi Silk Weaver Facility Centre',
    },
    listings: [...ESHAAN_BASE_LISTINGS],
    orders: [...ESHAAN_MOCK_ORDERS],
    paithaniPublished: false,
  };
}

export function saveDemoState(state: SihDemoState): void {
  try {
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(STORE_KEY, JSON.stringify(state));
      // Cross-tab notification so the buyer portal reacts without a page reload.
      window.dispatchEvent(
        new StorageEvent('storage', { key: STORE_KEY, newValue: JSON.stringify(state) }),
      );
    }
    if (typeof document !== 'undefined') {
      document.cookie = `kalakriti_sih_paithani=${state.paithaniPublished ? '1' : '0'}; path=/; max-age=86400`;
    }
  } catch { /* ignore */ }
}

/** Seeds the store if absent, then returns the current state. */
export function ensureDemoState(): SihDemoState {
  return loadDemoState();
}

/**
 * Called from the artisan listing submission handler.
 * Appends the Paithani listing at the top of the store so the buyer portal
 * immediately shows it at position [0] of the Weaving feed.
 */
export function publishPaithaniListing(): SihDemoState {
  const state = loadDemoState();
  if (!state.paithaniPublished) {
    state.listings.unshift({ ...PAITHANI_LISTING, publishedAt: new Date().toISOString() });
    state.paithaniPublished = true;
    saveDemoState(state);
  }
  if (typeof document !== 'undefined') {
    document.cookie = 'kalakriti_sih_paithani=1; path=/; max-age=86400';
  }
  return state;
}

export function isPaithaniPublished(): boolean {
  try {
    if (typeof document !== 'undefined') {
      if (document.cookie.split(';').some((c) => c.trim().startsWith('kalakriti_sih_paithani=1'))) {
        return true;
      }
    }
  } catch { /* ignore */ }
  return loadDemoState().paithaniPublished;
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
  ensureDemoState();
}

if (typeof window !== 'undefined') {
  (window as unknown as Record<string, unknown>).__resetSihDemo = resetDemoState;
}
