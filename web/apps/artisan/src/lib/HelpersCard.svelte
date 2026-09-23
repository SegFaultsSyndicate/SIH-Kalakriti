<!--
  apps/artisan/src/lib/HelpersCard.svelte

  F14 "People helping me": every field agent the artisan has let act for
  them, and a way to take that back. Revoking is the artisan's own act --
  the server refuses it from an agent's on-behalf session -- so the card is
  not rendered at all while an agent is acting.
-->
<script lang="ts">
  import { locale, formatDate } from '@kalakriti/i18n';
  import { Button, Dialog, showToast } from '@kalakriti/ui';
  import { listHelpers, revokeHelper, type Helper } from '@kalakriti/api';

  const t = $derived(locale.t);

  let helpers = $state<Helper[] | null>(null);
  let confirming = $state<Helper | null>(null);
  let confirmOpen = $state(false);
  let revoking = $state(false);

  $effect(() => {
    listHelpers()
      .then((r) => (helpers = r.helpers ?? []))
      .catch(() => (helpers = []));
  });

  function ask(h: Helper): void {
    confirming = h;
    confirmOpen = true;
  }

  async function revoke(): Promise<void> {
    if (!confirming?.link_id) return;
    revoking = true;
    try {
      await revokeHelper(confirming.link_id);
      helpers = (helpers ?? []).filter((h) => h.link_id !== confirming?.link_id);
      showToast({ message: t('helpers.revoked'), variant: 'success' });
      confirmOpen = false;
    } catch {
      showToast({ message: t('api.error.unknown'), variant: 'error' });
    } finally {
      revoking = false;
    }
  }
</script>

{#if helpers && helpers.length > 0}
  <section class="helpers" aria-labelledby="helpers-title">
    <h2 id="helpers-title" class="helpers__title">{t('helpers.title')}</h2>
    <p class="helpers__desc">{t('helpers.desc')}</p>
    <ul class="helpers__list" role="list">
      {#each helpers as h (h.link_id)}
        <li class="helpers__row">
          <div>
            <p class="helpers__name">{h.display_name}</p>
            {#if h.linked_at}
              <p class="helpers__since">{t('helpers.since', { date: formatDate(new Date(h.linked_at), locale.code) })}</p>
            {/if}
          </div>
          <Button size="md" variant="secondary" onclick={() => ask(h)}>{t('helpers.revoke')}</Button>
        </li>
      {/each}
    </ul>
  </section>
{/if}

<Dialog bind:open={confirmOpen} title={t('helpers.revoke')}>
  <p>{t('helpers.confirm', { name: confirming?.display_name ?? '' })}</p>
  <div class="helpers__actions">
    <Button variant="ghost" onclick={() => (confirmOpen = false)}>{t('action.cancel')}</Button>
    <Button variant="danger" loading={revoking} onclick={revoke}>{t('helpers.revoke')}</Button>
  </div>
</Dialog>

<style>
  .helpers {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    padding: var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    background: var(--k-surface-raised);
  }

  .helpers__title {
    margin: 0;
    font-size: var(--k-text-md);
  }

  .helpers__desc,
  .helpers__since {
    margin: 0;
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .helpers__list {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .helpers__row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-3);
    padding-block: var(--k-space-2);
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
  }

  .helpers__name {
    margin: 0;
    font-weight: 600;
  }

  .helpers__actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-3);
  }
</style>
