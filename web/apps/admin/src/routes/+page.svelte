<!--
  apps/admin/src/routes/+page.svelte
  Dashboard landing: quick links to the four Batch 13 surfaces.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { session } from '@kalakriti/api';
  import { NAV_ITEMS } from '$lib/nav';

  const t = $derived(locale.t);
  const role = $derived(session.claims?.['role'] as string | undefined);
  const items = $derived(NAV_ITEMS.filter((item) => !item.role || item.role === role));
</script>

<svelte:head>
  <title>{t('nav.dashboard')} — {t('admin.home.title')}</title>
</svelte:head>

<h1>{t('nav.dashboard')}</h1>

{#if session.status !== 'authenticated'}
  <p>{t('admin.home.signedOut')} <a href="/login">{t('login.heading')}</a></p>
{:else if items.length === 0}
  <p>{t('admin.home.noAccess')}</p>
{:else}
  <ul class="tiles">
    {#each items as item (item.href)}
      <li>
        <a class="tile" href={item.href}>
          <Icon name={item.icon} />
          <span>{t(item.labelKey)}</span>
        </a>
      </li>
    {/each}
  </ul>
{/if}

<style>
  .tiles {
    list-style: none;
    margin: var(--k-space-5) 0 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(12rem, 1fr));
    gap: var(--k-space-3);
  }

  .tile {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    padding: var(--k-space-5);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-md);
    color: var(--k-text-primary);
    font-weight: var(--k-weight-medium);
  }
</style>
