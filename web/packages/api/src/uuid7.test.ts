// packages/api/src/uuid7.test.ts
import { describe, expect, it } from 'vitest';
import { uuid7 } from './uuid7';

describe('uuid7', () => {
  it('has the canonical 8-4-4-4-12 shape with version 7 and variant 10', () => {
    const id = uuid7();
    expect(id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
  });

  it('is unique across many calls', () => {
    const ids = new Set(Array.from({ length: 1000 }, () => uuid7()));
    expect(ids.size).toBe(1000);
  });

  it('sorts lexicographically with creation order', async () => {
    const first = uuid7();
    await new Promise((resolve) => setTimeout(resolve, 5));
    const second = uuid7();
    expect(first < second).toBe(true);
  });
});
