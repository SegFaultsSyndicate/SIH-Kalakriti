<!--
  apps/admin/src/lib/Breadcrumb.svelte

  Derived from the route's own path segments, matched back to NAV_ITEMS for
  a label where one exists; an unmatched segment (an id in the URL) falls
  back to its raw text rather than disappearing.
-->
<script lang="ts">
  import { page } from '$app/stores';
  import { locale } from '@kalakriti/i18n';
  import { NAV_ITEMS } from './nav';

  const t = $derived(locale.t);

  const segments = $derived(
    $page.url.pathname
      .split('/')
      .filter((s) => s.length > 0)
      .map((segment, i, all) => {
        const href = '/' + all.slice(0, i + 1).join('/');
        const item = NAV_ITEMS.find((n) => n.href === href);
        return { href, label: item ? t(item.labelKey) : decodeURIComponent(segment) };
      }),
  );
</script>

<nav aria-label={t('breadcrumb.label')} class="breadcrumb">
  <ol class="breadcrumb__list">
    <li><a href="/">{t('nav.dashboard')}</a></li>
    {#each segments as segment, i (segment.href)}
      <li aria-current={i === segments.length - 1 ? 'page' : undefined}>
        {#if i === segments.length - 1}
          <span>{segment.label}</span>
        {:else}
          <a href={segment.href}>{segment.label}</a>
        {/if}
      </li>
    {/each}
  </ol>
</nav>

<style>
  .breadcrumb {
    margin-block-end: var(--k-space-4);
  }

  .breadcrumb__list {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--k-space-1);
    list-style: none;
    margin: 0;
    padding: 0;
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .breadcrumb__list li + li::before {
    content: '/';
    margin-inline-end: var(--k-space-1);
    color: var(--k-text-tertiary, var(--k-text-secondary));
  }

  .breadcrumb__list a {
    color: var(--k-text-secondary);
  }

  .breadcrumb__list li[aria-current='page'] span {
    color: var(--k-text-primary);
    font-weight: var(--k-weight-medium);
  }
</style>
