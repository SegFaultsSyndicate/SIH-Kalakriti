export function clampRecommendedRange(
  recommendedMinPaise: number,
  recommendedMaxPaise: number,
  floorPaise: number,
): { min: number; max: number } {
  const min = Math.max(recommendedMinPaise, floorPaise);
  return { min, max: Math.max(recommendedMaxPaise, min) };
}
