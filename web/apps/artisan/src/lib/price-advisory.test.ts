import { describe, expect, it } from 'vitest';
import { clampRecommendedRange } from './price-advisory';

describe('clampRecommendedRange', () => {
  it('never displays a recommendation below the cost floor', () => {
    expect(clampRecommendedRange(900, 1200, 1000)).toEqual({ min: 1000, max: 1200 });
  });

  it('raises the upper bound when a floor exceeds both server bounds', () => {
    expect(clampRecommendedRange(900, 950, 1000)).toEqual({ min: 1000, max: 1000 });
  });
});
