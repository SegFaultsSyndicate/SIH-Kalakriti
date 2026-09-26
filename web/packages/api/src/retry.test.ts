// packages/api/src/retry.test.ts
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it } from 'vitest';
import { setupServer } from 'msw/node';
import { http, HttpResponse } from 'msw';
import { call } from './retry';
import { ApiError } from './transport';
import { getAccessToken, setAccessToken } from './auth';
import { setUnauthorizedHandler } from './config';
import { setRefreshToken } from './session-refresh';

const server = setupServer();

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => {
  setAccessToken(undefined);
  setRefreshToken(undefined);
  setUnauthorizedHandler(undefined);
});

describe('call retry behaviour', () => {
  it('retries a gateway failure (503) and succeeds once the server recovers', async () => {
    let attempts = 0;
    server.use(
      http.post('http://localhost/api/v1/listings', () => {
        attempts += 1;
        if (attempts < 3) return new HttpResponse(null, { status: 503 });
        return HttpResponse.json({ listing_id: 'l1' }, { status: 201 });
      }),
    );

    const result = await call('/listings', {
      method: 'POST',
      body: { craft_id: 'weaving' },
      backoffMs: 1,
    });

    expect(attempts).toBe(3);
    expect(result).toEqual({ listing_id: 'l1' });
  });

  it('does not retry a bare 500 -- only 502/503/504 and network failures are retryable', async () => {
    let attempts = 0;
    server.use(
      http.post('http://localhost/api/v1/listings', () => {
        attempts += 1;
        return new HttpResponse(null, { status: 500 });
      }),
    );

    await expect(call('/listings', { method: 'POST', body: {}, backoffMs: 1 })).rejects.toMatchObject({
      status: 500,
    });
    expect(attempts).toBe(1);
  });

  it('gives up after the configured number of retries', async () => {
    let attempts = 0;
    server.use(
      http.post('http://localhost/api/v1/listings', () => {
        attempts += 1;
        return new HttpResponse(null, { status: 503 });
      }),
    );

    await expect(
      call('/listings', { method: 'POST', body: {}, retries: 1, backoffMs: 1 }),
    ).rejects.toBeInstanceOf(ApiError);
    expect(attempts).toBe(2); // first attempt + 1 retry
  });

  it('reuses the same idempotency key across every retry of a write', async () => {
    const seenKeys: (string | null)[] = [];
    server.use(
      http.post('http://localhost/api/v1/listings', ({ request }) => {
        seenKeys.push(request.headers.get('Idempotency-Key'));
        if (seenKeys.length < 3) return new HttpResponse(null, { status: 503 });
        return HttpResponse.json({ listing_id: 'l2' }, { status: 201 });
      }),
    );

    await call('/listings', { method: 'POST', body: {}, backoffMs: 1 });

    expect(seenKeys).toHaveLength(3);
    expect(seenKeys[0]).not.toBeNull();
    expect(new Set(seenKeys).size).toBe(1);
  });

  it('auto-attaches a UUIDv7 idempotency key to POST/PATCH/DELETE but not GET', async () => {
    let postKey: string | null = null;
    let getKey: string | null | undefined;
    server.use(
      http.post('http://localhost/api/v1/listings', ({ request }) => {
        postKey = request.headers.get('Idempotency-Key');
        return HttpResponse.json({ listing_id: 'l3' }, { status: 201 });
      }),
      http.get('http://localhost/api/v1/listings', ({ request }) => {
        getKey = request.headers.get('Idempotency-Key');
        return HttpResponse.json({ listings: [] }, { status: 200 });
      }),
    );

    await call('/listings', { method: 'POST', body: {} });
    await call('/listings', { method: 'GET' });

    expect(postKey).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
    expect(getKey).toBeNull();
  });

  it('a caller-supplied idempotency key is never overridden', async () => {
    let seenKey: string | null = null;
    server.use(
      http.post('http://localhost/api/v1/listings', ({ request }) => {
        seenKey = request.headers.get('Idempotency-Key');
        return HttpResponse.json({ listing_id: 'l4' }, { status: 201 });
      }),
    );

    await call('/listings', { method: 'POST', body: {}, idempotencyKey: 'outbox-key-1' });

    expect(seenKey).toBe('outbox-key-1');
  });

  it('does not retry a non-retryable client error', async () => {
    let attempts = 0;
    server.use(
      http.post('http://localhost/api/v1/listings', () => {
        attempts += 1;
        return HttpResponse.json({ error: 'invalid_input', message: 'bad' }, { status: 400 });
      }),
    );

    await expect(call('/listings', { method: 'POST', body: {} })).rejects.toMatchObject({
      status: 400,
    });
    expect(attempts).toBe(1);
  });

  it('calls the unauthorized handler once on a 401 and does not retry it', async () => {
    setAccessToken('stale-token');
    let attempts = 0;
    server.use(
      http.get('http://localhost/api/v1/listings', () => {
        attempts += 1;
        return new HttpResponse(null, { status: 401 });
      }),
    );

    const seenPaths: string[] = [];
    setUnauthorizedHandler((path) => seenPaths.push(path));

    await expect(call('/listings', { method: 'GET' })).rejects.toMatchObject({ status: 401 });

    expect(attempts).toBe(1);
    expect(seenPaths).toEqual(['/listings']);
    expect(getAccessToken()).toBe('stale-token'); // no refresh token is available
  });

  it('propagates a 401 unchanged when no unauthorized handler is set', async () => {
    server.use(
      http.get('http://localhost/api/v1/listings', () => new HttpResponse(null, { status: 401 })),
    );

    await expect(call('/listings', { method: 'GET' })).rejects.toMatchObject({ status: 401 });
  });

  it('keeps local mock sessions out of real auth and login redirects', async () => {
    const mockToken = 'eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJyb2xlIjoiTUlOSVNUUlkiLCJleHAiOjQ3MDAwMDAwMDB9.mock';
    const seenPaths: string[] = [];
    let authorization: string | null = 'unexpected';
    setAccessToken(mockToken);
    setRefreshToken('mock-refresh-token');
    setUnauthorizedHandler((path) => seenPaths.push(path));
    server.use(
      http.get('http://localhost/api/v1/listings', ({ request }) => {
        authorization = request.headers.get('Authorization');
        return new HttpResponse(null, { status: 401 });
      }),
    );

    await expect(call('/listings', { method: 'GET' })).rejects.toMatchObject({ status: 401 });

    expect(authorization).toBeNull();
    expect(seenPaths).toEqual([]);
    expect(getAccessToken()).toBe(mockToken);
  });
});
