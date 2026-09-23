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
  /** Any of these roles may see it (for items shared by officers and the ministry). */
  roles?: string[];
  /**
   * Optional CSS length overriding the default 1.25rem icon box. Used for the
   * hand-authored badge emblem, whose narrow art fills far less of a square
   * box than the stroke icons — the box is enlarged instead of stretching or
   * clipping the artwork.
   */
  iconSize?: string;
  /**
   * Suppress the role pill on the dashboard quick-link tile only (the role is
   * still required for access and still shown in the sidebar). Used where the
   * role tag is redundant on the card.
   */
  hideRoleTag?: boolean;
}

export const NAV_ITEMS: NavItem[] = [
  { href: '/impact', labelKey: 'nav.impact', icon: 'income-statement', roles: ['MINISTRY', 'CLUSTER_OFFICER'] },
  { href: '/finance', labelKey: 'nav.finance', icon: 'dollar-sign', roles: ['MINISTRY', 'CLUSTER_OFFICER'] },
  { href: '/staff', labelKey: 'nav.staff', icon: 'users', roles: ['MINISTRY', 'CLUSTER_OFFICER'] },
  { href: '/insights', labelKey: 'nav.insights', icon: 'income-statement', role: 'MINISTRY', hideRoleTag: true },
  { href: '/clusters', labelKey: 'nav.clusters', icon: 'cluster' },
  { href: '/moderation', labelKey: 'nav.moderation', icon: 'warning' },
  { href: '/crafts', labelKey: 'nav.crafts', icon: 'weaving' },
  { href: '/companies', labelKey: 'nav.companies', icon: 'package' },
  { href: '/artisans', labelKey: 'nav.badges', icon: 'badge', role: 'MINISTRY', iconSize: '1.75rem' },
  { href: '/schemes', labelKey: 'nav.schemes', icon: 'verified-artisan', role: 'MINISTRY', hideRoleTag: true },
];

/** Whether `role` may see `item`. */
export function visibleTo(item: NavItem, role: string | undefined): boolean {
  if (item.roles) return role !== undefined && item.roles.includes(role);
  return !item.role || item.role === role;
}
