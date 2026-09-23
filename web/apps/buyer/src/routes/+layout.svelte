<!--
  apps/buyer/src/routes/+layout.svelte

  Marketplace shell. Batch 1 establishes the government-portal lockup, the
  skip link and the container; discovery, search and the editorial home
  sections are Batch 9.
-->
<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { page } from '$app/state';
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
    setAcceptLanguage,
    suggest,
  } from '@kalakriti/api';
  import { goto } from '$app/navigation';
  import BuyerFooter from '$lib/BuyerFooter.svelte';
  import AccountMenu from '$lib/AccountMenu.svelte';
  import CategorySubnav from '$lib/CategorySubnav.svelte';
  import { currency } from '$lib/currency.svelte';
  import { fuzzySuggest } from '$lib/fuzzy-search';
  import { ARTISAN_CRAFT_CATEGORIES } from '$lib/craft-categories';

  interface Props {
    children: import('svelte').Snippet;
  }

  let { children }: Props = $props();

  const t = $derived(locale.t);
  const tt = $derived(locale.tooltip);

  // Gates {@render children()} below: locale.init() is async (it awaits a
  // dynamic catalogue import), so without this a page's first paint runs
  // with an empty catalogue -- every t() call falls through to English --
  // then re-renders in the real language a tick later. That flash is what
  // "some things are getting changed their wordings upon changing the
  // language" was actually describing; see I18N_PLAN.md's F-1.
  let localeReady = $state(false);

  let mobileNavOpen = $state(false);

  // Header hides on scroll-down past HIDE_AT, and only reappears once the
  // user has scrolled back up REVEAL_DELTA px -- a small upward flick
  // shouldn't yank it back in, only a deliberate scroll-up does.
  const HIDE_AT = 80;
  const REVEAL_DELTA = 60;
  let headerHidden = $state(false);
  let stickyHeight = $state(0);
  let lastScrollY = 0;
  let upAccum = 0;

  function onScroll(): void {
    const y = window.scrollY;
    const dy = y - lastScrollY;
    if (dy > 0) {
      upAccum = 0;
      if (y > HIDE_AT) headerHidden = true;
    } else if (dy < 0) {
      upAccum += -dy;
      if (upAccum > REVEAL_DELTA || y <= HIDE_AT) headerHidden = false;
    }
    lastScrollY = y;
  }

  function toggleMobileNav(): void {
    mobileNavOpen = !mobileNavOpen;
  }

  function closeMobileNav(): void {
    mobileNavOpen = false;
  }

  function onKeydown(e: KeyboardEvent): void {
    if (e.key === 'Escape') mobileNavOpen = false;
  }

  // Top-bar nav links stay highlighted while their route is active, same
  // "you are here" affordance as the pills in CategorySubnav below.
  function isCurrentRoute(href: string): boolean {
    return page.url.pathname === href || page.url.pathname.startsWith(href + '/');
  }

  // Predictive/typo-tolerant suggestions for the always-visible header
  // search box -- the box a buyer sees on every page, not just /search's own
  // bar. Same pattern as search/+page.svelte's suggestion effect: instant
  // local fuzzy matches first, then the real suggest() index once it lands.
  const HEADER_SEARCH_TERMS = ARTISAN_CRAFT_CATEGORIES.map((c) => c.name);
  let headerQuery = $state('');
  let headerSuggestions = $state<string[]>([]);
  let showHeaderSuggestions = $state(false);
  let headerSuggestTimer: ReturnType<typeof setTimeout> | undefined;

  function onHeaderSearchInput(): void {
    clearTimeout(headerSuggestTimer);
    const trimmed = headerQuery.trim();
    if (trimmed.length < 2) {
      headerSuggestions = [];
      showHeaderSuggestions = false;
      return;
    }
    const localMatches = fuzzySuggest(trimmed, HEADER_SEARCH_TERMS, 6);
    headerSuggestions = localMatches;
    showHeaderSuggestions = localMatches.length > 0;
    headerSuggestTimer = setTimeout(async () => {
      try {
        const res = await suggest(trimmed);
        const apiSuggestions = res.suggestions ?? [];
        const next = apiSuggestions.length > 0 ? apiSuggestions : fuzzySuggest(trimmed, HEADER_SEARCH_TERMS, 6);
        headerSuggestions = next;
        showHeaderSuggestions = next.length > 0;
      } catch {
        const fallback = fuzzySuggest(trimmed, HEADER_SEARCH_TERMS, 6);
        headerSuggestions = fallback;
        showHeaderSuggestions = fallback.length > 0;
      }
    }, 200);
  }

  function submitHeaderSearch(event: SubmitEvent): void {
    event.preventDefault();
    showHeaderSuggestions = false;
    void goto(`/search?q=${encodeURIComponent(headerQuery)}`);
  }

  function selectHeaderSuggestion(item: string): void {
    headerQuery = item;
    showHeaderSuggestions = false;
    void goto(`/search?q=${encodeURIComponent(item)}`);
  }

  function closeHeaderSuggestions(): void {
    showHeaderSuggestions = false;
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
    setAcceptLanguage(locale.meta.tag);
  });

  $effect(() => {
    void locale.init().then(() => {
      localeReady = true;
    });
  });
  $effect(() => {
    void currency.init();
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
  <meta name="description" content={t('app.metaDescription')} />
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
<svelte:window onkeydown={onKeydown} onscroll={onScroll} />

<div class="shell">
  <div class="shell__sticky-spacer" style:block-size="{stickyHeight}px"></div>
  <div
    class="shell__sticky-wrap"
    class:shell__sticky-wrap--hidden={headerHidden}
    bind:clientHeight={stickyHeight}
  >
  <header class="shell__header">
    <button
      type="button"
      class="shell__menu-toggle"
      onclick={toggleMobileNav}
      aria-expanded={mobileNavOpen}
      aria-controls="shell-mobile-nav"
      aria-label={mobileNavOpen ? t('nav.shell.closeMenu') : t('nav.shell.openMenu')}
      title={tt('tooltip.mobileNav')}
    >
      <Icon name="menu" size="1.25rem" />
    </button>

    <a class="shell__lockup" href="/">
      <img class="shell__emblem" src="/favicon.svg" alt="" width="32" height="32" />
      <span class="shell__wordmark">{t('app.name')}</span>
    </a>

    <nav class="shell__nav" aria-label={t('nav.shell.mainAriaLabel')}>
      <a class="shell__nav-link" class:is-current={isCurrentRoute('/catalog')} aria-current={isCurrentRoute('/catalog') ? 'page' : undefined} href="/catalog">{t('nav.shell.craftDirectory')}</a>
      <a class="shell__nav-link" class:is-current={isCurrentRoute('/gi-tagged')} aria-current={isCurrentRoute('/gi-tagged') ? 'page' : undefined} href="/gi-tagged">{t('nav.shell.giHeritage')}</a>
      <a class="shell__nav-link" class:is-current={isCurrentRoute('/fairs')} aria-current={isCurrentRoute('/fairs') ? 'page' : undefined} href="/fairs">{t('nav.shell.exhibitions')}</a>
      <a class="shell__nav-link" class:is-current={isCurrentRoute('/company/register')} aria-current={isCurrentRoute('/company/register') ? 'page' : undefined} href="/company/register">{t('nav.shell.enterprise')}</a>
      <a class="shell__nav-link" class:is-current={isCurrentRoute('/case-studies')} aria-current={isCurrentRoute('/case-studies') ? 'page' : undefined} href="/case-studies">{t('nav.shell.impactStudies')}</a>
    </nav>

    <div class="shell__actions">
      <div class="shell__search-wrap">
        <form class="shell__search-inline" action="/search" role="search" onsubmit={submitHeaderSearch}>
          <Icon name="search" size="1rem" class="shell__search-icon" />
          <input
            type="search"
            name="q"
            placeholder={t('search.placeholder')}
            aria-label={t('nav.search')}
            bind:value={headerQuery}
            oninput={onHeaderSearchInput}
            onfocus={() => { if (headerSuggestions.length > 0) showHeaderSuggestions = true; }}
            onblur={closeHeaderSuggestions}
            role="combobox"
            aria-expanded={showHeaderSuggestions}
            aria-controls="shell-search-suggestions"
            autocomplete="off"
          />
        </form>
        {#if showHeaderSuggestions && headerSuggestions.length > 0}
          <ul class="shell__search-suggestions" id="shell-search-suggestions" role="listbox">
            {#each headerSuggestions as item (item)}
              <li>
                <button type="button" class="shell__search-suggestion" onmousedown={() => selectHeaderSuggestion(item)}>
                  <Icon name="search" size="0.85rem" />
                  <span>{item}</span>
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </div>
      <a class="shell__search-toggle" href="/search" aria-label={t('nav.search')} title={tt('tooltip.search')}>
        <Icon name="search" size="1.1rem" />
      </a>
      <!-- AccountMenu last: its popover anchors flush to *its own* right
           edge (inset-inline-end: 0 relative to the trigger), so it only
           avoids running off the left edge of a phone screen if nothing
           sits to its right pushing it away from the header's true right
           edge. -->
      <LanguageSelector triggerSize="lg" />
      <AccessibilityControl statementHref="/accessibility" triggerSize="lg" />
      <AccountMenu />
    </div>
  </header>

  {#if mobileNavOpen}
    <nav class="shell__mobile-nav" id="shell-mobile-nav" aria-label={t('nav.shell.mainMobileAriaLabel')}>
      <a class="shell__mobile-link" class:is-current={isCurrentRoute('/catalog')} aria-current={isCurrentRoute('/catalog') ? 'page' : undefined} href="/catalog" onclick={closeMobileNav}>{t('nav.shell.craftDirectory')}</a>
      <a class="shell__mobile-link" class:is-current={isCurrentRoute('/gi-tagged')} aria-current={isCurrentRoute('/gi-tagged') ? 'page' : undefined} href="/gi-tagged" onclick={closeMobileNav}>{t('nav.shell.giHeritage')}</a>
      <a class="shell__mobile-link" class:is-current={isCurrentRoute('/case-studies')} aria-current={isCurrentRoute('/case-studies') ? 'page' : undefined} href="/case-studies" onclick={closeMobileNav}>{t('nav.shell.impactStudies')}</a>
    </nav>
  {/if}

  <CategorySubnav />
  </div>

  <main class="shell__main" id="main-content" tabindex="-1">
    {#if localeReady}
      <ErrorBoundary
        source="buyer-shell"
        dsn={env.PUBLIC_SENTRY_DSN}
        title={t('error.boundary.title')}
        body={t('error.boundary.body')}
        retryLabel={t('error.boundary.retry')}
      >
        {@render children()}
      </ErrorBoundary>
    {:else}
      <p class="shell__boot" role="status" aria-live="polite">{t('state.loading')}</p>
    {/if}
  </main>

  <BuyerFooter />
</div>

<style>
  .shell__sticky-wrap {
    position: fixed;
    inset-block-start: 0;
    inset-inline: 0;
    z-index: 90;
    transition: transform 0.25s ease;
  }

  .shell__sticky-wrap--hidden {
    transform: translateY(-100%);
  }

  .shell__boot {
    padding-block: var(--k-space-6);
    text-align: center;
    color: var(--k-text-secondary);
  }

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

    .shell__mobile-link.is-current {
      color: var(--k-terracotta);
      background-color: var(--k-surface-sunken);
      box-shadow: inset 2px 0 0 var(--k-terracotta);
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

  .shell__nav-link.is-current {
    color: var(--k-terracotta-700, var(--k-accent-primary-text));
    background-color: rgba(244, 240, 234, 0.9);
    box-shadow: inset 0 -2px 0 var(--k-terracotta-700, var(--k-accent-primary-text));
  }
</style>
