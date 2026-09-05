// apps/artisan/src/lib/orders.ts
//
// The artisan's commercial life: direct orders and collective lots. Online-
// first for reads, same reasoning as $lib/listings.ts -- GET /orders/{id} is
// real but there is no GET /orders (list) anywhere in the backend (see
// ml_wiring.md's batch 10 section), so /orders shows a locally-cached index
// of every order this device has ever touched (viewed via a notification
// deep link, or acted on), refreshed from the server when online.
//
// Lot responses (accept/decline) are the one write here that MUST work
// offline -- the batch 10 brief calls it out explicitly -- so they go
// through the outbox (order.respond, wired in outbox-send.ts) exactly like
// batch 8's listing writes. Reporting progress and requesting reallocation
// are both real, already-mutating a lot already committed to production;
// they follow the online-first pattern batch 9 established for listing
// management, gated on network.online rather than queued.

import { getCached, setCached, network, enqueue } from '@kalakriti/offline';
import { getOrder, reportProgress as apiReportProgress, requestReallocation as apiRequestReallocation, type components } from '@kalakriti/api';

export type BulkOrder = components['schemas']['BulkOrder'];
export type OrderLot = components['schemas']['OrderLot'];
export type LotState = NonNullable<OrderLot['state']>;

const TOUCHED_KEY = 'orders:touched';
const orderCacheKey = (id: string) => `order:${id}`;

/** Every order id this device has ever fetched or acted on -- the local substitute for a real list-orders endpoint. */
async function touchedOrderIds(): Promise<string[]> {
  return (await getCached<string[]>(TOUCHED_KEY)) ?? [];
}

async function markTouched(orderId: string): Promise<void> {
  const ids = await touchedOrderIds();
  if (!ids.includes(orderId)) await setCached(TOUCHED_KEY, [orderId, ...ids]);
}

/** Fetches one order fresh and remembers it locally, whether or not this device already knew about it. */
export async function fetchOrder(orderId: string): Promise<BulkOrder> {
  const order = await getOrder(orderId);
  await setCached(orderCacheKey(orderId), order);
  await markTouched(orderId);
  return order;
}

/** Last-known copy of one order, for an offline reopen. */
export async function cachedOrder(orderId: string): Promise<BulkOrder | undefined> {
  return getCached<BulkOrder>(orderCacheKey(orderId));
}

/** Every order this device knows about, freshest cache each, for the /orders list. */
export async function cachedOrders(): Promise<BulkOrder[]> {
  const ids = await touchedOrderIds();
  const orders: BulkOrder[] = [];
  for (const id of ids) {
    const order = await cachedOrder(id);
    if (order) orders.push(order);
  }
  return orders;
}

/** Refreshes every locally-known order from the server; best-effort per order, so one failure doesn't block the rest. */
export async function refreshTouchedOrders(): Promise<void> {
  if (!network.online) return;
  const ids = await touchedOrderIds();
  await Promise.all(
    ids.map((id) =>
      fetchOrder(id).catch(() => {
        /* keep the stale cache; the list still shows something. */
      }),
    ),
  );
}

/** This artisan's own lots across every locally-known order, newest offer first. */
export function myLots(orders: BulkOrder[], artisanId: string): OrderLot[] {
  const lots = orders.flatMap((o) => o.lots ?? []).filter((l) => l.artisan_id === artisanId);
  return lots.sort((a, b) => (b.offered_at ?? '').localeCompare(a.offered_at ?? ''));
}

/** Whether a lot needs the artisan's attention right now, for /orders' sort-by-what-needs-action. */
export function needsAction(lot: OrderLot): boolean {
  return lot.state === 'OFFERED' || lot.state === 'QC_FAILED';
}

/**
 * Queues an accept/decline so it works with no connection -- the one write
 * on this screen the brief requires to survive offline, same outbox pattern
 * as batch 8's listing writes. Draining happens the normal way, no different
 * handling needed here.
 */
export async function queueLotResponse(
  lotId: string,
  accept: boolean,
  fields: { promised_ship_date?: string; decline_reason?: string },
): Promise<void> {
  await enqueue({
    kind: 'order.respond',
    payload: { lotId, body: { accept, ...fields } },
  });
}

/** Production progress, or a rework resubmission from QC_FAILED. Online-only -- see this file's own header note. */
export async function reportProgress(
  lotId: string,
  progressPct: number,
  mediaIds: string[],
  note?: string,
): Promise<OrderLot> {
  return apiReportProgress(lotId, { progress_pct: progressPct, media: mediaIds, note });
}

/** The non-punitive give-up path: frees the lot's units for reoffer instead of a silent miss. Online-only. */
export async function requestReallocation(lotId: string, reason: string): Promise<OrderLot> {
  return apiRequestReallocation(lotId, { reason });
}

export { network };
