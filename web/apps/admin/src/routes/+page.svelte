<!--
  apps/admin/src/routes/+page.svelte
  Dashboard landing: quick links to the console's surfaces.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { session } from '@kalakriti/api';
  import { NAV_ITEMS, visibleTo } from '$lib/nav';

  const t = $derived(locale.t);
  const role = $derived(session.claims?.['role'] as string | undefined);
  const items = $derived(NAV_ITEMS.filter((item) => visibleTo(item, role)));
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
  <div class="tile-grid">
    {#each items as item (item.href)}
      <a class="tile" href={item.href}>
        <div class="tile__header">
          <div class="tile__icon-badge">
            <Icon name={item.icon} size="1.4rem" />
          </div>
          {#if item.role && !item.hideRoleTag}
            <span class="tile__role-pill">{item.role}</span>
          {/if}
        </div>

        <div class="tile__body">
          <h3 class="tile__name">{t(item.labelKey)}</h3>
        </div>

        <div class="tile__footer">
          <span class="tile__path">{item.href}</span>
          <span class="tile__cta">{t('craftGrid.explore')}</span>
        </div>
      </a>
    {/each}
  </div>
{/if}

<style>
  .tile-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 1rem;
    margin-block-start: 1.5rem;
  }

  @media (max-width: 1100px) {
    .tile-grid {
      grid-template-columns: repeat(3, 1fr);
    }
  }

  @media (max-width: 768px) {
    .tile-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (max-width: 480px) {
    .tile-grid {
      grid-template-columns: 1fr;
    }
  }

  .tile {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    background-color: var(--k-surface-base);
    border: 1px solid var(--k-border-subtle);
    border-radius: 10px;
    padding: 1.25rem;
    text-decoration: none;
    color: inherit;
    transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
    position: relative;
    overflow: hidden;
  }

  .tile::after {
    content: '';
    position: absolute;
    inset-inline: 0;
    inset-block-start: 0;
    block-size: 3px;
    background: transparent;
    transition: background-color 0.18s ease;
  }

  .tile:hover {
    transform: translateY(-3px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
    border-color: var(--k-border-danger);
  }

  .tile:hover::after {
    background-color: var(--k-accent-danger-bg);
  }

  .tile__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-block-end: 0.85rem;
  }

  .tile__icon-badge {
    inline-size: 2.6rem;
    block-size: 2.6rem;
    border-radius: 8px;
    background-color: var(--k-surface-raised);
    border: 1px solid var(--k-border-subtle);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--k-terracotta-600);
    transition: background-color 0.15s ease, color 0.15s ease;
  }

  .tile:hover .tile__icon-badge {
    background-color: var(--k-accent-danger-bg);
    color: var(--k-text-on-accent);
    border-color: var(--k-border-danger);
  }

  .tile__role-pill {
    font-size: 0.68rem;
    font-weight: 700;
    color: var(--k-stone-600);
    background-color: var(--k-surface-raised);
    padding: 0.2rem 0.5rem;
    border-radius: 999px;
    border: 1px solid var(--k-border-muted);
  }

  .tile__body {
    flex: 1;
    margin-block-end: 1rem;
  }

  .tile__name {
    font-size: 1.05rem;
    font-weight: 700;
    color: var(--k-text-primary);
    margin: 0;
  }

  .tile__footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-block-start: 0.75rem;
    border-block-start: 1px solid var(--k-khadi-100);
    font-size: 0.725rem;
  }

  .tile__path {
    color: var(--k-text-tertiary);
  }

  .tile__cta {
    font-weight: 700;
    color: var(--k-accent-danger-muted);
    transition: transform 0.15s ease;
  }

  .tile:hover .tile__cta {
    transform: translateX(3px);
  }
</style>