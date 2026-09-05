<!--
  apps/artisan/src/lib/StateBadge.svelte

    <StateBadge group="needsAttention" />

  No Badge/Chip/Pill primitive exists in @kalakriti/ui yet (checked before
  adding one here) -- this is the one place batch 9 needs a state pill, so it
  stays local rather than becoming a new shared component for a single caller.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import type { IconName } from '@kalakriti/icons';
  import { STATE_GROUP_LABEL, type StateGroup } from './listings';

  interface Props {
    group: StateGroup;
  }

  let { group }: Props = $props();

  const t = $derived(locale.t);

  const ICON: Record<StateGroup, IconName> = {
    needsAttention: 'warning',
    draft: 'edit',
    pending: 'clock',
    published: 'success',
    suspended: 'lock',
  };
</script>

<span class="k-state-badge k-state-badge--{group}">
  <Icon name={ICON[group]} class="k-state-badge__icon" />
  {t(STATE_GROUP_LABEL[group])}
</span>

<style>
  /*
   * Only `warning` has a real background+text pair calibrated for a filled
   * pill (palette.css). danger/success are foreground-only colours,
   * contrast-verified against surface-base -- used as a border+text
   * treatment instead, same as the capture step's .photo-issue in batch 8.
   */
  .k-state-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    padding-block: var(--k-space-1);
    padding-inline: var(--k-space-2);
    border: var(--k-hairline) solid transparent;
    border-radius: var(--k-radius-pill);
    font-size: var(--k-text-sm);
    font-weight: 600;
    white-space: nowrap;
  }

  .k-state-badge__icon {
    inline-size: 1em;
    block-size: 1em;
  }

  .k-state-badge--needsAttention {
    border-color: var(--k-accent-danger);
    color: var(--k-accent-danger);
  }

  .k-state-badge--draft,
  .k-state-badge--suspended {
    border-color: var(--k-border-hairline);
    color: var(--k-text-secondary);
  }

  .k-state-badge--pending {
    background-color: var(--k-accent-warning-bg);
    color: var(--k-accent-warning-text);
  }

  .k-state-badge--published {
    border-color: var(--k-accent-success);
    color: var(--k-accent-success);
  }
</style>
