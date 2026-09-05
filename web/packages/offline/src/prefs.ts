// packages/offline/src/prefs.ts
//
// Device-local settings: theme, text size, voice. NOT language -- see db.ts.

import { db } from './db';

export async function getPref<T = unknown>(key: string): Promise<T | undefined> {
  const row = await db.prefs.get(key);
  return row?.value as T | undefined;
}

export async function setPref(key: string, value: unknown): Promise<void> {
  await db.prefs.put({ key, value, updatedAt: Date.now() });
}
