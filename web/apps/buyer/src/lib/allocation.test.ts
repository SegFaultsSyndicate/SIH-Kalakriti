// apps/buyer/src/lib/allocation.test.ts
import { describe, expect, it } from 'vitest';
import { initialAllocationState, applyEvent, allocatedQuantity, narrateEvent, type RawOrderEvent } from './allocation';

describe('applyEvent', () => {
  it('applies a lot snapshot and skips a replayed event_id', () => {
    const state = initialAllocationState([]);
    const event: RawOrderEvent = {
      event_id: 'evt-1',
      bulk_order_id: 'order-1',
      type: 'lot_offered',
      lot: { id: 'lot-1', artisan_id: 'art-1', quantity: 50, state: 'OFFERED', progress_pct: 0 },
    };
    expect(applyEvent(state, event)).toBe(true);
    expect(state.lots['lot-1'].state).toBe('OFFERED');

    // Reconnect backfill replays the same event -- must be a no-op, not a duplicate.
    expect(applyEvent(state, event)).toBe(false);
    expect(Object.keys(state.lots)).toHaveLength(1);
  });

  it('a dropout and reallocation are two distinct lots, not a silent overwrite', () => {
    const state = initialAllocationState([
      { id: 'lot-1', artisan_id: 'art-1', quantity: 50, state: 'ACCEPTED', progress_pct: 10 },
    ]);
    applyEvent(state, {
      event_id: 'evt-2',
      bulk_order_id: 'order-1',
      type: 'lot_gave_up',
      lot: { id: 'lot-1', artisan_id: 'art-1', quantity: 50, state: 'REALLOCATED', progress_pct: 10 },
    });
    applyEvent(state, {
      event_id: 'evt-3',
      bulk_order_id: 'order-1',
      type: 'lot_offered',
      lot: {
        id: 'lot-2',
        artisan_id: 'art-2',
        quantity: 50,
        state: 'OFFERED',
        progress_pct: 0,
        reallocated_from_lot_id: 'lot-1',
      },
    });
    expect(state.lots['lot-1'].state).toBe('REALLOCATED');
    expect(state.lots['lot-2'].reallocated_from_lot_id).toBe('lot-1');
    expect(allocatedQuantity(state.lots)).toBe(0); // neither lot is ACCEPTED/IN_PRODUCTION/QC_PENDING/COMPLETED right now
  });
});

describe('narrateEvent', () => {
  it('names the specific artisan and district when resolved', () => {
    const line = narrateEvent(
      {
        event_id: 'evt-1',
        bulk_order_id: 'order-1',
        type: 'lot_offered',
        lot: { id: 'lot-1', artisan_id: 'art-1', quantity: 20, state: 'OFFERED', progress_pct: 0 },
      },
      { 'art-1': 'Meera Devi' },
      { 'art-1': 'Barmer' },
    );
    expect(line).toEqual({
      id: 'evt-1',
      key: 'allocation.live.offered',
      params: { artisan: 'Meera Devi', district: 'Barmer', quantity: '20', pct: '0' },
    });
  });

  it('falls back to a literal, not a fabricated name, when unresolved', () => {
    const line = narrateEvent(
      {
        event_id: 'evt-1',
        bulk_order_id: 'order-1',
        type: 'lot_accepted',
        lot: { id: 'lot-1', artisan_id: 'art-unknown', quantity: 20, state: 'ACCEPTED', progress_pct: 0 },
      },
      {},
      {},
    );
    expect(line?.params.artisan).toBe('An artisan');
  });
});
