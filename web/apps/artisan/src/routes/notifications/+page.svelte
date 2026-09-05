<!--
  apps/artisan/src/routes/notifications/+page.svelte

  The fanned-out feed: GET /feed (channel-svc's own doc comment: "a
  follower's feed is exactly the notifications addressed to them" -- the
  same read model backs both). Grouped by kind, marked read on open via
  POST /feed/{id}/read, deep-linked from payload.

  Deduplication is entirely server-side (services/channel-svc/internal/channel/consumer/fanout.go:
  one notification per recipient per artisan per hour) -- nothing here
  re-dedupes what the feed already returns, since the backend's own batching
  rule is what the brief means by "deduplicated per the backend's batching
  rules".
-->
<script lang="ts">
  import { locale, formatDate } from '@kalakriti/i18n';
  import { EmptyState } from '@kalakriti/ui';
  import { Card, Skeleton } from '@kalakriti/patterns';
  import { Icon } from '@kalakriti/icons';
  import { getFeed, markFeedItemRead, type components } from '@kalakriti/api';
  import type { MessageKey } from '@kalakriti/i18n';

  type FeedItem = components['schemas']['FeedItem'];

  const t = $derived(locale.t);

  let loading = $state(true);
  let items = $state<FeedItem[]>([]);

  async function load(): Promise<void> {
    loading = true;
    try {
      const res = await getFeed();
      items = res.feed ?? [];
    } catch {
      items = [];
    }
    loading = false;
  }

  $effect(() => {
    void load();
  });

  const GROUPS = ['orders', 'listings', 'social'] as const;
  type Group = (typeof GROUPS)[number];

  function groupFor(kind: string | undefined): Group {
    if (kind === 'ARTISAN_FOLLOWED') return 'social';
    if (kind === 'LISTING_APPROVAL_DUE') return 'listings';
    return 'orders';
  }

  const grouped = $derived.by(() => {
    const byGroup = new Map<Group, FeedItem[]>(GROUPS.map((g) => [g, []]));
    for (const item of items) byGroup.get(groupFor(item.kind))?.push(item);
    return byGroup;
  });

  function deepLink(item: FeedItem): string | undefined {
    const payload = item.payload ?? {};
    if (payload.lot_id && payload.order_id) return `/orders/${payload.order_id}/lots/${payload.lot_id}`;
    if (payload.order_id) return `/orders/${payload.order_id}`;
    if (payload.listing_id) return `/listings/${payload.listing_id}`;
    return undefined;
  }

  async function onOpen(item: FeedItem): Promise<void> {
    if (item.read_at || !item.id) return;
    item.read_at = new Date().toISOString();
    items = [...items];
    try {
      await markFeedItemRead(item.id);
    } catch {
      /* best-effort -- staying marked read locally is fine even if the write lags. */
    }
  }

  const GROUP_LABEL: Record<Group, MessageKey> = {
    orders: 'notifications.group.orders',
    listings: 'notifications.group.listings',
    social: 'notifications.group.social',
  };
</script>

<svelte:head>
  <title>{t('notifications.heading')} — {t('app.name')}</title>
</svelte:head>

<main class="notifications-page">
  <h1>{t('notifications.heading')}</h1>

  {#if loading && items.length === 0}
    <Skeleton shape="card" height="4rem" />
  {:else if items.length === 0}
    <EmptyState illustration="empty-no-notifications" heading={t('notifications.empty')} />
  {:else}
    {#each GROUPS as group (group)}
      {@const rows = grouped.get(group) ?? []}
      {#if rows.length > 0}
        <section class="notifications-page__group">
          <h2>{t(GROUP_LABEL[group])}</h2>
          <ul role="list">
            {#each rows as item (item.id)}
              {@const href = deepLink(item)}
              <li>
                {#snippet row()}
                  <Card variant="hairline" element="div" class="notifications-page__item">
                    {#if !item.read_at}
                      <span class="notifications-page__dot" aria-hidden="true"></span>
                    {/if}
                    <div class="notifications-page__body">
                      <p class="notifications-page__title">{item.title}</p>
                      <p class="notifications-page__text">{item.body}</p>
                      {#if item.created_at}
                        <p class="notifications-page__time">
                          {formatDate(item.created_at, locale.code, { dateStyle: 'medium', timeStyle: 'short' })}
                        </p>
                      {/if}
                    </div>
                    {#if href}
                      <Icon name="chevron-right" />
                    {/if}
                  </Card>
                {/snippet}
                {#if href}
                  <a href={href} class="notifications-page__link" onclick={() => onOpen(item)}>
                    {@render row()}
                  </a>
                {:else}
                  <button type="button" class="notifications-page__link notifications-page__link--button" onclick={() => onOpen(item)}>
                    {@render row()}
                  </button>
                {/if}
              </li>
            {/each}
          </ul>
        </section>
      {/if}
    {/each}
  {/if}
</main>

<style>
  .notifications-page {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding: var(--k-space-4);
    padding-block-end: calc(var(--k-space-4) + env(safe-area-inset-bottom));
  }

  .notifications-page__group ul {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .notifications-page__group h2 {
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
    margin-block-end: var(--k-space-2);
  }

  .notifications-page__link {
    display: block;
    color: inherit;
    text-decoration: none;
    background: none;
    border: none;
    inline-size: 100%;
    text-align: start;
    padding: 0;
    cursor: pointer;
  }

  :global(.notifications-page__item) {
    display: flex;
    align-items: start;
    gap: var(--k-space-2);
    padding: var(--k-space-3);
  }

  .notifications-page__dot {
    flex-shrink: 0;
    margin-block-start: 0.4em;
    inline-size: 0.5rem;
    block-size: 0.5rem;
    border-radius: 50%;
    background-color: var(--k-accent-primary-bg);
  }

  .notifications-page__body {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .notifications-page__title {
    font-weight: 600;
  }

  .notifications-page__text {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .notifications-page__time {
    color: var(--k-text-secondary);
    font-size: var(--k-text-xs);
  }
</style>
