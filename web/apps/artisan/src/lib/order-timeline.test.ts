// apps/artisan/src/lib/order-timeline.test.ts
import { describe, expect, it } from 'vitest';
import { narrate, appendNarrated, acceptedCount, type OrderTimelineEvent } from './order-timeline';

function event(overrides: Partial<OrderTimelineEvent>): OrderTimelineEvent {
  return { event_id: 'e1', bulk_order_id: 'o1', ...overrides };
}

describe('narrate', () => {
  it('renders a known event type as a translation key, never the raw type string', () => {
    const line = narrate(event({ type: 'lot_accepted' }));
    expect(line?.key).toBe('timeline.lotAccepted');
  });

  it('renders qc_recorded as pass or fail depending on the result', () => {
    expect(narrate(event({ type: 'qc_recorded', qc_result: { passed: true } }))?.key).toBe('timeline.qcPassed');
    expect(narrate(event({ type: 'qc_recorded', qc_result: { passed: false } }))?.key).toBe('timeline.qcFailed');
  });

  it('maps a known order state to its own key, never a bare state dump', () => {
    const line = narrate(event({ type: 'order_state_changed', state: 'CONFIRMED' }));
    expect(line?.key).toBe('timeline.orderConfirmed');
  });

  it('returns null for an unrecognised order state rather than fabricating a line', () => {
    expect(narrate(event({ type: 'order_state_changed', state: 'SOMETHING_NEW' }))).toBeNull();
  });

  it('returns null for an unknown event type', () => {
    expect(narrate(event({ type: 'not_a_real_event' }))).toBeNull();
  });
});

describe('appendNarrated', () => {
  it('skips an event_id already present -- the reconnect/backfill dedupe guarantee', () => {
    const first = appendNarrated([], event({ event_id: 'e1', type: 'lot_accepted' }));
    expect(first).toHaveLength(1);
    const second = appendNarrated(first, event({ event_id: 'e1', type: 'lot_accepted' }));
    expect(second).toHaveLength(1);
  });

  it('appends a new event_id', () => {
    const first = appendNarrated([], event({ event_id: 'e1', type: 'lot_accepted' }));
    const second = appendNarrated(first, event({ event_id: 'e2', type: 'lot_declined' }));
    expect(second).toHaveLength(2);
  });
});

describe('acceptedCount', () => {
  it('counts only lot_accepted events', () => {
    const events = [
      event({ event_id: '1', type: 'lot_offered' }),
      event({ event_id: '2', type: 'lot_accepted' }),
      event({ event_id: '3', type: 'lot_accepted' }),
      event({ event_id: '4', type: 'lot_declined' }),
    ];
    expect(acceptedCount(events)).toBe(2);
  });
});
