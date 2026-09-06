<!--
  packages/ui/src/Breadcrumbs.svelte

  Accessible navigation breadcrumb trail adhering to the Kalakriti design system.
  Uses semantic <nav aria-label="Breadcrumb">, ordered list, hairline chevron dividers,
  and clean typography with zero AI fluff.
-->
<script lang="ts">
  export interface BreadcrumbItem {
    label: string;
    href?: string;
  }

  interface Props {
    items: BreadcrumbItem[];
    homeLabel?: string;
  }

  let { items, homeLabel = 'Home' }: Props = $props();
</script>

<nav class="k-breadcrumbs" aria-label="Breadcrumb">
  <ol class="k-breadcrumbs__list">
    <li class="k-breadcrumbs__item">
      <a href="/" class="k-breadcrumbs__link">
        <svg class="k-breadcrumbs__home-icon" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
          <polyline points="9 22 9 12 15 12 15 22" />
        </svg>
        <span>{homeLabel}</span>
      </a>
    </li>

    {#each items as item, index (item.label + index)}
      {@const isLast = index === items.length - 1}
      <li class="k-breadcrumbs__separator" aria-hidden="true">
        <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="9 18 15 12 9 6" />
        </svg>
      </li>

      <li class="k-breadcrumbs__item">
        {#if isLast || !item.href}
          <span class="k-breadcrumbs__current" aria-current="page">{item.label}</span>
        {:else}
          <a href={item.href} class="k-breadcrumbs__link">{item.label}</a>
        {/if}
      </li>
    {/each}
  </ol>
</nav>

<style>
  .k-breadcrumbs {
    padding-block: var(--k-space-2);
    margin-block-end: var(--k-space-3);
  }

  .k-breadcrumbs__list {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    list-style: none;
    padding: 0;
    margin: 0;
    font-size: var(--k-text-xs);
    line-height: 1.4;
    color: var(--k-text-muted);
  }

  .k-breadcrumbs__item {
    display: inline-flex;
    align-items: center;
  }

  .k-breadcrumbs__separator {
    display: inline-flex;
    align-items: center;
    padding-inline: var(--k-space-2);
    color: var(--k-border-subtle);
  }

  .k-breadcrumbs__link {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-1);
    color: var(--k-text-secondary);
    text-decoration: none;
    transition: color var(--k-duration-fast) ease;
  }

  .k-breadcrumbs__link:hover {
    color: var(--k-accent-secondary);
    text-decoration: underline;
  }

  .k-breadcrumbs__current {
    font-weight: var(--k-weight-semibold);
    color: var(--k-text-primary);
  }

  .k-breadcrumbs__home-icon {
    flex-shrink: 0;
  }
</style>
