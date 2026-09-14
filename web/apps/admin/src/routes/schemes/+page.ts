// apps/admin/src/routes/schemes/+page.ts
import { listSchemes, type GovernmentScheme } from '@kalakriti/api';

export const ssr = false;

export async function load(): Promise<{ schemes: GovernmentScheme[] }> {
  try {
    const res = await listSchemes();
    return { schemes: res.schemes ?? [] };
  } catch (err) {
    console.error('Failed to load schemes', err);
    return { schemes: [] };
  }
}
