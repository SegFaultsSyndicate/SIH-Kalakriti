<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { BadgeGrid } from '@kalakriti/ui';
  import { SELF_BADGES, loadSelfBadges, saveSelfBadges } from '$lib/self-badges';
  import type { PageData } from './$types';

  interface Props { data: PageData }
  let { data }: Props = $props();

  const t = $derived(locale.t);

  let applied = $state<string[]>([]);
  $effect(() => {
    void loadSelfBadges().then((ids) => (applied = ids));
  });

  function toggle(id: string): void {
    applied = applied.includes(id) ? applied.filter((x) => x !== id) : [...applied, id];
    void saveSelfBadges(applied);
  }
</script>

<svelte:head><title>{t('badges.title')}</title></svelte:head>

<div class="k-badges-page">
  <header class="k-badges-header">
    <h1>{t('badges.title')}</h1>
    <p>{t('badges.subtitle')}</p>
  </header>

  <section class="self-badges" aria-labelledby="self-badges-heading">
    <h2 id="self-badges-heading">{t('badges.self.heading')}</h2>
    <p class="self-badges__hint">{t('badges.self.hint')}</p>
    <ul class="self-badges__list" role="list">
      {#each SELF_BADGES as badge (badge.id)}
        {@const on = applied.includes(badge.id)}
        <li>
          <button
            type="button"
            class="self-badge"
            class:self-badge--on={on}
            aria-pressed={on}
            onclick={() => toggle(badge.id)}
          >
            <span class="self-badge__icon"><Icon name={badge.icon} size="1.1rem" /></span>
            <span class="self-badge__label">{t(badge.labelKey)}</span>
            <span class="self-badge__check" aria-hidden="true">
              <Icon name={on ? 'check' : 'plus'} size="0.9rem" />
            </span>
          </button>
        </li>
      {/each}
    </ul>
  </section>

  <BadgeGrid catalog={data.catalog} granted={data.granted} progress={data.progress} />
</div>

<style>
  .k-badges-page {
    padding-block: var(--k-space-4);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-5);
    max-width: 960px;
    margin: 0 auto;
  }

  .k-badges-header {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .k-badges-header p,
  .self-badges__hint {
    margin: 0;
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .self-badges {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    padding: var(--k-space-4);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: 1rem;
    background: var(--k-surface-raised);
  }

  .self-badges h2 {
    font-size: var(--k-text-md);
    margin: 0;
  }

  .self-badges__list {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(min(100%, 13rem), 1fr));
    gap: var(--k-space-2);
    margin: var(--k-space-2) 0 0;
    padding: 0;
    list-style: none;
  }

  .self-badge {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    inline-size: 100%;
    min-block-size: var(--k-touch-min);
    padding: var(--k-space-2) var(--k-space-3) var(--k-space-2) var(--k-space-2);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-pill);
    background: var(--k-surface-base);
    color: var(--k-text-primary);
    font-size: var(--k-text-sm);
    text-align: start;
    cursor: pointer;
    transition: background-color 0.15s ease, border-color 0.15s ease;
  }

  .self-badge:hover {
    border-color: var(--k-accent-primary-bg);
  }

  .self-badge__icon {
    display: grid;
    place-items: center;
    flex-shrink: 0;
    inline-size: 2rem;
    block-size: 2rem;
    border-radius: var(--k-radius-pill);
    background: var(--k-surface-sunken);
    color: var(--k-accent-primary-text);
  }

  .self-badge__label {
    flex: 1;
    min-inline-size: 0;
  }

  .self-badge__check {
    display: grid;
    place-items: center;
    flex-shrink: 0;
    inline-size: 1.5rem;
    block-size: 1.5rem;
    border-radius: var(--k-radius-pill);
    border: var(--k-hairline) solid var(--k-border-interactive);
    color: var(--k-text-secondary);
  }

  .self-badge--on {
    border-color: var(--k-accent-primary-bg);
    background: color-mix(in srgb, var(--k-accent-primary-bg) 9%, var(--k-surface-base));
    font-weight: var(--k-weight-semibold);
  }

  .self-badge--on .self-badge__icon {
    background: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
  }

  .self-badge--on .self-badge__check {
    border-color: var(--k-accent-primary-bg);
    background: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
  }
</style>
