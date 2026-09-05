// packages/api/src/auth.test.ts
import { beforeEach, describe, expect, it } from 'vitest';
import {
  getAccessToken,
  getRefreshToken,
  restoreAccessToken,
  setAccessToken,
  setRefreshToken,
} from './auth';

beforeEach(() => {
  setAccessToken(undefined);
  setRefreshToken(undefined);
});

describe('access token', () => {
  it('is readable synchronously from memory once set', () => {
    setAccessToken('token-1');
    expect(getAccessToken()).toBe('token-1');
  });

  it('is written to its IndexedDB mirror, readable back after what stands in for a reload', async () => {
    setAccessToken('token-2');
    await new Promise((resolve) => setTimeout(resolve, 0)); // let the write-behind put() settle

    // restoreAccessToken reads IndexedDB, not the in-memory variable -- this
    // is the same call a real reload's app-start code makes.
    const restored = await restoreAccessToken();
    expect(restored).toBe('token-2');
    expect(getAccessToken()).toBe('token-2');
  });

  it('clearing the token removes the IndexedDB mirror too', async () => {
    setAccessToken('token-3');
    await new Promise((resolve) => setTimeout(resolve, 0));
    setAccessToken(undefined);
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(await restoreAccessToken()).toBeUndefined();
  });

  it('persists and restores the refresh token alongside the access token', async () => {
    setAccessToken('access-token');
    setRefreshToken('refresh-token');
    await new Promise((resolve) => setTimeout(resolve, 0));

    setAccessToken(undefined);
    await new Promise((resolve) => setTimeout(resolve, 0));
    await restoreAccessToken();

    expect(getRefreshToken()).toBe('refresh-token');
  });
});
