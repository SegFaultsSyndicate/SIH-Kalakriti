// apps/artisan/src/routes/schemes/+page.ts
import { matchSchemes, type components } from '@kalakriti/api';

export const ssr = false;

export type SchemeMatch = components['schemas']['SchemeMatch'];

export async function load(): Promise<{ matches: SchemeMatch[] }> {
  try {
    const res = await matchSchemes();
    return { matches: res.matches ?? [] };
  } catch (err) {
    console.error('Failed to load scheme matches', err);
    return { matches: [] };
  }
}
