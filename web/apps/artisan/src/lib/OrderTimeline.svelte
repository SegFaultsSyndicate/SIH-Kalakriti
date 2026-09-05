<!--
  apps/artisan/src/lib/OrderTimeline.svelte

    <OrderTimeline orderId={order.id} />

  GET /orders/{id}/events via watchOrderEvents (@kalakriti/api) -- real SSE,
  reconnects with backoff and resumes from Last-Event-ID on its own (see
  packages/api/src/sse.svelte.ts). This component's own job is turning each
  raw event into a narrated line (order-timeline.ts) and never showing a
  bare state name, per the batch 10 brief.

  Reconnect/backfill correctness: watchOrderEvents replays history from
  Last-Event-ID on every reconnect, and appendNarrated skips an event_id
  already in `lines` -- so a network drop mid-stream shows a brief
  "reconnecting" note, then continues with no gap and no duplicate line.
-->
<script lang="ts">
  import { onDestroy } from 'svelte';
  import { watchOrderEvents } from '@kalakriti/api';
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { appendNarrated, acceptedCount, type OrderTimelineEvent, type TimelineLine } from './order-timeline';

  interface Props {
    orderId: string;
  }

  let { orderId }: Props = $props();

  const t = $derived(locale.t);

  let lines = $state<TimelineLine[]>([]);
  let rawEvents = $state<OrderTimelineEvent[]>([]);

  const watcher = watchOrderEvents(orderId);
  const status = $derived(watcher.state.status);

  $effect(() => {
    const ev = watcher.state.lastEvent;
    if (!ev) return;
    let parsed: OrderTimelineEvent | undefined;
    try {
      parsed = JSON.parse(ev.data) as OrderTimelineEvent;
    } catch {
      return;
    }
    if (!parsed || rawEvents.some((e) => e.event_id === parsed!.event_id)) return;
    rawEvents = [...rawEvents, parsed];
    lines = appendNarrated(lines, parsed);
  });

  onDestroy(() => watcher.stop());

  const accepted = $derived(acceptedCount(rawEvents));
</script>

<div class="k-order-timeline">
  {#if status === 'connecting'}
    <p class="k-order-timeline__status" role="status">
      <Icon name="sync" class="k-order-timeline__status-icon" />
      {t('timeline.reconnecting')}
    </p>
  {/if}

  {#if accepted > 0}
    <p class="k-order-timeline__accepted">{t('timeline.acceptedCount', { count: String(accepted) })}</p>
  {/if}

  {#if lines.length === 0 && status !== 'connecting'}
    <p class="k-order-timeline__empty">{t('timeline.empty')}</p>
  {:else}
    <ol class="k-order-timeline__list">
      {#each lines as line (line.id)}
        <li class="k-order-timeline__item">
          <span class="k-order-timeline__dot" aria-hidden="true"></span>
          <p>{t(line.key, line.params)}</p>
        </li>
      {/each}
    </ol>
  {/if}
</div>

<style>
  .k-order-timeline {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .k-order-timeline__status {
    display: flex;
    align-items: center;
    gap: var(--k-space-1);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  :global(.k-order-timeline__status-icon) {
    inline-size: 1em;
    block-size: 1em;
  }

  .k-order-timeline__accepted {
    font-weight: 600;
  }

  .k-order-timeline__empty {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .k-order-timeline__list {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    border-inline-start: var(--k-hairline) solid var(--k-border-hairline);
    padding-inline-start: var(--k-space-3);
  }

  .k-order-timeline__item {
    position: relative;
    display: flex;
    align-items: start;
    gap: var(--k-space-2);
  }

  .k-order-timeline__dot {
    flex-shrink: 0;
    margin-block-start: 0.4em;
    inline-size: 0.5rem;
    block-size: 0.5rem;
    border-radius: 50%;
    background-color: var(--k-accent-primary-bg);
    margin-inline-start: calc(-1 * var(--k-space-3) - 0.25rem);
  }
</style>
