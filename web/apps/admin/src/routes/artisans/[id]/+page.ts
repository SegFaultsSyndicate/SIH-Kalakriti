import { listBadgeCatalog, listArtisanBadges } from '@kalakriti/api';
import { BADGE_CATALOG, GRANTED_BADGES, mergeWithStubs } from '$lib/stubs';
import type { PageLoad } from './$types';

export const ssr = false;

export const load: PageLoad = async ({ params }) => {
  const artisanId = params.id;
  const [catalogRes, grantedRes] = await Promise.all([
    listBadgeCatalog().catch(() => ({ badges: [] })),
    listArtisanBadges(artisanId).catch(() => ({ artisan_badges: [] })),
  ]);
  const badges = catalogRes.badges ?? [];
  const granted = grantedRes.artisan_badges ?? [];
  if (!badges.length && import.meta.env.VITE_USE_MOCKS === '1') {
    console.warn('[mock fallback] badge catalog');
    return { artisanId, badgeCatalog: BADGE_CATALOG, artisanBadges: GRANTED_BADGES };
  }
  return {
    artisanId,
    badgeCatalog: mergeWithStubs(badges, BADGE_CATALOG, (b) => b.id),
    artisanBadges: mergeWithStubs(granted, GRANTED_BADGES, (g) => g.badge.id),
  };
};
