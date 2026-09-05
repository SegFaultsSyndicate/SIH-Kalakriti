<!--
  apps/buyer/src/routes/+layout.svelte

  Marketplace shell. Batch 1 establishes the government-portal lockup, the
  skip link and the container; discovery, search and the editorial home
  sections are Batch 9.
-->
<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { locale } from '@kalakriti/i18n';
  import { SkipLink, RouteAnnouncer, AccessibilityControl, LanguageSelector, a11y } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import { ErrorBoundary } from '@kalakriti/observability';
  import { env } from '$env/dynamic/public';

  interface Props {
    children: import('svelte').Snippet;
  }

  let { children }: Props = $props();

  const t = $derived(locale.t);

  $effect(() => {
    void locale.init();
  });
  $effect(() => {
    void a11y.init();
  });
  $effect(() => a11y.start());

  // registerType is 'autoUpdate', so registration is all this needs to do:
  // a new worker takes over on the next navigation with no prompt. Correct
  // here and deliberately not what the artisan app does.
  onMount(async () => {
    const { registerSW } = await import('virtual:pwa-register');
    registerSW({ immediate: true });
  });
</script>

<svelte:head>
  <title>{t('app.name')}</title>
</svelte:head>

<SkipLink target="main-content" />
<RouteAnnouncer />

<div class="shell">
  <header class="shell__header">
    <a class="shell__lockup" href="/">
      <img class="shell__emblem" src="/favicon.svg" alt="" width="32" height="32" />
      <span class="shell__wordmark">
        {t('app.name')}
        <span class="shell__ministry">{t('app.ministry')}</span>
      </span>
    </a>

    <div class="shell__actions">
      <a class="shell__icon-link" href="/search" aria-label={t('nav.search')}>
        <Icon name="search" />
      </a>
      <a class="shell__icon-link" href="/orders" aria-label={t('buyer.orders.heading')}>
        <Icon name="collective-order" />
      </a>
      <LanguageSelector />
      <AccessibilityControl statementHref="/accessibility" />
    </div>
  </header>

  <main class="shell__main" id="main-content" tabindex="-1">
    <ErrorBoundary
      source="buyer-shell"
      dsn={env.PUBLIC_SENTRY_DSN}
      title={t('error.boundary.title')}
      body={t('error.boundary.body')}
      retryLabel={t('error.boundary.retry')}
    >
      {@render children()}
    </ErrorBoundary>
  </main>

  <footer class="shell__footer">
    <p>{t('app.ministry')}</p>
    <a href="/bulk-order">{t('bulkOrder.heading')}</a>
  </footer>
</div>
