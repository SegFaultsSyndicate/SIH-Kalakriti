/**
 * apps/artisan/src/lib/business-tips.ts
 *
 * Daily "tip of the day" business/financial advice shown once per day by
 * TipOfTheDayModal.svelte. Unlike PriceAdvisory (a real advisePricing()
 * result for one listing), these are generic, static guidance -- same
 * MessageKey-array shape as self-badges.ts's SELF_BADGES.
 */
import type { MessageKey } from '@kalakriti/i18n';
import type { IconName } from '@kalakriti/icons';

export interface BusinessTip {
  id: string;
  bodyKey: MessageKey;
  icon: IconName;
}

export const BUSINESS_TIPS: readonly BusinessTip[] = [
  { id: 'price-floor', bodyKey: 'tips.priceFloor', icon: 'fair-price' },
  { id: 'track-income', bodyKey: 'tips.trackIncome', icon: 'income-statement' },
  { id: 'bulk-orders', bodyKey: 'tips.bulkOrders', icon: 'collective-order' },
  { id: 'photo-quality', bodyKey: 'tips.photoQuality', icon: 'camera' },
  { id: 'respond-fast', bodyKey: 'tips.respondFast', icon: 'clock' },
  { id: 'save-savings', bodyKey: 'tips.saveSavings', icon: 'lock' },
  { id: 'gi-premium', bodyKey: 'tips.giPremium', icon: 'gi-tagged' },
  { id: 'restock-early', bodyKey: 'tips.restockEarly', icon: 'package' },
  { id: 'cluster-support', bodyKey: 'tips.clusterSupport', icon: 'shg' },
  { id: 'story-sells', bodyKey: 'tips.storySells', icon: 'provenance' },
];
