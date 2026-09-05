<!-- apps/buyer/src/routes/accessibility/+page.svelte -->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { AccessibilityStatement } from '@kalakriti/ui';

  const t = $derived(locale.t);
</script>

<svelte:head>
  <title>{t('a11y.statement.title')} — {t('app.name')}</title>
</svelte:head>

<article class="statement">
  <AccessibilityStatement />
  <section aria-labelledby="buyer-context">
    <h2 id="buyer-context">{t('a11y.buyer.context.title')}</h2>
    <p>{t('a11y.buyer.context.body')}</p>
  </section>
  <p class="statement__contact">
    <a href="mailto:accessibility@kalakriti.gov.in">{t('a11y.statement.contact.email')}</a>
  </p>
</article>

<style>
  .statement {
    display: grid;
    gap: var(--k-space-4);
  }
  .statement :global(h1) {
    margin-block-end: 0;
  }
  .statement section {
    border-block-start: var(--k-hairline) solid var(--k-border-hairline);
    padding-block-start: var(--k-space-4);
  }
  .statement__contact {
    margin-block-start: var(--k-space-2);
  }
</style>
