<!--
  apps/buyer/src/routes/journal/+page.svelte

  The Virasat Journal index -- the footer's "Indiahandmade & Guild Blog".
  Lists every essay in $lib/journal, newest first, and links the RSS feed.
-->
<script lang="ts">
  import { locale, formatDate } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';
  import { ESSAYS, essayKey } from '$lib/journal';

  const t = $derived(locale.t);
  const essays = [...ESSAYS].sort((a, b) => b.published.localeCompare(a.published));
</script>

<svelte:head>
  <title>{t('journal.breadcrumbLabel')} — {t('app.name')}</title>
  <meta name="description" content={t('home.journal.subheading')} />
  <link rel="alternate" type="application/rss+xml" title={t('journal.breadcrumbLabel')} href="/rss.xml" />
</svelte:head>

<div class="journal-index">
  <header class="journal-index__header">
    <span class="journal-index__kicker">{t('home.journal.kicker')}</span>
    <h1>{t('home.journal.heading')}</h1>
    <p>{t('home.journal.subheading')}</p>
    <a class="journal-index__rss" href="/rss.xml">
      <Icon name="share" size="0.9rem" />
      <span>{t('journal.index.rss')}</span>
    </a>
  </header>

  <ul class="journal-index__list" role="list">
    {#each essays as essay (essay.slug)}
      <li>
        <a class="essay-card" href="/journal/{essay.slug}">
          <img src={essay.heroImage} alt="" loading="lazy" />
          <div class="essay-card__body">
            <span class="essay-card__meta">
              {t(essayKey(essay, 'cluster'))} · <time datetime={essay.published}>{formatDate(essay.published, locale.code, { dateStyle: 'long' })}</time>
            </span>
            <h2>{t(essayKey(essay, 'title'))}</h2>
            <p>{t(essayKey(essay, 'lead'))}</p>
            <span class="essay-card__byline">{t(essayKey(essay, 'author'))} · {t(essayKey(essay, 'readTime'))}</span>
          </div>
        </a>
      </li>
    {/each}
  </ul>
</div>

<style>
  .journal-index {
    max-inline-size: 64rem;
    margin-inline: auto;
    padding: var(--k-space-6) var(--k-space-4) var(--k-space-12);
  }

  .journal-index__header {
    margin-block-end: var(--k-space-6);
  }

  .journal-index__kicker {
    font-size: 0.75rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--k-accent-primary-text);
  }

  .journal-index__header h1 {
    font-family: var(--k-font-display, serif);
    font-size: clamp(1.6rem, 4vw, 2.2rem);
    margin: var(--k-space-2) 0;
  }

  .journal-index__header p {
    color: var(--k-text-secondary);
    max-inline-size: 44rem;
    margin: 0 0 var(--k-space-3);
  }

  .journal-index__rss {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-2);
    font-size: var(--k-text-sm);
    font-weight: 700;
    color: var(--k-accent-primary-text);
  }

  .journal-index__list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: var(--k-space-5);
  }

  .essay-card {
    display: grid;
    grid-template-columns: minmax(0, 16rem) 1fr;
    gap: var(--k-space-4);
    color: inherit;
    text-decoration: none;
    border: 1px solid var(--k-border-subtle);
    border-radius: var(--k-radius-lg);
    overflow: hidden;
    background: var(--k-surface-raised, var(--k-surface-base));
  }

  @media (max-width: 40rem) {
    .essay-card {
      grid-template-columns: 1fr;
    }
  }

  .essay-card img {
    inline-size: 100%;
    block-size: 100%;
    aspect-ratio: 4 / 3;
    object-fit: cover;
  }

  .essay-card__body {
    padding: var(--k-space-4);
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .essay-card__meta,
  .essay-card__byline {
    font-size: var(--k-text-xs);
    color: var(--k-text-secondary);
  }

  .essay-card h2 {
    font-size: var(--k-text-lg);
    margin: 0;
  }

  .essay-card p {
    margin: 0;
    color: var(--k-text-secondary);
  }

  .essay-card:hover h2 {
    color: var(--k-accent-primary-text);
  }
</style>
