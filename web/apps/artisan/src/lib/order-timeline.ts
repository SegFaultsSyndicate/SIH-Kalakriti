// apps/artisan/src/lib/order-timeline.ts
//
// Turns GET /orders/{id}/events' raw SSE payloads (see
// services/bff/internal/bff/client/order.go's orderEventToMap) into a
// human-readable narrative line, never a bare state name -- per the batch 10
// brief. The real payload carries artisan_id, not a display name (fulfilment.proto
// has no artisan-name lookup anywhere), so a line can honestly say "an artisan
// accepted a lot" but not "Meera accepted a lot" -- naming this limit here
// rather than silently fabricating a name.
//
// De-duplication for reconnect/backfill: every event carries event_id, and
// the SSE transport (packages/api/src/sse.svelte.ts) already resumes from
// Last-Event-ID -- this module's own job is just to skip an event_id it has
// already narrated, in case the resume point still repeats one.

import type { MessageKey } from '@kalakriti/i18n';

export interface OrderTimelineEvent {
  event_id: string;
  bulk_order_id: string;
  type?: string;
  lot?: { id?: string; artisan_id?: string; quantity?: number; state?: string; progress_pct?: number };
  qc_result?: { passed?: boolean; defects?: { code?: string; severity?: string }[] };
  state?: string;
  payment_split?: { net_total?: { amount_paise?: number } };
}

export interface TimelineLine {
  id: string;
  key: MessageKey;
  params?: Record<string, string>;
  occurredAt: number;
}

const ORDER_STATE_KEY: Record<string, MessageKey> = {
  ALLOCATING: 'timeline.orderAllocating',
  PARTIALLY_ALLOCATED: 'timeline.orderPartiallyAllocated',
  CONFIRMED: 'timeline.orderConfirmed',
  IN_PRODUCTION: 'timeline.orderInProduction',
  AMENDMENT_PENDING: 'timeline.orderAmendmentPending',
  COMPLETED: 'timeline.orderCompleted',
  CANCELLED: 'timeline.orderCancelled',
};

/** Narrates one raw SSE event into a translatable line, or null for a type this module does not render (never a bare dump of the raw payload). */
export function narrate(event: OrderTimelineEvent): TimelineLine | null {
  const base = { id: event.event_id, occurredAt: Date.now() };
  switch (event.type) {
    case 'lot_offered':
      return { ...base, key: 'timeline.lotOffered', params: { quantity: String(event.lot?.quantity ?? '') } };
    case 'lot_accepted':
      return { ...base, key: 'timeline.lotAccepted' };
    case 'lot_declined':
      return { ...base, key: 'timeline.lotDeclined' };
    case 'lot_expired':
      return { ...base, key: 'timeline.lotExpired' };
    case 'lot_progressed':
      return { ...base, key: 'timeline.lotProgressed', params: { pct: String(event.lot?.progress_pct ?? 0) } };
    case 'lot_gave_up':
      return { ...base, key: 'timeline.lotGaveUp' };
    case 'qc_recorded':
      return { ...base, key: event.qc_result?.passed ? 'timeline.qcPassed' : 'timeline.qcFailed' };
    case 'order_state_changed': {
      const key = event.state ? ORDER_STATE_KEY[event.state] : undefined;
      return key ? { ...base, key } : null;
    }
    case 'payment_settled':
      return { ...base, key: 'timeline.paymentSettled' };
    default:
      return null;
  }
}

/** Appends a narrated event to an existing line list, skipping an event_id already present (reconnect backfill safety) and keeping newest last. */
export function appendNarrated(lines: TimelineLine[], event: OrderTimelineEvent): TimelineLine[] {
  if (lines.some((l) => l.id === event.event_id)) return lines;
  const line = narrate(event);
  return line ? [...lines, line] : lines;
}

/** How many `lot_accepted` events have been seen so far -- the running count the "Nine artisans accepted" framing in the brief calls for, computed client-side since no single event carries a running total. */
export function acceptedCount(events: OrderTimelineEvent[]): number {
  return events.filter((e) => e.type === 'lot_accepted').length;
}
