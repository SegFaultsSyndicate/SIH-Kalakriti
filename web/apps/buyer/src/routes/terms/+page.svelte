<!--
  apps/buyer/src/routes/terms/+page.svelte

  Official Terms of Service for the Kalakriti Cultural Marketplace.
  Governed under Ministry of Social Justice & Empowerment standards.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Breadcrumbs } from '@kalakriti/ui';

  const t = $derived(locale.t);

  const breadcrumbs = $derived([
    { label: t('nav.home') || 'Marketplace', href: '/' },
    { label: t('terms.breadcrumbLabel') },
  ]);
</script>

<svelte:head>
  <title>{t('terms.headTitle', { appName: t('app.name') })}</title>
  <meta name="description" content={t('terms.metaDescription')} />
</svelte:head>

<div class="legal-page">
  <div class="legal-container">
    <Breadcrumbs items={breadcrumbs} homeLabel="Marketplace" />

    <header class="legal-header">
      <span class="legal-kicker">{t('terms.kicker')}</span>
      <h1 class="legal-title">{t('terms.title')}</h1>
      <p class="legal-date">{t('terms.lastUpdated')}</p>
    </header>

    <div class="legal-content">
      <section class="legal-section">
        <h2>{t('terms.section1.heading')}</h2>
        <p>{t('terms.section1.body')}</p>
      </section>

      <section class="legal-section">
        <h2>{t('terms.section2.heading')}</h2>
        <p>{t('terms.section2.body')}</p>
      </section>

      <section class="legal-section">
        <h2>{t('terms.section3.heading')}</h2>
        <p>{t('terms.section3.body')}</p>
      </section>

      <section class="legal-section">
        <h2>{t('terms.section4.heading')}</h2>
        <p>{t('terms.section4.body')}</p>
      </section>

      <section class="legal-section">
        <h2>{t('terms.section5.heading')}</h2>
        <p>{t('terms.section5.body')}</p>
      </section>

      <section class="legal-section">
        <h2>{t('terms.section6.heading')}</h2>
        <p>{t('terms.section6.body')}</p>
      </section>
    </div>
  </div>
</div>

<style>
  .legal-page {
    padding-block: var(--k-space-6) var(--k-space-12);
  }

  .legal-container {
    max-inline-size: 52rem;
    margin-inline: auto;
    padding-inline: var(--k-space-4);
  }

  .legal-header {
    margin-block-end: var(--k-space-6);
    border-block-end: 1px solid var(--k-border-subtle);
    padding-block-end: var(--k-space-4);
  }

  .legal-kicker {
    display: inline-block;
    font-size: var(--k-text-xs);
    font-weight: var(--k-weight-bold);
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--k-accent-secondary);
    margin-block-end: var(--k-space-1);
  }

  .legal-title {
    font-family: var(--k-font-display, Georgia, serif);
    font-size: var(--k-text-3xl);
    font-weight: var(--k-weight-bold);
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-2) 0;
  }

  .legal-date {
    font-size: var(--k-text-xs);
    color: var(--k-text-muted);
    margin: 0;
  }

  .legal-content {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-6);
    line-height: 1.7;
    color: var(--k-text-secondary);
  }

  .legal-section h2 {
    font-size: var(--k-text-base);
    font-weight: var(--k-weight-bold);
    color: var(--k-text-primary);
    margin: 0 0 var(--k-space-2) 0;
  }

  .legal-section p {
    font-size: var(--k-text-sm);
    margin: 0;
  }
</style>
