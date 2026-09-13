// packages/api/src/auth.ts
//
// The access token lives in memory for every read; IndexedDB is only a
// write-behind mirror so a reload doesn't force a fresh OTP flow. Never
// localStorage: it's synchronous (blocks the main thread on a low-end
// device) and any injected script can read it. Reads are memory-only, so
// every app layout must await restoreAccessToken() before its first request.

import Dexie, { type EntityTable } from 'dexie';

interface StoredToken {
  id: 'access' | 'refresh';
  token: string;
}

const db = new Dexie('kalakriti-auth') as Dexie & {
  tokens: EntityTable<StoredToken, 'id'>;
};
db.version(1).stores({ tokens: 'id' });

let accessToken: string | undefined;

export function setAccessToken(token: string | undefined): void {
  accessToken = token;
  if (token === undefined) {
    void db.tokens.delete('access');
  } else {
    void db.tokens.put({ id: 'access', token });
  }
}

export function getAccessToken(): string | undefined {
  return accessToken;
}

let refreshToken: string | undefined;

export function setRefreshToken(token: string | undefined): void {
  refreshToken = token;
  if (token === undefined) {
    void db.tokens.delete('refresh');
  } else {
    void db.tokens.put({ id: 'refresh', token });
  }
}

export function getRefreshToken(): string | undefined {
  return refreshToken;
}

/** Reads the IndexedDB mirror into memory. Call once at app start, before the first request. */
export async function restoreAccessToken(): Promise<string | undefined> {
  try {
    const [access, refresh] = await Promise.all([db.tokens.get('access'), db.tokens.get('refresh')]);
    if (access?.token) accessToken = access.token;
    if (refresh?.token) refreshToken = refresh.token;
  } catch {}

  return accessToken;
}
