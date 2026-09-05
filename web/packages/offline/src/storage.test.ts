// packages/offline/src/storage.test.ts
import { beforeEach, describe, expect, it } from 'vitest';
import { db } from './db';
import { enqueue } from './outbox';
import { evictOldestCachedMedia } from './storage';

beforeEach(async () => {
  await db.drafts.clear();
  await db.outbox.clear();
  await db.media.clear();
});

function blob() {
  return new Blob(['x']);
}

describe('evictOldestCachedMedia', () => {
  it('drops uploaded, unreferenced media oldest-first', async () => {
    await db.media.bulkAdd([
      { id: 'm1', blob: blob(), mimeType: 'image/jpeg', byteSize: 1, uploaded: true, capturedAt: 1 },
      { id: 'm2', blob: blob(), mimeType: 'image/jpeg', byteSize: 1, uploaded: true, capturedAt: 2 },
    ]);

    const freed = await evictOldestCachedMedia();

    expect(freed).toBe(2);
    expect(await db.media.count()).toBe(0);
  });

  it('never evicts media still referenced by a queued outbox entry', async () => {
    await db.media.add({
      id: 'm1',
      blob: blob(),
      mimeType: 'image/jpeg',
      byteSize: 1,
      uploaded: true,
      capturedAt: 1,
    });
    await enqueue({ kind: 'listing.submit', payload: {}, mediaIds: ['m1'] });

    const freed = await evictOldestCachedMedia();

    expect(freed).toBe(0);
    expect(await db.media.get('m1')).toBeDefined();
  });

  it('never evicts media not yet uploaded', async () => {
    await db.media.add({
      id: 'm1',
      blob: blob(),
      mimeType: 'image/jpeg',
      byteSize: 1,
      uploaded: false,
      capturedAt: 1,
    });

    const freed = await evictOldestCachedMedia();

    expect(freed).toBe(0);
  });
});
