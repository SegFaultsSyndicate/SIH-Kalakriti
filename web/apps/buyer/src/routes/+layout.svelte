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
  import {
    session,
    restoreAccessToken,
    setSessionRefreshHandler,
    setUnauthorizedHandler,
    createLoginRedirectHandler,
  } from '@kalakriti/api';
  import { goto } from '$app/navigation';
  import BuyerFooter from '$lib/BuyerFooter.svelte';
  import AccountMenu from '$lib/AccountMenu.svelte';
  import CategorySubnav from '$lib/CategorySubnav.svelte';

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
  $effect(() => {
    setUnauthorizedHandler(createLoginRedirectHandler((url) => goto(url)));
    setSessionRefreshHandler((token) => session.establish(token));
    return () => {
      setUnauthorizedHandler(undefined);
      setSessionRefreshHandler(undefined);
    };
  });

  $effect(() => {
    void (async () => {
      const token = await restoreAccessToken();
      if (token) session.establish(token);
    })();
  });

  // registerType is 'autoUpdate', so registration is all this needs to do:
  // a new worker takes over on the next navigation with no prompt. Correct
  // here and deliberately not what the artisan app does.
  onMount(async () => {
    // @ts-expect-error virtual:pwa-register is injected by vite-plugin-pwa at build time.
    const { registerSW } = await import('virtual:pwa-register');
    registerSW({ immediate: true });
  });
</script>

<svelte:head>
  <title>{t('app.name')} - Ministry of Social Justice & Empowerment</title>
  <meta name="description" content="Kalakriti is India's national AI cataloging, cryptographic GI provenance, and collective fulfillment marketplace for master artisans and heritage looms." />
  <script type="application/ld+json">
    {
      "@context": "https://schema.org",
      "@type": "GovernmentOrganization",
      "name": "Kalakriti - Ministry of Social Justice & Empowerment",
      "url": "https://buyer.kalakriti.gov.in",
      "logo": "https://buyer.kalakriti.gov.in/favicon.svg",
      "description": "National AI cataloging and collective fulfillment platform for marginalized artisans in India."
    }
  </script>
  {#if env.PUBLIC_GA_ID}
    <script async src={`https://www.googletagmanager.com/gtag/js?id=${env.PUBLIC_GA_ID}`}></script>
    <script>
      window.dataLayer = window.dataLayer || [];
      function gtag(){dataLayer.push(arguments);}
      gtag('js', new Date());
      gtag('config', '{env.PUBLIC_GA_ID}');
    </script>
  {/if}
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

    <nav class="shell__nav" aria-label="Main Navigation">
      <a class="shell__nav-link" href="/catalog">Craft Directory</a>
      <a class="shell__nav-link" href="/gi-tagged">GI Heritage</a>
      <a class="shell__nav-link" href="/case-studies">Impact Studies</a>
    </nav>

    <div class="shell__actions">
      <a class="shell__icon-link" href="/search" aria-label={t('nav.search')}>
        <Icon name="search" />
      </a>
      <a class="shell__icon-link" href="/orders" aria-label={t('buyer.orders.heading')}>
        <Icon name="collective-order" />
      </a>
      
      <AccountMenu />

      <LanguageSelector />
      <AccessibilityControl statementHref="/accessibility" />
    </div>
  </header>

  <CategorySubnav />

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

  <BuyerFooter />
</div>

<style>
  .shell__nav {
    display: flex;
    align-items: center;
    gap: var(--k-space-5, 1.25rem);
    margin-inline-start: var(--k-space-6, 1.5rem);
  }

  @media (max-width: 860px) {
    .shell__nav {
      display: none;
    }
  }

  .shell__nav-link {
    font-size: var(--k-text-sm, 0.875rem);
    font-weight: 600;
    color: var(--k-text-secondary, #6b635b);
    text-decoration: none;
    padding: 0.25rem 0.5rem;
    border-radius: var(--k-radius-sm, 4px);
    transition: color 0.15s ease, background-color 0.15s ease;
  }

  .shell__nav-link:hover {
    color: var(--k-terracotta, #b84a39);
    background-color: var(--k-surface-sunken, #f5f2eb);
  }
</style>
