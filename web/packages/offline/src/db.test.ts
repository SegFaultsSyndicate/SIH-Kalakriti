// packages/offline/src/db.test.ts
import Dexie from 'dexie';
import { describe, expect, it } from 'vitest';
import { db, KalakritiDatabase } from './db';

describe('KalakritiDatabase', () => {
  it('survives a reload: a draft written before close is readable after reopening', async () => {
    await db.drafts.put({
      id: 'draft-1',
      fields: { title: 'Silk scarf' },
      mediaIds: [],
      createdAt: Date.now(),
      updatedAt: Date.now(),
    });
    db.close();

    const reopened = new KalakritiDatabase(db.name);
    const draft = await reopened.drafts.get('draft-1');
    expect(draft?.fields.title).toBe('Silk scarf');
    reopened.close();
  });

  it('backfills dependsOn on outbox rows written under the v1 schema', async () => {
    const name = 'kalakriti-v1-fixture';
    const v1 = new Dexie(name);
    v1.version(1).stores({
      drafts: 'id, updatedAt, remoteId',
      outbox: 'id, status, createdAt, draftId, [status+createdAt]',
      media: 'id, uploaded, capturedAt',
    });
    await v1.table('outbox').add({
      id: 'legacy-1',
      kind: 'listing.create',
      payload: {},
      status: 'pending',
      mediaIds: [],
      attempts: 0,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      idempotencyKey: 'k1',
    });
    v1.close();

    const upgraded = new KalakritiDatabase(name);
    const row = await upgraded.outbox.get('legacy-1');
    expect(row?.dependsOn).toEqual([]);
    upgraded.close();
    await Dexie.delete(name);
  });
});
