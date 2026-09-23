/**
 * apps/buyer/src/lib/fuzzy-search.ts
 *
 * Small, dependency-free fuzzy matching for client-side search boxes
 * (catalog's craft filter, search's local suggestion fallback). Not a
 * replacement for search-svc's real query understanding -- just enough
 * edit-distance tolerance that "potery" or "wodwork" still finds the right
 * craft instead of nothing, for the boxes that only ever filtered by exact
 * substring before.
 */

export function normalize(value: string): string {
  return value.trim().toLowerCase();
}

function levenshtein(a: string, b: string): number {
  if (a === b) return 0;
  if (a.length === 0) return b.length;
  if (b.length === 0) return a.length;

  let prev = new Array<number>(b.length + 1);
  let curr = new Array<number>(b.length + 1);
  for (let j = 0; j <= b.length; j++) prev[j] = j;

  for (let i = 1; i <= a.length; i++) {
    curr[0] = i;
    for (let j = 1; j <= b.length; j++) {
      curr[j] =
        a[i - 1] === b[j - 1]
          ? prev[j - 1]
          : 1 + Math.min(prev[j], curr[j - 1], prev[j - 1]);
    }
    [prev, curr] = [curr, prev];
  }
  return prev[b.length];
}

/** Edit distance tolerated for a query of this length: short queries need a
 * near-exact match, longer ones can absorb a couple of typos. */
function toleranceFor(length: number): number {
  if (length <= 3) return 0;
  if (length <= 5) return 1;
  return 2;
}

/**
 * True if `query` matches `text`: a direct substring hit, or one of
 * `text`'s words is within edit-distance tolerance of `query` -- handles
 * partial input ("pott" -> "pottery") and simple misspellings
 * ("potery"/"wodwork") without a search-index dependency.
 */
export function fuzzyIncludes(text: string, query: string): boolean {
  const q = normalize(query);
  if (!q) return true;
  const t = normalize(text);
  if (t.includes(q)) return true;

  const tolerance = toleranceFor(q.length);
  if (tolerance === 0) return false;

  const words = t.split(/[^\p{L}\p{N}]+/u).filter(Boolean);
  return words.some((word) => {
    if (Math.abs(word.length - q.length) > tolerance + 2) return false;
    if (levenshtein(q, word) <= tolerance) return true;
    // Also compare against the word's own query-length prefix, so a typo
    // early in a longer word ("wodwork" vs "woodworking") still counts.
    return levenshtein(q, word.slice(0, q.length + tolerance)) <= tolerance;
  });
}

/** Lower is a better match; undefined means no match at all. */
function fuzzyScore(text: string, normalizedQuery: string): number | undefined {
  const t = normalize(text);
  const idx = t.indexOf(normalizedQuery);
  if (idx === 0) return 0;
  if (idx > 0) return 1;
  return fuzzyIncludes(t, normalizedQuery) ? 2 : undefined;
}

/**
 * Ranks `candidates` by fuzzy closeness to `query` (exact prefix first,
 * then substring, then typo-tolerant matches), for an instant local
 * suggestion list that doesn't need a network round trip.
 */
export function fuzzySuggest(
  query: string,
  candidates: readonly string[],
  limit = 6,
): string[] {
  const q = normalize(query);
  if (!q) return [];

  const seen = new Set<string>();
  const scored: { candidate: string; score: number }[] = [];
  for (const candidate of candidates) {
    const key = normalize(candidate);
    if (seen.has(key)) continue;
    const score = fuzzyScore(candidate, q);
    if (score === undefined) continue;
    seen.add(key);
    scored.push({ candidate, score });
  }

  scored.sort((a, b) => a.score - b.score || a.candidate.length - b.candidate.length);
  return scored.slice(0, limit).map((s) => s.candidate);
}
