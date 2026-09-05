import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it } from 'vitest';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { call } from './retry';
import { getAccessToken, setAccessToken } from './auth';
import { refreshSession, setRefreshToken } from './session-refresh';

const server = setupServer();

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

beforeEach(() => {
  setAccessToken('expired-token');
  setRefreshToken(undefined);
});

describe('refreshSession', () => {
  it('refreshes the access token and retries the original request once after a 401', async () => {
    let protectedCalls = 0;
    server.use(
      http.post('http://localhost/api/v1/auth/refresh', async ({ request }) => {
        expect(await request.json()).toEqual({ refresh_token: 'refresh-token' });
        return HttpResponse.json({ access_token: 'fresh-token' });
      }),
      http.get('http://localhost/api/v1/listings', ({ request }) => {
        protectedCalls += 1;
        expect(request.headers.get('Authorization')).toBe(
          protectedCalls === 1 ? 'Bearer expired-token' : 'Bearer fresh-token',
        );
        if (protectedCalls === 1) return new HttpResponse(null, { status: 401 });
        return HttpResponse.json({ listings: [] });
      }),
    );

    const result = await call('/listings', {
      method: 'GET',
      refreshToken: 'refresh-token',
    });

    expect(result).toEqual({ listings: [] });
    expect(protectedCalls).toBe(2);
    expect(getAccessToken()).toBe('fresh-token');
  });

  it('clears the access token when the refresh endpoint rejects the refresh token', async () => {
    server.use(
      http.post('http://localhost/api/v1/auth/refresh', () => new HttpResponse(null, { status: 401 })),
    );

    await expect(refreshSession('bad-refresh-token')).resolves.toBe(false);
    expect(getAccessToken()).toBeUndefined();
  });
});
