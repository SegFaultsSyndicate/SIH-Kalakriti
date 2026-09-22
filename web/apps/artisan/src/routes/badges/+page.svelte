<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { BadgeGrid, SectionHeader } from '@kalakriti/ui';
  import type { PageData } from './$types';

  interface Props { data: PageData }
  let { data }: Props = $props();

  const t = $derived(locale.t);
</script>

<svelte:head><title>{t('badges.title')}</title></svelte:head>

<main id="main-content" class="k-badges-page">
  <div class="k-badges-header">
    <SectionHeader heading={t('badges.title')} kicker={t('badges.subtitle')} />
  </div>
  <BadgeGrid catalog={data.catalog} granted={data.granted} progress={data.progress} />
</main>

<style>
  .k-badges-page {
    padding: var(--k-space-4);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    max-width: 960px;
    margin: 0 auto;
  }

  .k-badges-header {
    margin-bottom: var(--k-space-2);
  }
</style>
