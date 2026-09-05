// packages/api/src/auth.ts
//
// The access token lives in memory for every read; IndexedDB is only a
// write-behind mirror so a reload doesn't force a fresh OTP flow. Never
// localStorage: it's synchronous (blocks the main thread on a low-end
// device) and any injected script can read it.

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

const STORAGE_ACCESS_KEY = 'kalakriti.auth.access';
const STORAGE_REFRESH_KEY = 'kalakriti.auth.refresh';

export function setAccessToken(token: string | undefined): void {
  accessToken = token;
  if (token === undefined) {
    void db.tokens.delete('access');
    try {
      if (typeof localStorage !== 'undefined') localStorage.removeItem(STORAGE_ACCESS_KEY);
    } catch {}
  } else {
    void db.tokens.put({ id: 'access', token });
    try {
      if (typeof localStorage !== 'undefined') localStorage.setItem(STORAGE_ACCESS_KEY, token);
    } catch {}
  }
}

export function getAccessToken(): string | undefined {
  if (accessToken) return accessToken;
  try {
    if (typeof localStorage !== 'undefined') {
      accessToken = localStorage.getItem(STORAGE_ACCESS_KEY) ?? undefined;
    }
  } catch {}
  return accessToken;
}

let refreshToken: string | undefined;

export function setRefreshToken(token: string | undefined): void {
  refreshToken = token;
  if (token === undefined) {
    void db.tokens.delete('refresh');
    try {
      if (typeof localStorage !== 'undefined') localStorage.removeItem(STORAGE_REFRESH_KEY);
    } catch {}
  } else {
    void db.tokens.put({ id: 'refresh', token });
    try {
      if (typeof localStorage !== 'undefined') localStorage.setItem(STORAGE_REFRESH_KEY, token);
    } catch {}
  }
}

export function getRefreshToken(): string | undefined {
  if (refreshToken) return refreshToken;
  try {
    if (typeof localStorage !== 'undefined') {
      refreshToken = localStorage.getItem(STORAGE_REFRESH_KEY) ?? undefined;
    }
  } catch {}
  return refreshToken;
}

/** Reads the IndexedDB mirror (and synchronous fallback) into memory. Call once at app start, before the first request. */
export async function restoreAccessToken(): Promise<string | undefined> {
  try {
    const [access, refresh] = await Promise.all([db.tokens.get('access'), db.tokens.get('refresh')]);
    if (access?.token) accessToken = access.token;
    if (refresh?.token) refreshToken = refresh.token;
  } catch {}

  if (!accessToken) {
    try {
      if (typeof localStorage !== 'undefined') {
        accessToken = localStorage.getItem(STORAGE_ACCESS_KEY) ?? undefined;
      }
    } catch {}
  }

  if (!refreshToken) {
    try {
      if (typeof localStorage !== 'undefined') {
        refreshToken = localStorage.getItem(STORAGE_REFRESH_KEY) ?? undefined;
      }
    } catch {}
  }

  return accessToken;
}
