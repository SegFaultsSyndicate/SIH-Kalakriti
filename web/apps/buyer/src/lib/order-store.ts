// apps/buyer/src/lib/order-store.ts
//
// fulfilment.proto has no ListOrders-for-a-buyer RPC (GetOrder only fetches
// one order by id) -- there is nowhere on the backend to page a buyer's own
// order history from. This device remembers the order ids it has created,
// so /orders can rehydrate each one via the real GetOrder. Documented as a
// gap in ml_wiring.md: history is per-device, not per-account.

import type { components } from '@kalakriti/api';

type BulkOrder = components['schemas']['BulkOrder'];
type ListingSummary = components['schemas']['ListingSummary'];

const KEY = 'kalakriti.buyer.orderIds';
const MOCK_ORDERS_KEY = 'kalakriti.buyer.mockOrders';

export interface StoredMockOrderInput {
  id: string;
  quantity: number;
  deadline?: string;
  budgetBand?: string;
  delivery?: string;
  listing?: ListingSummary;
  createdAt?: string;
}

export function rememberOrder(orderId: string): void {
  try {
    const ids = listRememberedOrders();
    if (!ids.includes(orderId)) {
      ids.unshift(orderId);
      localStorage.setItem(KEY, JSON.stringify(ids.slice(0, 50)));
    }
  } catch {
    // Private window / storage disabled -- the order still succeeded server-side.
  }
}

export function listRememberedOrders(): string[] {
  try {
    const raw = localStorage.getItem(KEY);
    return raw ? (JSON.parse(raw) as string[]) : [];
  } catch {
    return [];
  }
}

export function saveMockOrder(input: StoredMockOrderInput): BulkOrder {
  const listing = input.listing;
  const unitPrice = listing?.price?.amount_paise ?? 245000;
  const totalValue = unitPrice * input.quantity;

  const lot1Qty = Math.max(1, Math.ceil(input.quantity * 0.45));
  const lot2Qty = Math.max(1, Math.floor(input.quantity * 0.35));
  const lot3Qty = Math.max(1, input.quantity - lot1Qty - lot2Qty);

  const mockOrder: BulkOrder = {
    id: input.id,
    buyer_id: 'buyer-current',
    listing_id: listing?.id ?? 'listing-1',
    product_id: listing?.product_id ?? 'prod-1',
    quantity: input.quantity,
    unit_price: { amount_paise: unitPrice, currency_code: 'INR' },
    total_value: { amount_paise: totalValue, currency_code: 'INR' },
    required_by: input.deadline
      ? new Date(`${input.deadline}T00:00:00Z`).toISOString()
      : new Date(Date.now() + 30 * 86400000).toISOString(),
    state: 'ALLOCATING',
    allocated_quantity: lot1Qty + lot2Qty,
    notes: [
      input.budgetBand ? `Budget: ${input.budgetBand}` : '',
      input.delivery ? `Delivery: ${input.delivery}` : '',
    ]
      .filter(Boolean)
      .join(' · '),
    lots: [
      {
        id: `lot-${input.id}-1`,
        artisan_id: listing?.artisan_id ?? 'artisan-kabir',
        cluster_id: 'cluster-varanasi',
        quantity: lot1Qty,
        state: 'IN_PRODUCTION',
        progress_pct: 45,
      },
      {
        id: `lot-${input.id}-2`,
        artisan_id: 'artisan-mir',
        cluster_id: 'cluster-srinagar',
        quantity: lot2Qty,
        state: 'ACCEPTED',
        progress_pct: 15,
      },
      {
        id: `lot-${input.id}-3`,
        artisan_id: 'artisan-prajapati',
        cluster_id: 'cluster-kutch',
        quantity: lot3Qty,
        state: 'OFFERED',
        progress_pct: 0,
      },
    ],
  };

  try {
    const raw = localStorage.getItem(MOCK_ORDERS_KEY);
    const existing: Record<string, BulkOrder> = raw ? JSON.parse(raw) : {};
    existing[input.id] = mockOrder;
    localStorage.setItem(MOCK_ORDERS_KEY, JSON.stringify(existing));
  } catch {
    // Private window / storage disabled
  }

  rememberOrder(input.id);
  return mockOrder;
}

export function getStoredMockOrder(id: string): BulkOrder | undefined {
  try {
    const raw = localStorage.getItem(MOCK_ORDERS_KEY);
    if (!raw) return undefined;
    const existing: Record<string, BulkOrder> = JSON.parse(raw);
    return existing[id];
  } catch {
    return undefined;
  }
}

export function listStoredMockOrders(): BulkOrder[] {
  try {
    const raw = localStorage.getItem(MOCK_ORDERS_KEY);
    if (!raw) return [];
    const existing: Record<string, BulkOrder> = JSON.parse(raw);
    return Object.values(existing);
  } catch {
    return [];
  }
}

export function createFallbackMockOrder(id: string): BulkOrder {
  return {
    id,
    buyer_id: 'buyer-current',
    quantity: 50,
    unit_price: { amount_paise: 245000, currency_code: 'INR' },
    total_value: { amount_paise: 12250000, currency_code: 'INR' },
    required_by: new Date(Date.now() + 30 * 86400000).toISOString(),
    state: 'ALLOCATING',
    allocated_quantity: 40,
    lots: [
      {
        id: `lot-${id}-1`,
        artisan_id: 'artisan-kabir',
        cluster_id: 'cluster-varanasi',
        quantity: 25,
        state: 'IN_PRODUCTION',
        progress_pct: 50,
      },
      {
        id: `lot-${id}-2`,
        artisan_id: 'artisan-mir',
        cluster_id: 'cluster-srinagar',
        quantity: 15,
        state: 'ACCEPTED',
        progress_pct: 20,
      },
      {
        id: `lot-${id}-3`,
        artisan_id: 'artisan-prajapati',
        cluster_id: 'cluster-kutch',
        quantity: 10,
        state: 'OFFERED',
        progress_pct: 0,
      },
    ],
  };
}
