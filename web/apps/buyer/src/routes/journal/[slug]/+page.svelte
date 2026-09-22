<!--
  apps/buyer/src/routes/journal/[slug]/+page.svelte

  Virasat Cultural Journal essay pages. The home page's "Chronicles of
  Living Heritage" rail used to send "Read Heritage Essay" to a generic
  /search?q=... link -- there was no essay to read. These are the three
  essays it teases (ajrakh, patola, dhokra), same static-editorial pattern
  as /case-studies.
-->
<script lang="ts">
  import { page } from '$app/state';
  import { locale } from '@kalakriti/i18n';
  import { Breadcrumbs } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import { ESSAYS, essayKey, paragraphKeys } from '$lib/journal';

  const t = $derived(locale.t);

  const slug = $derived(page.params.slug ?? '');
  const essay = $derived(ESSAYS.find((e) => e.slug === slug));

  const breadcrumbs = $derived([
    { label: t('journal.breadcrumbLabel'), href: '/journal' },
    { label: (essay ? t(essayKey(essay, 'title')) : null) ?? t('journal.breadcrumbEssayFallback') },
  ]);
</script>

<svelte:head>
  <title>{essay ? t('journal.headTitleSuffix', { title: t(essayKey(essay, 'title')) }) : t('journal.headTitleFallback')}</title>
  {#if essay}
    <meta name="description" content={t(essayKey(essay, 'lead'))} />
  {/if}
</svelte:head>

{#if !essay}
  <div class="essay-page">
    <div class="essay-container">
      <Breadcrumbs items={[{ label: t('journal.notFoundBreadcrumb') }]} />
      <h1>{t('journal.notFoundHeading')}</h1>
      <p><a href="/">{t('journal.returnHome')}</a></p>
    </div>
  </div>
{:else}
  <div class="essay-page">
    <div class="essay-container">
      <Breadcrumbs items={breadcrumbs} homeLabel={t('nav.marketplace')} />

      <header class="essay-header">
        <span class="essay-kicker">{t('journal.kickerPrefix', { cluster: t(essayKey(essay, 'cluster')).toLocaleUpperCase() })}</span>
        <h1 class="essay-title">{t(essayKey(essay, 'title'))}</h1>
        <p class="essay-lead">{t(essayKey(essay, 'lead'))}</p>
        <div class="essay-meta">
          <span>{t(essayKey(essay, 'author'))}</span>
          <span class="dot">•</span>
          <span>{t(essayKey(essay, 'readTime'))}</span>
          <span class="dot">•</span>
          <span>{t(essayKey(essay, 'region'))}</span>
        </div>
      </header>

      <div class="essay-hero">
        <img src={essay.heroImage} alt={t(essayKey(essay, 'title'))} loading="lazy" />
      </div>

      <div class="essay-body">
        {#each paragraphKeys(essay) as key (key)}
          <p>{t(key)}</p>
        {/each}
      </div>

      <blockquote class="essay-quote">
        <p>"{t(essayKey(essay, 'quote'))}"</p>
        <footer>{t(essayKey(essay, 'quoteAuthor'))}</footer>
      </blockquote>

      <div class="essay-actions">
        <a href="/" class="essay-back-link">
          <Icon name="arrow-right" size="0.85rem" />
          <span>{t('journal.backToHome')}</span>
        </a>
      </div>
    </div>
  </div>
{/if}

<style>
  .essay-page {
    padding-block: var(--k-space-6) var(--k-space-12);
    background: var(--k-surface-base);
  }

  .essay-container {
    max-inline-size: 42rem;
    margin-inline: auto;
    padding-inline: var(--k-space-4);
  }

  .essay-header {
    margin-block: var(--k-space-6) var(--k-space-5);
  }

  .essay-kicker {
    display: inline-block;
    font-size: 0.72rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    color: var(--k-accent-primary-text);
    margin-block-end: var(--k-space-2);
  }

  .essay-title {
    font-family: var(--k-font-display, serif);
    font-size: clamp(1.6rem, 4vw, 2.2rem);
    font-weight: 800;
    color: var(--k-text-primary);
    line-height: 1.2;
    margin: 0 0 var(--k-space-3);
  }

  .essay-lead {
    font-size: var(--k-text-md);
    color: var(--k-text-secondary);
    line-height: var(--k-leading-normal);
    margin: 0 0 var(--k-space-3);
  }

  .essay-meta {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2);
    font-size: var(--k-text-sm);
    color: var(--k-text-secondary);
  }

  .essay-meta .dot {
    opacity: 0.5;
  }

  .essay-hero {
    border-radius: var(--k-radius-lg);
    overflow: hidden;
    aspect-ratio: 16 / 9;
    margin-block-end: var(--k-space-6);
  }

  .essay-hero img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
  }

  .essay-body {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
    font-size: var(--k-text-base);
    line-height: 1.75;
    color: var(--k-text-primary);
  }

  .essay-quote {
    margin: var(--k-space-7) 0;
    padding: var(--k-space-5);
    background: var(--k-surface-base);
    border-inline-start: 4px solid var(--k-border-accent);
    border-radius: 0 var(--k-radius-md) var(--k-radius-md) 0;
  }

  .essay-quote p {
    font-size: var(--k-text-md);
    font-style: italic;
    color: var(--k-stone-700);
    line-height: var(--k-leading-normal);
    margin: 0 0 var(--k-space-2);
  }

  .essay-quote footer {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .essay-actions {
    border-block-start: 1px solid var(--k-border-subtle);
    padding-block-start: var(--k-space-5);
  }

  .essay-back-link {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-sm);
    font-weight: 700;
    color: var(--k-accent-primary-text);
    text-decoration: none;
  }

  .essay-back-link :global(svg) {
    transform: rotate(180deg);
  }
</style>
