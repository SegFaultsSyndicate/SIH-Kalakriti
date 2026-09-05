// packages/api/src/auth.ts
//
// The access token lives in memory for every read; IndexedDB is only a
// write-behind mirror so a reload doesn't force a fresh OTP flow. Never
// localStorage: it's synchronous (blocks the main thread on a low-end
// device) and any injected script can read it.

import Dexie, { type EntityTable } from 'dexie';

interface StoredToken {
  id: 'access';
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

/** Reads the IndexedDB mirror into memory. Call once at app start, before the first request. */
export async function restoreAccessToken(): Promise<string | undefined> {
  const row = await db.tokens.get('access');
  accessToken = row?.token;
  return accessToken;
}

