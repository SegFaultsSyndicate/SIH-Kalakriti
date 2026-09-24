// apps/artisan/src/routes/schemes/+page.ts
import { matchSchemes, type components } from '@kalakriti/api';
import { MOCK_SCHEME_MATCHES, type CasteCategorizedSchemeMatch } from '$lib/mock-data';

export const ssr = false;

export type SchemeMatch = components['schemas']['SchemeMatch'] | CasteCategorizedSchemeMatch;

export async function load(): Promise<{ matches: SchemeMatch[] }> {
  try {
    const res = await matchSchemes();
    if (res.matches && res.matches.length > 0) {
      return { matches: res.matches };
    }
    return { matches: MOCK_SCHEME_MATCHES };
  } catch (err) {
    console.warn('[mock fallback] load scheme matches:', err);
    return { matches: MOCK_SCHEME_MATCHES };
  }
}
