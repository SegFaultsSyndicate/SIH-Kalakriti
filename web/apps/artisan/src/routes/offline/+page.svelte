<!--
  apps/artisan/src/routes/offline/+page.svelte

  The offline screen.

  It is NOT a dead end that says "you are offline". Being offline is the normal
  case for this user, not an error, so this screen does the two things that are
  actually useful in that state: it accounts for the work already recorded and
  still waiting to be sent, and it gets the artisan straight back to capturing
  more. The connection is the app's problem, not theirs.

  The queue reads from IndexedDB, which is available with no network at all, so
  every number here is real even on airplane mode.
-->
<script lang="ts">
  import { formatRelativeTime, locale, type MessageKey } from '@kalakriti/i18n';
  import { OutboxView, retryFailed, network, type OutboxEntry } from '@kalakriti/offline';

  const t = $derived(locale.t);
  const tp = $derived(locale.tPlural.bind(locale));

  const outbox = new OutboxView();

  $effect(() => outbox.start());
  $effect(() => network.start());

  let retrying = $state(false);

  async function retry() {
    retrying = true;
    try {
      await retryFailed();
    } finally {
      retrying = false;
    }
  }

  function kindLabel(entry: OutboxEntry): string {
    return t(`outbox.kind.${entry.kind}` as MessageKey);
  }

  function statusLabel(entry: OutboxEntry): string {
    return t(`sync.${entry.status}` as MessageKey);
  }
</script>

<svelte:head>
  <title>{t('offline.title')} — {t('app.name')}</title>
</svelte:head>

<h1>{t('offline.heading')}</h1>
<p class="lede">{t('network.offline.explain')}</p>

<!--
  Capture first, status second. The artisan came here because something did not
  load; the fastest way out of that is to let them keep working.
-->
<div class="actions">
  <a class="action action--primary" href="/listing/new/capture">{t('offline.newListing')}</a>
  <a class="action" href="/">{t('offline.continue')}</a>
</div>

<section class="queue" aria-labelledby="queue-heading">
  <h2 id="queue-heading">{t('offline.queue.heading')}</h2>

  {#if !outbox.loaded}
    <!-- Loading state. -->
    <p class="state" role="status" aria-live="polite">{t('state.loading')}</p>
  {:else if outbox.error}
    <!-- Error state: IndexedDB unavailable or evicted. -->
    <div class="state state--error" role="alert">
      <p class="state__title">{t('state.error.title')}</p>
      <p>{t('state.error.body')}</p>
    </div>
  {:else if outbox.entries.length === 0}
    <!-- Empty state. -->
    <p class="state">{t('offline.queue.empty')}</p>
  {:else}
    <p class="count" aria-live="polite">
      {tp('offline.queue.count', outbox.counts.total)}
      {#if outbox.counts.failed > 0}
        <span class="count__failed">
          {t('offline.queue.failed', { count: outbox.counts.failed })}
        </span>
      {/if}
    </p>

    <ul role="list" class="entries">
      {#each outbox.entries as entry (entry.id)}
        <li class="entry" data-status={entry.status}>
          <span class="entry__kind">{kindLabel(entry)}</span>
          <!-- Status is words, never colour alone. -->
          <span class="entry__status">{statusLabel(entry)}</span>
          <time class="entry__time" datetime={new Date(entry.createdAt).toISOString()}>
            {t('offline.savedAt', {
              time: formatRelativeTime(entry.createdAt, locale.code),
            })}
          </time>
          {#if entry.lastError}
            <p class="entry__error">{entry.lastError}</p>
          {/if}
        </li>
      {/each}
    </ul>

    {#if outbox.counts.failed > 0}
      <button
        type="button"
        class="action"
        onclick={retry}
        disabled={retrying || !network.online}
      >
        {t('offline.queue.retryAll')}
      </button>
    {/if}
  {/if}
</section>

<style>
  .lede {
    margin-block-start: var(--k-space-3);
    color: var(--k-text-secondary);
  }

  .actions {
    display: grid;
    gap: var(--k-space-3);
    margin-block-start: var(--k-space-6);
  }

  .action {
    display: flex;
    align-items: center;
    justify-content: center;
    min-block-size: var(--k-touch-min);
    padding: var(--k-space-3) var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    background-color: transparent;
    color: var(--k-text-primary);
    text-decoration: none;
    font-size: var(--k-text-md);
    cursor: pointer;
  }

  .action--primary {
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    border-color: var(--k-accent-primary-bg);
    font-weight: var(--k-weight-medium);
  }

  .action:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .queue {
    margin-block-start: var(--k-space-7);
    padding-block-start: var(--k-space-4);
    border-block-start: var(--k-rule) solid var(--k-border-hairline);
  }

  .state {
    margin-block-start: var(--k-space-4);
    color: var(--k-text-secondary);
  }

  .state--error {
    padding: var(--k-space-4);
    border-inline-start: var(--k-rule-heavy) solid var(--k-accent-danger);
    background-color: var(--k-surface-sunken);
  }

  .state__title {
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
  }

  .count {
    margin-block-start: var(--k-space-3);
    color: var(--k-text-secondary);
  }

  .count__failed {
    display: block;
    color: var(--k-accent-danger);
    font-weight: var(--k-weight-medium);
  }

  .entries {
    margin-block: var(--k-space-4);
  }

  /* Rows separated by a hairline, not by a stack of floating cards. */
  .entry {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: var(--k-space-1) var(--k-space-3);
    padding-block: var(--k-space-3);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }

  .entry:last-child {
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  .entry__kind {
    font-weight: var(--k-weight-medium);
  }

  .entry__status {
    text-align: end;
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .entry[data-status='failed'] .entry__status,
  .entry[data-status='needsAttention'] .entry__status,
  .entry[data-status='blocked'] .entry__status {
    color: var(--k-accent-danger);
  }

  .entry__time {
    grid-column: 1 / -1;
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .entry__error {
    grid-column: 1 / -1;
    font-size: var(--k-text-sm);
    color: var(--k-accent-danger);
  }
</style>
