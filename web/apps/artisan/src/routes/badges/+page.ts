import { listBadgeCatalog, listArtisanBadges, getBadgeProgress, session } from '@kalakriti/api';

export const ssr = false;

export async function load() {
  const artisanId = (session.claims?.['sub'] as string) || '';
  const [catalogRes, grantedRes, progressRes] = await Promise.all([
    listBadgeCatalog().catch(() => ({ badges: [] })),
    artisanId ? listArtisanBadges(artisanId).catch(() => ({ artisan_badges: [] })) : Promise.resolve({ artisan_badges: [] }),
    getBadgeProgress().catch(() => ({ progress: [] })),
  ]);
  return {
    catalog: catalogRes.badges ?? [],
    granted: grantedRes.artisan_badges ?? [],
    progress: progressRes.progress ?? [],
  };
}
