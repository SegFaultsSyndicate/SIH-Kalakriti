/**
 * apps/artisan/src/lib/sih-my-works.ts
 *
 * SIH Demo Video - Artisan "My Works" mock catalog.
 *
 * Provides Eshaan's pre-existing 3 listings as Listing-shaped objects so
 * the existing /listings page renders them immediately on first load,
 * bypassing the backend (which has no published stock in the dev environment).
 *
 * These objects are merged into the cached listings array in the /listings
 * page's load() function before the real fetch is attempted, so even if the
 * API call fails they stay visible.
 */

import type { components } from '@kalakriti/api';
import { ensureDemoState, type SihListing } from './sih-demo-store';

type Listing = components['schemas']['Listing'];

function sihToListing(item: SihListing): Listing {
  return {
    id: item.id,
    artisan_id: 'sih-artisan-eshaan',
    craft_id: item.craftSlug,
    state: item.state as Listing['state'],
    type: item.madeToOrder ? 'MADE_TO_ORDER' : 'READY_STOCK',
    price: { amount_paise: item.pricePaise, currency_code: 'INR' },
    stock_quantity: item.madeToOrder ? undefined : 5,
    translations: [
      {
        language: 'en',
        title: item.title,
        description: `${item.title} — handcrafted by ${item.artisanName}, master weaver from ${item.district}. GI-certified ${item.craftName} (${item.giNo}).`,
      },
    ],
    provenance_id: undefined,
  } as Listing;
}

/**
 * Returns Eshaan's current listings (base catalog + Paithani if published)
 * as fully-shaped Listing objects, newest-first.
 *
 * The artisan /listings page merges these into the display array so the
 * demo catalog is always visible regardless of backend state.
 */
export function getSihMyWorks(): Listing[] {
  const state = ensureDemoState();
  const published = state.listings.filter((l) => l.state === 'PUBLISHED');
  // Newest published first
  published.sort((a, b) => new Date(b.publishedAt).getTime() - new Date(a.publishedAt).getTime());
  return published.map(sihToListing);
}

/**
 * Returns mock orders shaped for the /orders page display.
 * Each order maps to a "direct" lot (single lot = direct order convention).
 */
export interface SihMockOrderRow {
  id: string;
  listingTitle: string;
  buyerName: string;
  quantity: number;
  totalPaise: number;
  state: 'COMPLETED' | 'IN_PROGRESS' | 'OFFERED';
  placedAt: string;
}

export function getSihMockOrders(): SihMockOrderRow[] {
  const state = ensureDemoState();
  return state.orders.map((o) => ({
    id: o.id,
    listingTitle: o.listingTitle,
    buyerName: o.buyerName,
    quantity: o.quantity,
    totalPaise: o.totalPaise,
    state: o.state,
    placedAt: o.placedAt,
  }));
}

/** Financial summary derived from mock orders, for the earnings dashboard. */
export interface SihEarningsSummary {
  totalSalesPaise: number;
  totalOrdersCompleted: number;
  activeOrdersCount: number;
  pendingOrdersCount: number;
  growthPct: number;
  currentMonthPaise: number;
  baselineMonthlyPaise: number;
}

export function getSihEarningsSummary(): SihEarningsSummary {
  const state = ensureDemoState();
  const orders = state.orders;
  const completed = orders.filter((o) => o.state === 'COMPLETED');
  const active = orders.filter((o) => o.state === 'IN_PROGRESS');
  const pending = orders.filter((o) => o.state === 'OFFERED');

  const totalSalesPaise = completed.reduce((acc, o) => acc + o.totalPaise, 0);
  // Current month approximation: orders placed in Sept 2026
  const currentMonthPaise = orders
    .filter((o) => o.placedAt.startsWith('2026-09') && o.state !== 'OFFERED')
    .reduce((acc, o) => acc + o.totalPaise, 0);

  return {
    totalSalesPaise,
    totalOrdersCompleted: completed.length,
    activeOrdersCount: active.length,
    pendingOrdersCount: pending.length,
    growthPct: 142,
    currentMonthPaise: currentMonthPaise || 4400000, // ₹44,000 fallback
    baselineMonthlyPaise: 1200000, // ₹12,000 pre-Kalakriti baseline
  };
}
