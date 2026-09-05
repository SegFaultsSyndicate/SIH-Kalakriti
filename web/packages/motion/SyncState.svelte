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

  /**
   * @typedef {object} Props
   * @property {'offline'|'pending'|'syncing'|'synced'} [state]
   * @property {string} [title] The state named in words. Sync state is never
   *   carried by the glyph alone.
   */

  /** @type {Props & Record<string, unknown>} */
  let { state = 'synced', title, ...rest } = $props();

  const Component = $derived(ICONS[state]);
</script>

<Component
  class="k-sync-state k-sync-state--{state}"
  role={title ? 'img' : undefined}
  aria-label={title}
  aria-hidden={title ? undefined : 'true'}
  focusable="false"
  {...rest}
/>
