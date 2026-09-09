// @vitest-environment node
//
// apps/artisan/src/lib/ontology.test.ts
//
// See outbox-send.test.ts for why this runs under node: msw's node
// interceptors don't see jsdom's own fetch.
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { setupServer } from 'msw/node';
import { http, HttpResponse } from 'msw';
import { db } from '@kalakriti/offline';

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(async () => {
  await db.prefs.clear();
  // loadCrafts caches in a module-level `let`; a fresh module instance per
  // test is what actually exercises the offline-fallback branch rather than
  // reusing whatever the previous test already resolved.
  vi.resetModules();
});

describe('loadCrafts', () => {
  it('falls back to the last successful fetch when offline', async () => {
    server.use(
      http.get('*/api/v1/crafts', () =>
        HttpResponse.json({ crafts: [{ id: 'craft-1', slug: 'weaving', display_name: 'Weaving' }] }),
      ),
    );
    const { loadCrafts } = await import('./ontology');
    const online = await loadCrafts();
    expect(online).toEqual([
      { id: 'craft-1', slug: 'weaving', displayName: 'Weaving', icon: 'weaving', nameKey: 'craft.weaving.name' },
    ]);

    vi.resetModules();
    server.use(http.get('*/api/v1/crafts', () => HttpResponse.error()));
    const { loadCrafts: loadCraftsAgain } = await import('./ontology');
    await expect(loadCraftsAgain()).resolves.toEqual(online);
  });

  it('rejects when neither the network nor a prior fetch has an answer', async () => {
    server.use(http.get('*/api/v1/crafts', () => HttpResponse.error()));
    const { loadCrafts } = await import('./ontology');
    await expect(loadCrafts()).rejects.toBeTruthy();
  });
});
