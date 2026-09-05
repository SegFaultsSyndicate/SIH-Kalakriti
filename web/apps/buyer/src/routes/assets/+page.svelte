<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import IdentityShowcase from '@kalakriti/identity/IdentityShowcase.svelte';
  import { HeroBackdrop } from '@kalakriti/illustrations';
  import { Illustration } from '@kalakriti/illustrations';
  import { Divider } from '@kalakriti/ornament';
  import { Section } from '@kalakriti/patterns';
  import printTag from '../../../../../packages/print/src/tag-a4-colour.svg?url';

  const t = $derived(locale.t);
</script>

<svelte:head>
  <title>{t('assets.title')} — {t('app.name')}</title>
</svelte:head>

<HeroBackdrop element="div" class="asset-page__hero">
  <p class="asset-page__kicker">{t('assets.kicker')}</p>
  <h1>{t('assets.title')}</h1>
  <p>{t('assets.body')}</p>
</HeroBackdrop>

<Divider variant="jaali-octstar" density="fine" />

<Section variant="khadi-plain">
  <IdentityShowcase title={t('assets.identityTitle')} />
</Section>

<section class="asset-page__gallery" aria-labelledby="asset-gallery-heading">
  <h2 id="asset-gallery-heading">{t('assets.illustrationsTitle')}</h2>
  <div class="asset-page__illustrations">
    <Illustration name="feature-provenance" title={t('assets.provenanceIllustration')} />
    <Illustration name="feature-fair-pricing" title={t('assets.fairPricingIllustration')} />
    <Illustration name="onboard-find-buyers" title={t('assets.buyersIllustration')} />
    <Illustration name="onboard-photograph-work" title={t('assets.photographIllustration')} />
  </div>
</section>

<section class="asset-page__links" aria-labelledby="asset-links-heading">
  <h2 id="asset-links-heading">{t('assets.linksTitle')}</h2>
  <a href={printTag} download="kalakriti-tag-a4-colour.svg">{t('assets.printTagLink')}</a>
  <a href="/docs/index.html">{t('assets.docsLink')}</a>
  <a href="/contact-sheet-index.html">{t('assets.contactSheetsLink')}</a>
  <a href="/docs/pitch-summary.html">{t('assets.pitchLink')}</a>
</section>

<style>
  :global(.asset-page__hero) {
    display: grid;
    gap: var(--k-space-3);
    padding: var(--k-space-7) var(--k-space-5);
    color: var(--k-text-primary);
  }

  .asset-page__kicker {
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    text-transform: lowercase;
  }

  :global(.asset-page__hero h1) {
    max-inline-size: var(--k-measure-narrow);
    font-size: var(--k-text-3xl);
  }

  :global(.asset-page__hero p:last-child) {
    max-inline-size: var(--k-measure-readable);
    color: var(--k-text-secondary);
  }

  .asset-page__links {
    display: grid;
    gap: var(--k-space-3);
    padding-block: var(--k-space-6);
  }

  .asset-page__gallery {
    display: grid;
    gap: var(--k-space-4);
    padding-block: var(--k-space-6);
  }

  .asset-page__illustrations {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
    gap: var(--k-space-4);
  }

  .asset-page__illustrations :global(svg) {
    inline-size: 100%;
    block-size: auto;
  }

  .asset-page__links h2 {
    font-size: var(--k-text-lg);
  }
</style>
