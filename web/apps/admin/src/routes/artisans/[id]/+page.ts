import { listBadgeCatalog, listArtisanBadges } from '@kalakriti/api';
import type { PageLoad } from './$types';

export const ssr = false;

export const load: PageLoad = async ({ params }) => {
  const artisanId = params.id;
  const [catalogRes, grantedRes] = await Promise.all([
    listBadgeCatalog().catch(() => ({ badges: [] })),
    listArtisanBadges(artisanId).catch(() => ({ artisan_badges: [] })),
  ]);
  return {
    artisanId,
    badgeCatalog: catalogRes.badges ?? [],
    artisanBadges: grantedRes.artisan_badges ?? [],
  };
};
