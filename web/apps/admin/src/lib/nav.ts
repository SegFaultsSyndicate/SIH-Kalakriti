// apps/admin/src/lib/nav.ts
//
// The single source of truth for the sidebar and the command palette --
// the palette's target list IS this route list, not a separate index.

import type { MessageKey } from '@kalakriti/i18n';
import type { IconName } from '@kalakriti/icons';

export interface NavItem {
  href: string;
  labelKey: MessageKey;
  icon: IconName;
  /** Claim role required to see this item; omitted means any authenticated role. */
  role?: string;
}

export const NAV_ITEMS: NavItem[] = [
  { href: '/insights', labelKey: 'nav.insights', icon: 'income-statement', role: 'MINISTRY' },
  { href: '/clusters', labelKey: 'nav.clusters', icon: 'cluster' },
  { href: '/moderation', labelKey: 'nav.moderation', icon: 'warning' },
  { href: '/crafts', labelKey: 'nav.crafts', icon: 'weaving' },
];
