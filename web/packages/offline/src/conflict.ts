// packages/offline/src/conflict.ts
//
// Conflict detection: local draft vs. a server version fetched elsewhere.
//
// The comparison itself is real and tested. What is NOT real: nothing in
// this codebase currently produces a server version to compare against --
// the BFF spec has no GET /listings/{id} and no version field on a listing.
// `baseVersion` on DraftRecord stays unset until a later batch's fetch path
// populates it. This file builds the mechanism, not the missing input.

import type { DraftRecord } from './db';

export interface Conflict {
  draftId: string;
  localBaseVersion: number;
  serverVersion: number;
}

/** A conflict exists when the draft was edited against a version the server has moved past. */
export function detectConflict(draft: DraftRecord, serverVersion: number): Conflict | null {
  if (draft.baseVersion === undefined) return null;
  if (draft.baseVersion >= serverVersion) return null;
  return { draftId: draft.id, localBaseVersion: draft.baseVersion, serverVersion };
}

export type ConflictResolution = 'keep-mine' | 'use-theirs';

/**
 * Apply a resolution to the draft. "keep mine" just advances baseVersion so
 * the next sync attempt no longer looks stale (the artisan's fields are
 * untouched). "use theirs" replaces fields with the server's and advances
 * baseVersion the same way -- the caller supplies serverFields since this
 * package doesn't own the shape of a listing.
 */
export function resolveConflict(
  draft: DraftRecord,
  resolution: ConflictResolution,
  serverVersion: number,
  serverFields?: Record<string, unknown>,
): DraftRecord {
  return {
    ...draft,
    fields: resolution === 'use-theirs' && serverFields ? serverFields : draft.fields,
    baseVersion: serverVersion,
    updatedAt: Date.now(),
  };
}
