import { describe, expect, it } from 'vitest';
import { applyOptimistic } from './optimistic';

describe('applyOptimistic', () => {
  it('rolls back when enqueueing the optimistic intent fails', async () => {
    let value = 0;
    await expect(
      applyOptimistic({
        apply: () => (value = 1),
        rollback: () => (value = 0),
        enqueue: async () => {
          throw new Error('quota exceeded');
        },
      }),
    ).rejects.toThrow('quota exceeded');
    expect(value).toBe(0);
  });
});
