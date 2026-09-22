/**
 * apps/artisan/src/lib/self-badges.ts
 *
 * Badges an artisan applies to themselves (Women-led, Natural dyes, ...).
 * Unlike the catalog badges on /badges, nothing verifies these -- they are
 * the artisan's own claims (the /badges page says so where they are picked).
 *
 * ponytail: stored on this device only (prefs table), since the BFF has no
 * field for them. Move to the artisan profile API once one exists, so buyers
 * can see them on the storefront.
 */
import type { MessageKey } from '@kalakriti/i18n';
import type { IconName } from '@kalakriti/icons';
import { getPref, setPref } from '@kalakriti/offline';

export interface SelfBadge {
  id: string;
  labelKey: MessageKey;
  icon: IconName;
}

export const SELF_BADGES: SelfBadge[] = [
  { id: 'women-led', labelKey: 'selfBadge.womenLed', icon: 'users' },
  { id: 'natural-dyes', labelKey: 'selfBadge.naturalDyes', icon: 'painting' },
  { id: 'eco-materials', labelKey: 'selfBadge.ecoMaterials', icon: 'handmade-certified' },
  { id: 'custom-orders', labelKey: 'selfBadge.customOrders', icon: 'made-to-order' },
  { id: 'bulk-orders', labelKey: 'selfBadge.bulkOrders', icon: 'collective-order' },
  { id: 'family-tradition', labelKey: 'selfBadge.familyTradition', icon: 'charkha-spinner' },
  { id: 'trains-apprentices', labelKey: 'selfBadge.trainsApprentices', icon: 'shg' },
  { id: 'ships-india', labelKey: 'selfBadge.shipsIndia', icon: 'package' },
];

const PREF_KEY = 'artisan.selfBadges';

export async function loadSelfBadges(): Promise<string[]> {
  const ids = (await getPref<string[]>(PREF_KEY)) ?? [];
  // Drop ids from a badge that was since removed from SELF_BADGES.
  return ids.filter((id) => SELF_BADGES.some((b) => b.id === id));
}

export async function saveSelfBadges(ids: string[]): Promise<void> {
  // Plain copy: callers pass Svelte $state proxies, which IndexedDB can't
  // structured-clone (DataCloneError).
  await setPref(PREF_KEY, [...ids]);
}
