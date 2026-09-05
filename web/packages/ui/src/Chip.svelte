<!--
  packages/ui/src/Chip.svelte

    <Chip label={t('search.filters.giOnly')} onremove={() => clearFilter('gi_only')} />
    <Chip label={t('search.crossLingual.badge')} />   -- no onremove: informational, not a filter

  A removable filter chip -- the parsed-query-understanding affordance
  batch 11's /search demands: what the system understood, shown as
  something the buyer can see and take back. Without onremove it is a
  plain informational tag (e.g. the cross-lingual-match badge).
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';

  interface Props {
    label: string;
    onremove?: () => void;
    class?: string;
  }

  let { label, onremove, class: className }: Props = $props();

  const t = $derived(locale.t);
</script>

<span class="k-chip {className || ''}" class:k-chip--removable={!!onremove}>
  {label}
  {#if onremove}
    <button type="button" class="k-chip__remove" onclick={onremove} aria-label={t('search.filters.remove', { label })}>
      <Icon name="close" />
    </button>
  {/if}
</span>

<style>
  .k-chip {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    padding: var(--k-space-1) var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-full, 999px);
    font-size: var(--k-text-sm);
    line-height: var(--k-leading-tight);
    background-color: var(--k-surface-raised);
  }

  .k-chip--removable {
    padding-inline-end: var(--k-space-2);
  }

  .k-chip__remove {
    display: flex;
    border: none;
    background: none;
    padding: 0;
    cursor: pointer;
    color: inherit;
  }
</style>
