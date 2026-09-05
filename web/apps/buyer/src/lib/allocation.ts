// apps/buyer/src/lib/allocation.ts
//
// Reduces GET /orders/{id}/events' raw SSE payloads (see
// services/bff/internal/bff/client/order.go's orderEventToMap) into the
// allocation view's lot table and a narrated live-region line. Every event
// carries event_id, and packages/api/src/sse.svelte.ts already resumes from
// Last-Event-ID on reconnect -- applyEvent just has to be idempotent against
// a replayed event, which it is: re-applying the same lot snapshot is a
// no-op, and the live-line list is keyed by event_id like
// apps/artisan/src/lib/order-timeline.ts already does for the artisan side.
//
// There is no artisan-name lookup on the event payload itself (fulfilment.proto
// only carries artisan_id) -- names/districts come from a separate resolve
// step the page runs against GET /artisans/{id}/storefront for each unique
// artisan_id seen, real data, not fabricated.

import type { MessageKey } from '@kalakriti/i18n';

export interface RawLot {
  id: string;
  artisan_id: string;
  cluster_id?: string;
  quantity: number;
  state: string;
  progress_pct: number;
  reallocated_from_lot_id?: string;
  decline_reason?: string;
}

export interface RawPaymentLine {
  payee_id: string;
  lot_id: string;
  gross_amount: { amount_paise: number };
  net_amount: { amount_paise: number };
}

export interface RawPaymentSplit {
  gross_total: { amount_paise: number };
  commission_total: { amount_paise: number };
  net_total: { amount_paise: number };
  lines: RawPaymentLine[];
}

export interface RawOrderEvent {
  event_id: string;
  bulk_order_id: string;
  occurred_at?: string;
  type?: string;
  lot?: RawLot;
  state?: string;
  payment_split?: RawPaymentSplit;
}

export interface AllocationState {
  lots: Record<string, RawLot>;
  orderState?: string;
  paymentSplit?: RawPaymentSplit;
  seenEventIds: Set<string>;
}

export function initialAllocationState(lots: RawLot[]): AllocationState {
  const byId: Record<string, RawLot> = {};
  for (const lot of lots) byId[lot.id] = lot;
  return { lots: byId, seenEventIds: new Set() };
}

/** Applies one event to allocation state in place, skipping an event_id already applied. Returns whether it changed anything (false for a pure replay). */
export function applyEvent(state: AllocationState, event: RawOrderEvent): boolean {
  if (state.seenEventIds.has(event.event_id)) return false;
  state.seenEventIds.add(event.event_id);

  if (event.lot) state.lots[event.lot.id] = event.lot;
  if (event.type === 'order_state_changed' && event.state) state.orderState = event.state;
  if (event.type === 'payment_settled' && event.payment_split) state.paymentSplit = event.payment_split;
  return true;
}

export function allocatedQuantity(lots: Record<string, RawLot>): number {
  return Object.values(lots)
    .filter((l) => l.state === 'ACCEPTED' || l.state === 'IN_PRODUCTION' || l.state === 'QC_PENDING' || l.state === 'COMPLETED')
    .reduce((sum, l) => sum + l.quantity, 0);
}

export interface LiveLine {
  id: string;
  key: MessageKey;
  params: Record<string, string>;
}

/** Narrates one event with resolved artisan names, or null for an event type this view doesn't announce. names/districts is keyed by artisan_id. */
export function narrateEvent(
  event: RawOrderEvent,
  names: Record<string, string>,
  districts: Record<string, string>,
): LiveLine | null {
  const artisanId = event.lot?.artisan_id ?? '';
  // Fallback is a literal, not a translation key -- this module stays
  // i18n-free so it can be unit tested without the locale store.
  const artisanLiteral = names[artisanId] ?? 'An artisan';
  const district = districts[artisanId];
  const quantity = String(event.lot?.quantity ?? '');
  const pct = String(event.lot?.progress_pct ?? 0);

  const params = (extra: Record<string, string> = {}): Record<string, string> => ({
    artisan: artisanLiteral ?? '',
    quantity,
    pct,
    ...extra,
  });

  switch (event.type) {
    case 'lot_offered':
      return district
        ? { id: event.event_id, key: 'allocation.live.offered', params: params({ district }) }
        : { id: event.event_id, key: 'allocation.live.offeredNoDistrict', params: params() };
    case 'lot_accepted':
      return { id: event.event_id, key: 'allocation.live.accepted', params: params() };
    case 'lot_declined':
      return { id: event.event_id, key: 'allocation.live.declined', params: params() };
    case 'lot_expired':
      return { id: event.event_id, key: 'allocation.live.expired', params: params() };
    case 'lot_progressed':
      return { id: event.event_id, key: 'allocation.live.progressed', params: params() };
    case 'lot_gave_up':
      return { id: event.event_id, key: 'allocation.live.gaveUp', params: params() };
    case 'qc_recorded':
      return { id: event.event_id, key: 'allocation.live.qcPassed', params: params() };
    case 'order_state_changed':
      return event.state
        ? { id: event.event_id, key: 'allocation.live.orderState', params: { state: event.state } }
        : null;
    case 'payment_settled':
      return { id: event.event_id, key: 'allocation.live.paymentSettled', params: {} };
    default:
      return null;
  }
}
