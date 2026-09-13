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

  let mobileNavOpen = $state(false);

  function toggleMobileNav(): void {
    mobileNavOpen = !mobileNavOpen;
  }

  function closeMobileNav(): void {
    mobileNavOpen = false;
  }

  function onKeydown(e: KeyboardEvent): void {
    if (e.key === 'Escape') mobileNavOpen = false;
  }

  $effect(() => {
    if (!mobileNavOpen) return;
    const mq = window.matchMedia('(min-width: 861px)');
    const onChange = () => {
      if (mq.matches) mobileNavOpen = false;
    };
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  });

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
  <title>{t('app.name')}</title>
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
<svelte:window onkeydown={onKeydown} />

<div class="shell">
  <header class="shell__header">
    <button
      type="button"
      class="shell__menu-toggle"
      onclick={toggleMobileNav}
      aria-expanded={mobileNavOpen}
      aria-controls="shell-mobile-nav"
      aria-label={mobileNavOpen ? 'Close main navigation' : 'Open main navigation'}
    >
      <Icon name="menu" size="1.25rem" />
    </button>

    <a class="shell__lockup" href="/">
      <img class="shell__emblem" src="/favicon.svg" alt="" width="32" height="32" />
      <span class="shell__wordmark">
        {t('app.name')}
      </span>
    </a>

    <nav class="shell__nav" aria-label="Main Navigation">
      <a class="shell__nav-link" href="/catalog">Craft Directory</a>
      <a class="shell__nav-link" href="/gi-tagged">GI Heritage</a>
      <a class="shell__nav-link" href="/fairs">Exhibitions & Melas</a>
      <a class="shell__nav-link" href="/case-studies">Impact Studies</a>
    </nav>

    <div class="shell__actions">
      <form class="shell__search-inline" action="/search" role="search">
        <input type="search" name="q" placeholder={t('search.placeholder')} aria-label={t('nav.search')} />
      </form>
      <a class="shell__icon-link" href="/search" aria-label={t('nav.search')}>
        <Icon name="search" />
      </a>
      <a class="shell__icon-link" href="/orders" aria-label={t('buyer.orders.heading')}>
        <Icon name="package" />
      </a>

      <AccountMenu />

      <LanguageSelector />
      <AccessibilityControl statementHref="/accessibility" />
    </div>
  </header>

  {#if mobileNavOpen}
    <nav class="shell__mobile-nav" id="shell-mobile-nav" aria-label="Main Navigation (mobile)">
      <a class="shell__mobile-link" href="/catalog" onclick={closeMobileNav}>Craft Directory</a>
      <a class="shell__mobile-link" href="/gi-tagged" onclick={closeMobileNav}>GI Heritage</a>
      <a class="shell__mobile-link" href="/case-studies" onclick={closeMobileNav}>Impact Studies</a>
    </nav>
  {/if}

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
  /* Hamburger: mobile only. */
  .shell__menu-toggle {
    display: none;
    align-items: center;
    justify-content: center;
    inline-size: 2.75rem;
    block-size: 2.75rem;
    flex: none;
    border: none;
    border-radius: var(--k-radius-md);
    background: transparent;
    color: var(--k-text-primary);
    cursor: pointer;
  }

  .shell__menu-toggle:hover,
  .shell__menu-toggle[aria-expanded='true'] {
    background-color: var(--k-surface-sunken);
  }

  /* Mobile nav panel: hidden on desktop, rendered below the header. */
  .shell__mobile-nav {
    display: none;
  }

  @media (max-width: 860px) {
    .shell__menu-toggle {
      display: flex;
    }

    .shell__mobile-nav {
      display: flex;
      flex-direction: column;
      border-block-end: var(--k-hairline) solid var(--k-border-hairline);
      background: var(--k-surface-raised);
      padding: var(--k-space-2) var(--k-gutter) var(--k-space-3);
    }

    .shell__mobile-link {
      display: flex;
      align-items: center;
      min-block-size: var(--k-touch-min);
      font-size: var(--k-text-base);
      font-weight: 600;
      color: var(--k-text-primary);
      text-decoration: none;
      padding-inline: var(--k-space-2);
      border-radius: var(--k-radius-sm);
    }

    .shell__mobile-link:hover {
      color: var(--k-terracotta);
      background-color: var(--k-surface-sunken);
    }
  }

  .shell__nav {
    display: flex;
    align-items: center;
    flex-shrink: 0;
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
    color: var(--k-text-secondary);
    text-decoration: none;
    padding: 0.25rem 0.5rem;
    border-radius: var(--k-radius-sm, 4px);
    white-space: nowrap;
    transition: color 0.15s ease, background-color 0.15s ease;
  }

  .shell__nav-link:hover {
    color: var(--k-text-secondary);
    background-color: rgba(244, 240, 234, 0.9);
  }
</style>
