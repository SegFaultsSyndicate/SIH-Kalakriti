<!--
  Kalakriti <SyncState /> — Batch 7

    <SyncState state="syncing" />

  Small, header-scale, non-distracting. `offline` reuses Batch 4's
  icons/offline.svg rather than a new file — import path below.
-->
<script>
  import Offline from '../icons/src/offline.svg';
  import Pending from './src/sync-pending.svg';
  import Syncing from './src/sync-syncing.svg';
  import Synced from './src/sync-synced.svg';

  const ICONS = { offline: Offline, pending: Pending, syncing: Syncing, synced: Synced };

  /** @type {'offline'|'pending'|'syncing'|'synced'} */
  export let state = 'synced';
  export let title = undefined;
</script>

<svelte:component
  this={ICONS[state]}
  class="k-sync-state k-sync-state--{state}"
  role={title ? 'img' : undefined}
  aria-label={title}
  aria-hidden={title ? undefined : 'true'}
  focusable="false"
  {...$$restProps}
/>
