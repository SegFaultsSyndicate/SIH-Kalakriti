// @vitest-environment node
//
// apps/artisan/src/lib/outbox-send.test.ts
//
// msw's node interceptors patch Node's own fetch; jsdom (the app's default
// test environment) ships its own fetch implementation that they don't see,
// which made every mocked request fall through to a real (failing) network
// call. No DOM is needed here, so node is both correct and faster.
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { setupServer } from 'msw/node';
import { http, HttpResponse } from 'msw';
import { db, enqueue, type OutboxEntry } from '@kalakriti/offline';
import { sendOutboxEntry } from './outbox-send';
import { getArtisanId } from './registration';

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => {
  server.resetHandlers();
  vi.unstubAllEnvs();
});
afterAll(() => server.close());

beforeEach(async () => {
  await db.drafts.clear();
  await db.outbox.clear();
  await db.media.clear();
});

function entry(overrides: Partial<OutboxEntry> & { kind: OutboxEntry['kind'] }): Promise<OutboxEntry> {
  return enqueue({
    kind: overrides.kind,
    payload: overrides.payload,
    draftId: overrides.draftId,
    mediaIds: overrides.mediaIds,
    dependsOn: overrides.dependsOn,
  });
}

describe('sendMediaUpload', () => {
  it('requests an upload url, PUTs the raw bytes, confirms, and records the remote id', async () => {
    await db.media.add({
      id: 'm1',
      blob: new Blob(['bytes']),
      mimeType: 'image/jpeg',
      byteSize: 5,
      uploaded: false,
      capturedAt: Date.now(),
      kind: 'photo',
    });
    let putBody: string | undefined;

    server.use(
      http.post('*/api/v1/media/upload-url', () =>
        HttpResponse.json({ media_id: 'remote-m1', upload_url: 'http://localhost/storage/m1' }),
      ),
      http.put('http://localhost/storage/m1', async ({ request }) => {
        putBody = await request.text();
        return new HttpResponse(null, { status: 200 });
      }),
      http.post('*/api/v1/media/remote-m1/confirm', () => new HttpResponse(null, { status: 204 })),
    );

    const e = await entry({ kind: 'media.upload', payload: { localMediaId: 'm1' } });
    const result = await sendOutboxEntry(e);

    expect(result).toEqual({ ok: true });
    expect(putBody).toBe('bytes');
    const media = await db.media.get('m1');
    expect(media?.remoteId).toBe('remote-m1');
    expect(media?.uploaded).toBe(true);
  });

  it('is retryable, not blocked, when the local media record is missing -- no, missing is unrecoverable', async () => {
    const e = await entry({ kind: 'media.upload', payload: { localMediaId: 'does-not-exist' } });
    const result = await sendOutboxEntry(e);
    expect(result.ok).toBe(false);
    expect(result.retryable).toBe(false);
  });
});

describe('sendListingCreate', () => {
  it('resolves local media ids to remote ids and writes the returned listing id onto the draft', async () => {
    await db.drafts.add({
      id: 'd1',
      fields: {},
      mediaIds: ['m1'],
      createdAt: Date.now(),
      updatedAt: Date.now(),
    });
    await db.media.add({
      id: 'm1',
      blob: new Blob(['x']),
      mimeType: 'image/jpeg',
      byteSize: 1,
      uploaded: true,
      capturedAt: Date.now(),
      kind: 'photo',
      remoteId: 'remote-m1',
    });

    server.use(
      http.post('*/api/v1/listings', async ({ request }) => {
        const body = (await request.json()) as { media?: string[] };
        expect(body.media).toEqual(['remote-m1']);
        return HttpResponse.json({ listing_id: 'listing-1' }, { status: 201 });
      }),
    );

    const e = await entry({
      kind: 'listing.create',
      draftId: 'd1',
      payload: { draftId: 'd1', localMediaIds: ['m1'], body: { craft_id: 'weaving', min_order_quantity: 1 } },
    });
    const result = await sendOutboxEntry(e);

    expect(result).toEqual({ ok: true });
    expect((await db.drafts.get('d1'))?.remoteId).toBe('listing-1');
  });

  it('stays retryable (not blocked) if a referenced media id has no remote id yet', async () => {
    await db.media.add({
      id: 'm1',
      blob: new Blob(['x']),
      mimeType: 'image/jpeg',
      byteSize: 1,
      uploaded: false,
      capturedAt: Date.now(),
      kind: 'photo',
    });
    const e = await entry({
      kind: 'listing.create',
      draftId: 'd1',
      payload: { draftId: 'd1', localMediaIds: ['m1'], body: { craft_id: 'weaving' } },
    });
    const result = await sendOutboxEntry(e);
    expect(result.ok).toBe(false);
    expect(result.retryable).toBe(true);
  });
});

describe('sendListingUpdate / submit / approve', () => {
  it('resolves the remote listing id from the draft, not the payload', async () => {
    await db.drafts.add({
      id: 'd1',
      fields: {},
      mediaIds: [],
      remoteId: 'listing-1',
      createdAt: Date.now(),
      updatedAt: Date.now(),
    });

    server.use(
      http.patch('*/api/v1/listings/listing-1', () => new HttpResponse(null, { status: 204 })),
      http.post('*/api/v1/listings/listing-1/submit', () => new HttpResponse(null, { status: 204 })),
      http.post('*/api/v1/listings/listing-1/approve', () => new HttpResponse(null, { status: 204 })),
    );

    const update = await entry({ kind: 'listing.update', draftId: 'd1', payload: { draftId: 'd1', body: {} } });
    expect(await sendOutboxEntry(update)).toEqual({ ok: true });

    const submit = await entry({ kind: 'listing.submit', draftId: 'd1', payload: { draftId: 'd1' } });
    expect(await sendOutboxEntry(submit)).toEqual({ ok: true });

    const approve = await entry({ kind: 'listing.approve', draftId: 'd1', payload: { draftId: 'd1' } });
    expect(await sendOutboxEntry(approve)).toEqual({ ok: true });
  });

  it('is retryable when the draft has no remote id yet (create has not landed)', async () => {
    await db.drafts.add({ id: 'd2', fields: {}, mediaIds: [], createdAt: Date.now(), updatedAt: Date.now() });
    const submit = await entry({ kind: 'listing.submit', draftId: 'd2', payload: { draftId: 'd2' } });
    const result = await sendOutboxEntry(submit);
    expect(result).toEqual({ ok: false, retryable: true, error: expect.any(String) });
  });
});

describe('sendProfileUpdate', () => {
  it('reports failure, and never fabricates an artisan id, when registration fails', async () => {
    // A previous version of this sender treated ANY registerArtisan
    // failure as success in dev mode, storing a fabricated
    // `artisan-${Date.now()}` as this app's own identity -- every later
    // authenticated request (follower-count, artisans/me, listing
    // creation) then carried an id no backend row could ever match. See
    // WIRING_AUDIT_PLAN.md F-4.
    // Independent of whatever VITE_USE_MOCKS a dev's own .env.local sets --
    // this test asserts the real-backend path regardless of local machine state.
    vi.stubEnv('VITE_USE_MOCKS', '0');
    server.use(http.post('*/api/v1/artisans', () => new HttpResponse(null, { status: 500 })));

    const update = await entry({ kind: 'profile.update', payload: { display_name: 'Test' } });
    const result = await sendOutboxEntry(update);

    expect(result.ok).toBe(false);
    expect(await getArtisanId()).toBeUndefined();
  });

  it('stores the real artisan id on success', async () => {
    server.use(http.post('*/api/v1/artisans', () => HttpResponse.json({ artisan_id: 'real-artisan-1' })));

    const update = await entry({ kind: 'profile.update', payload: { display_name: 'Test' } });
    const result = await sendOutboxEntry(update);

    expect(result).toEqual({ ok: true });
    expect(await getArtisanId()).toBe('real-artisan-1');
  });
});
