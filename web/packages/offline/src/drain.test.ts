// packages/offline/src/drain.test.ts
import { describe, expect, it, beforeEach } from 'vitest';
import { db } from './db';
import { enqueue } from './outbox';
import { drainOutbox, backoffMs, type SendFn, type SendResult } from './drain';

async function resetDb() {
  await db.drafts.clear();
  await db.outbox.clear();
  await db.media.clear();
  await db.cache.clear();
  await db.prefs.clear();
}

beforeEach(resetDb);

describe('drainOutbox', () => {
  it('sends pending entries FIFO and discards them on success', async () => {
    const first = await enqueue({ kind: 'listing.create', payload: { a: 1 } });
    await new Promise((r) => setTimeout(r, 2));
    const second = await enqueue({ kind: 'listing.create', payload: { a: 2 } });

    const sentOrder: string[] = [];
    const send: SendFn = async (entry) => {
      sentOrder.push(entry.id);
      return { ok: true };
    };

    await drainOutbox(send);

    expect(sentOrder).toEqual([first.id, second.id]);
    expect(await db.outbox.count()).toBe(0);
  });

  it('holds a dependent entry until its dependency is delivered', async () => {
    const upload = await enqueue({ kind: 'media.upload', payload: {} });
    const submit = await enqueue({
      kind: 'listing.submit',
      payload: {},
      dependsOn: [upload.id],
    });

    const sentOrder: string[] = [];
    const send: SendFn = async (entry) => {
      sentOrder.push(entry.id);
      return { ok: true };
    };

    // One pass: submit is skipped the first time through since upload hasn't
    // been discarded yet within the same pass ordering (upload runs first
    // because it was enqueued first, so by the time we reach submit in this
    // same pass its dependency is already gone).
    await drainOutbox(send);

    expect(sentOrder).toEqual([upload.id, submit.id]);
  });

  it('does not send a dependent entry while its dependency is still outstanding', async () => {
    const submit = await enqueue({
      kind: 'listing.submit',
      payload: {},
      dependsOn: ['not-yet-delivered'],
    });
    await db.outbox.add({
      id: 'not-yet-delivered',
      kind: 'media.upload',
      payload: {},
      status: 'pending',
      mediaIds: [],
      attempts: 0,
      createdAt: Date.now() - 1000,
      updatedAt: Date.now() - 1000,
      idempotencyKey: crypto.randomUUID(),
      dependsOn: [],
    });

    const sent: string[] = [];
    const send: SendFn = async (entry) => {
      sent.push(entry.id);
      // Never resolve the dependency -- fail it so it stays in the outbox.
      return { ok: false, retryable: true };
    };

    await drainOutbox(send);

    expect(sent).toContain('not-yet-delivered');
    expect(sent).not.toContain(submit.id);
    const row = await db.outbox.get(submit.id);
    expect(row?.status).toBe('pending');
  });

  it('moves a non-retryable failure straight to blocked, not needsAttention', async () => {
    const entry = await enqueue({ kind: 'listing.submit', payload: {} });
    const send: SendFn = async () => ({ ok: false, retryable: false, error: 'invalid_input' });

    await drainOutbox(send);

    const row = await db.outbox.get(entry.id);
    expect(row?.status).toBe('blocked');
    expect(row?.attempts).toBe(1);
  });

  it('turns a dependent entry into needsAttention when its dependency is terminal', async () => {
    const dependency = await enqueue({ kind: 'media.upload', payload: {} });
    const dependent = await enqueue({
      kind: 'listing.submit',
      payload: {},
      dependsOn: [dependency.id],
    });
    await db.outbox.update(dependency.id, { status: 'blocked' });

    await drainOutbox(async () => ({ ok: true }));

    expect(await db.outbox.get(dependent.id)).toMatchObject({
      status: 'needsAttention',
      lastError: `dependency:${dependency.id}`,
    });
  });

  it('records a thrown send failure as a retryable outbox failure', async () => {
    const entry = await enqueue({ kind: 'listing.submit', payload: {} });

    await drainOutbox(async () => {
      throw new Error('network interrupted');
    });

    expect(await db.outbox.get(entry.id)).toMatchObject({
      status: 'failed',
      attempts: 1,
      lastError: 'network interrupted',
    });
  });

  it('surfaces terminal failure as needsAttention after maxAttempts, never dropping the row', async () => {
    const entry = await enqueue({ kind: 'listing.submit', payload: {} });
    const send: SendFn = async () => ({ ok: false, retryable: true, error: 'timeout' });

    // Drive attempts up to the limit. Backoff is bypassed via a fake clock
    // that always reports "long past due".
    let clock = Date.now();
    const now = () => clock;
    for (let i = 0; i < 6; i++) {
      await drainOutbox(send, { now, maxAttempts: 3 });
      clock += 10 * 60_000;
    }

    const row = await db.outbox.get(entry.id);
    expect(row?.status).toBe('needsAttention');
    expect(row).toBeDefined(); // never silently dropped
  });

  it('respects exponential backoff between retries', async () => {
    const entry = await enqueue({ kind: 'listing.submit', payload: {} });
    let calls = 0;
    const send: SendFn = async () => {
      calls += 1;
      return { ok: false, retryable: true, error: 'x' } satisfies SendResult;
    };

    let clock = Date.now();
    const now = () => clock;

    await drainOutbox(send, { now }); // attempt 1 -> failed
    expect(calls).toBe(1);

    // Immediately draining again: backoff for attempts=1 hasn't elapsed.
    await drainOutbox(send, { now });
    expect(calls).toBe(1);

    clock += backoffMs(1) + 1;
    await drainOutbox(send, { now }); // attempt 2
    expect(calls).toBe(2);

    void entry;
  });

  it('recovers a row orphaned by a killed tab and resends it without duplicating', async () => {
    const entry = await enqueue({ kind: 'listing.create', payload: { title: 'bowl' } });
    // Simulate a tab killed mid-send: row stuck in `syncing`.
    await db.outbox.update(entry.id, { status: 'syncing' });

    const sends: string[] = [];
    const send: SendFn = async (e) => {
      sends.push(e.idempotencyKey);
      return { ok: true };
    };

    await drainOutbox(send);

    // Sent exactly once, using the original idempotency key, and cleared.
    expect(sends).toEqual([entry.idempotencyKey]);
    expect(await db.outbox.get(entry.id)).toBeUndefined();
  });
});
