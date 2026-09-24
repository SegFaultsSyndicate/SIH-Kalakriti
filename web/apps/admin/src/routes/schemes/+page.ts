// apps/admin/src/routes/schemes/+page.ts
import { listSchemes, type GovernmentScheme } from '@kalakriti/api';
import { GOVERNMENT_SCHEMES, mergeWithStubs } from '$lib/stubs';

export const ssr = false;

export async function load(): Promise<{ schemes: GovernmentScheme[] }> {
  try {
    const res = await listSchemes();
    return { schemes: mergeWithStubs(res.schemes ?? [], GOVERNMENT_SCHEMES, (s) => s.id) };
  } catch (err) {
    console.error('Failed to load schemes', err);
    if (import.meta.env.VITE_USE_MOCKS === '1') {
      console.warn('[mock fallback] listSchemes:', err);
      return { schemes: GOVERNMENT_SCHEMES };
    }
    return { schemes: [] };
  }
}