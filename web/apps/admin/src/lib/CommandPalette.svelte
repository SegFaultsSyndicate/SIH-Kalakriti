<!--
  apps/admin/src/lib/CommandPalette.svelte

    <CommandPalette bind:open={paletteOpen} />

  Cmd/Ctrl-K navigation and search over the same NAV_ITEMS the sidebar
  renders -- there is no separate search index, this is a filter over a
  dozen known destinations.
-->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { Dialog } from '@kalakriti/ui';
  import { NAV_ITEMS } from './nav';

  interface Props {
    open?: boolean;
  }

  let { open = $bindable(false) }: Props = $props();

  const t = $derived(locale.t);
  let query = $state('');
  let activeIndex = $state(0);
  let inputEl: HTMLInputElement | undefined = $state();

  const results = $derived(
    NAV_ITEMS.filter((item) => t(item.labelKey).toLowerCase().includes(query.toLowerCase())),
  );

  $effect(() => {
    activeIndex = 0;
  });

  $effect(() => {
    if (open) requestAnimationFrame(() => inputEl?.focus());
    else query = '';
  });

  function select(href: string): void {
    open = false;
    void goto(href);
  }

  function onKeydown(e: KeyboardEvent): void {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      activeIndex = Math.min(activeIndex + 1, results.length - 1);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      activeIndex = Math.max(activeIndex - 1, 0);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      const item = results[activeIndex];
      if (item) select(item.href);
    }
  }
</script>

<svelte:window
  onkeydown={(e) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault();
      open = !open;
    }
  }}
/>

<Dialog bind:open title={t('palette.title')}>
  <div class="palette">
    <input
      bind:this={inputEl}
      bind:value={query}
      onkeydown={onKeydown}
      type="text"
      class="palette__input"
      placeholder={t('palette.placeholder')}
      aria-label={t('palette.placeholder')}
      role="combobox"
      aria-expanded="true"
      aria-controls="palette-listbox"
    />
    <ul id="palette-listbox" class="palette__list" role="listbox">
      {#each results as item, i (item.href)}
        <li role="option" aria-selected={i === activeIndex}>
          <button
            type="button"
            class="palette__item"
            class:palette__item--active={i === activeIndex}
            onclick={() => select(item.href)}
          >
            <Icon name={item.icon} />
            {t(item.labelKey)}
          </button>
        </li>
      {:else}
        <li class="palette__empty">{t('palette.empty')}</li>
      {/each}
    </ul>
  </div>
</Dialog>

<style>
  .palette {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-3);
    min-inline-size: min(28rem, 90vw);
  }

  .palette__input {
    font: inherit;
    font-size: var(--k-text-lg);
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    background: var(--k-surface-base);
    color: var(--k-text-primary);
  }

  .palette__list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    max-block-size: 20rem;
    overflow-y: auto;
  }

  .palette__item {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    inline-size: 100%;
    padding: var(--k-space-2) var(--k-space-3);
    border: none;
    border-radius: var(--k-radius-sm);
    background: transparent;
    color: var(--k-text-primary);
    font: inherit;
    text-align: start;
    cursor: pointer;
  }

  .palette__item--active {
    background: var(--k-surface-sunken);
  }

  .palette__empty {
    padding: var(--k-space-3);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }
</style>
