<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import BadgeChip from './BadgeChip.svelte';
  import EmptyState from './EmptyState.svelte';

  interface BadgeCatalogEntry {
    id: string;
    code: string;
    kind: 'EARNED' | 'CONFERRED';
    tier?: 'BRONZE' | 'SILVER' | 'GOLD';
    icon_name: string;
    metric?: string;
    threshold?: number;
  }
  interface GrantedBadge { badge: BadgeCatalogEntry; granted_at: string; granted_by: string }
  interface ProgressEntry { metric: string; value: number }

  interface Props {
    catalog: BadgeCatalogEntry[];
    granted: GrantedBadge[];
    progress: ProgressEntry[];
  }

  let { catalog, granted, progress }: Props = $props();

  const t = $derived(locale.t);
  const grantedIds = $derived(new Set(granted.map((g) => g.badge.id)));
  const grantedByAt = $derived(new Map(granted.map((g) => [g.badge.id, g.granted_at])));
  const progressByMetric = $derived(new Map(progress.map((p) => [p.metric, p.value])));

  const grantedEntries = $derived(catalog.filter((b) => grantedIds.has(b.id)));

  // For each EARNED metric family, find the lowest-threshold tier not yet
  // granted -- that is the "in progress" row; higher tiers in the same
  // family are not shown separately until that one is reached.
  const inProgressEntries = $derived(
    catalog
      .filter((b) => b.kind === 'EARNED' && !grantedIds.has(b.id))
      .filter((b, _, all) => {
        const sameFamily = all.filter((x) => x.metric === b.metric && !grantedIds.has(x.id));
        const lowest = sameFamily.reduce((min, x) => ((x.threshold ?? 0) < (min.threshold ?? 0) ? x : min), sameFamily[0]);
        return b.id === lowest?.id;
      }),
  );

  const lockedConferredEntries = $derived(
    catalog.filter((b) => b.kind === 'CONFERRED' && !grantedIds.has(b.id)),
  );
</script>

<div class="k-badge-grid">
  {#if grantedEntries.length === 0 && inProgressEntries.length === 0}
    <EmptyState illustration="empty-no-listings" heading={t('badges.empty')} />
  {:else}
    {#each grantedEntries as badge (badge.id)}
      <BadgeChip {badge} granted={true} grantedAt={grantedByAt.get(badge.id)} />
    {/each}
    {#each inProgressEntries as badge (badge.id)}
      <BadgeChip
        {badge}
        granted={false}
        progress={{ current: progressByMetric.get(badge.metric ?? '') ?? 0, total: badge.threshold ?? 1 }}
      />
    {/each}
  {/if}
  {#each lockedConferredEntries as badge (badge.id)}
    <BadgeChip {badge} granted={false} />
  {/each}
</div>

<style>
  .k-badge-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: var(--k-space-3);
  }
</style>
