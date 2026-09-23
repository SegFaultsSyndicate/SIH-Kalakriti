<!--
  apps/artisan/src/routes/agent/+page.svelte

  F14 assisted mode: a field agent's home. The artisans who consented to
  this agent's help, and a way to start helping one -- which sets the
  X-On-Behalf-Of target and drops the agent into the ordinary artisan
  screens under a persistent "Helping {name}" banner (see +layout.svelte).
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale, formatRelativeTime } from '@kalakriti/i18n';
  import { Button, EmptyState } from '@kalakriti/ui';
  import {
    getMyStaffAccount,
    listMyArtisans,
    session,
    setAccessToken,
    setRefreshToken,
    type AgentArtisan,
    type StaffAccount,
  } from '@kalakriti/api';
  import { acting } from '$lib/acting.svelte';

  const t = $derived(locale.t);

  let me = $state<StaffAccount | null>(null);
  let artisans = $state<AgentArtisan[] | null>(null);
  let failed = $state(false);

  async function load(): Promise<void> {
    failed = false;
    try {
      const [m, a] = await Promise.all([getMyStaffAccount(), listMyArtisans()]);
      me = m.staff ?? null;
      artisans = a.artisans ?? [];
    } catch {
      failed = true;
    }
  }

  $effect(() => {
    void load();
  });

  function help(a: AgentArtisan): void {
    if (!a.artisan_id) return;
    acting.start({ id: a.artisan_id, name: a.display_name ?? '' });
    void goto('/');
  }

  async function signOut(): Promise<void> {
    acting.stop();
    setAccessToken(undefined);
    setRefreshToken(undefined);
    session.clear();
    await goto('/welcome');
  }
</script>

<svelte:head>
  <title>{t('agent.title')} — {t('app.name')}</title>
</svelte:head>

<div class="agent">
  <h1>{t('agent.title')}</h1>
  {#if me}
    <p class="agent__muted">{me.display_name}{me.district ? ` · ${me.district}` : ''}</p>
  {/if}

  <Button size="xl" onclick={() => goto('/agent/add')}>{t('agent.add')}</Button>

  {#if failed}
    <p class="agent__muted">{t('state.error.body')}</p>
    <Button variant="ghost" onclick={load}>{t('state.error.retry')}</Button>
  {:else if artisans !== null && artisans.length === 0}
    <EmptyState illustration="empty-listings" heading={t('agent.empty')} />
  {:else if artisans}
    <ul class="agent__list" role="list">
      {#each artisans as a (a.artisan_id)}
        <li class="agent__row">
          <div>
            <p class="agent__name">{a.display_name}</p>
            <p class="agent__muted">
              {[a.village, a.district].filter(Boolean).join(', ')}
              {#if a.last_activity_at}· {formatRelativeTime(new Date(a.last_activity_at), locale.code)}{/if}
            </p>
            {#if a.needs_review}
              <p class="agent__flag">{t('agent.needsReview')}</p>
            {/if}
          </div>
          <Button size="md" onclick={() => help(a)}>{t('agent.help')}</Button>
        </li>
      {/each}
    </ul>
  {/if}

  <Button variant="ghost" onclick={signOut}>{t('profile.session.signOut')}</Button>
</div>

<style>
  .agent {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    padding: var(--k-space-4);
  }

  .agent h1 {
    margin: 0;
    font-size: var(--k-text-xl);
  }

  .agent__muted {
    margin: 0;
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .agent__list {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .agent__row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-3);
    padding-block: var(--k-space-3);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  .agent__name {
    margin: 0;
    font-weight: 700;
  }

  .agent__flag {
    margin: 0;
    font-size: var(--k-text-xs);
    font-weight: 600;
    color: var(--k-accent-danger);
  }
</style>
