// packages/offline/src/conflict.test.ts
import { describe, expect, it } from 'vitest';
import { detectConflict, resolveConflict } from './conflict';
import type { DraftRecord } from './db';

function draft(overrides: Partial<DraftRecord> = {}): DraftRecord {
  return {
    id: 'd1',
    fields: { title: 'mine' },
    mediaIds: [],
    createdAt: 0,
    updatedAt: 0,
    ...overrides,
  };
}

describe('detectConflict', () => {
  it('is null when the draft has never seen a server version', () => {
    expect(detectConflict(draft(), 5)).toBeNull();
  });

  it('is null when the local base is caught up', () => {
    expect(detectConflict(draft({ baseVersion: 3 }), 3)).toBeNull();
  });

  it('flags a conflict when the server has moved past the local base', () => {
    const conflict = detectConflict(draft({ baseVersion: 2 }), 5);
    expect(conflict).toEqual({ draftId: 'd1', localBaseVersion: 2, serverVersion: 5 });
  });
});

describe('resolveConflict', () => {
  it('keep-mine advances baseVersion but leaves fields untouched', () => {
    const resolved = resolveConflict(draft({ baseVersion: 2 }), 'keep-mine', 5, {
      title: 'theirs',
    });
    expect(resolved.fields).toEqual({ title: 'mine' });
    expect(resolved.baseVersion).toBe(5);
  });

  it('use-theirs replaces fields and advances baseVersion', () => {
    const resolved = resolveConflict(draft({ baseVersion: 2 }), 'use-theirs', 5, {
      title: 'theirs',
    });
    expect(resolved.fields).toEqual({ title: 'theirs' });
    expect(resolved.baseVersion).toBe(5);
  });
});
