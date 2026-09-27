<!--
  apps/admin/src/routes/+layout.svelte

  Ministry dashboard shell: persistent sidebar, breadcrumb, Cmd/Ctrl-K
  command palette. Desktop-first and dense -- same tokens as the consumer
  apps, tighter spacing and no shadow-card grid. Batch 8 built the plain
  header-only shell; Batch 13 adds the sidebar, breadcrumb, palette and a
  real login (reusing the same OTP endpoints artisan/buyer already use --
  there is no separate admin auth backend).
-->
<script lang="ts">
  import '../app.css';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { locale } from '@kalakriti/i18n';
  import { SkipLink, RouteAnnouncer, AccessibilityControl, a11y } from '@kalakriti/ui';
  import { Icon } from '@kalakriti/icons';
  import {
    session,
    restoreAccessToken,
    setUnauthorizedHandler,
    setSessionRefreshHandler,
    createLoginRedirectHandler,
  } from '@kalakriti/api';
  import { NAV_ITEMS, visibleTo } from '$lib/nav';
  import CommandPalette from '$lib/CommandPalette.svelte';
  import Breadcrumb from '$lib/Breadcrumb.svelte';
  import { ErrorBoundary } from '@kalakriti/observability';
  import { env } from '$env/dynamic/public';

  interface Props {
    children: import('svelte').Snippet;
  }

  let { children }: Props = $props();

  const t = $derived(locale.t);
  const role = $derived(session.claims?.['role'] as string | undefined);
  const visibleItems = $derived(NAV_ITEMS.filter((item) => visibleTo(item, role)));
  const currentPath = $derived($page.url.pathname);

  let paletteOpen = $state(false);
  let sidebarOpen = $state(true);

  // Gates {@render children()} below: locale.init() is async, so without
  // this a page's first paint runs with an empty catalogue -- every t()
  // call falls through to English -- then re-renders in the real language a
  // tick later. See I18N_PLAN.md's F-1.
  let localeReady = $state(false);
  let sessionReady = $state(false);
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

  $effect(() => {
    void locale.init().then(() => {
      localeReady = true;
    });
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
      sessionReady = true;
    })();
  });
</script>

<svelte:head>
  <title>{t('admin.home.title')}</title>
</svelte:head>

<SkipLink target="main-content" />
<RouteAnnouncer />
<svelte:window onscroll={onScroll} />

<div class="shell">
  {#if !sidebarOpen}
    <div
      class="shell__sidebar-reveal-zone"
      aria-hidden="true"
      onpointerenter={(event) => {
        if (event.pointerType === 'mouse') sidebarOpen = true;
      }}
    ></div>
  {/if}
  <div class="shell__sticky-spacer" style:block-size="{stickyHeight}px"></div>
  <div
    class="shell__sticky-wrap"
    class:shell__sticky-wrap--hidden={headerHidden}
    bind:clientHeight={stickyHeight}
  >
  <header class="shell__header">
    {#if !sidebarOpen}
      <button
        type="button"
        class="shell__sidebar-reopen"
        aria-label={t('nav.shell.openMenu')}
        title={t('nav.shell.openMenu')}
        aria-controls="admin-dashboard-nav"
        aria-expanded={sidebarOpen}
        onclick={() => (sidebarOpen = true)}
      >
        <Icon name="menu" />
      </button>
    {/if}
    <a class="shell__lockup" href="/">
      <img class="shell__emblem" src="/favicon.svg" alt="" width="28" height="28" />
      <span class="shell__wordmark">
        {t('admin.home.title')}
        <span class="shell__ministry">{t('app.ministry')}</span>
      </span>
    </a>

    <div class="shell__actions">
      <button type="button" class="shell__palette-trigger" onclick={() => (paletteOpen = true)}>
        <Icon name="search" />
        <span>{t('palette.trigger')}</span>
        <kbd>Ctrl K</kbd>
      </button>
      {#if session.status === 'authenticated'}
        <span class="shell__role">{role ?? ''}</span>
      {:else}
        <a class="shell__login-link" href="/login">{t('login.heading')}</a>
      {/if}
      <AccessibilityControl statementHref="/accessibility" />
    </div>
  </header>
  </div>

  <div class="shell__body" class:shell__body--sidebar-hidden={!sidebarOpen}>
    <aside class="shell__sidebar" hidden={!sidebarOpen}>
      <button
        type="button"
        class="shell__sidebar-toggle"
        aria-label={t('nav.shell.closeMenu')}
        title={t('nav.shell.closeMenu')}
        aria-controls="admin-dashboard-nav"
        aria-expanded={sidebarOpen}
        onclick={() => (sidebarOpen = false)}
      >
        <Icon name="close" size="0.8rem" />
      </button>
      <nav id="admin-dashboard-nav" aria-label={t('nav.dashboard')}>
        <ul class="shell__nav-list">
          {#each visibleItems as item (item.href)}
            <li>
              <a
                href={item.href}
                class="shell__nav-link"
                aria-current={currentPath === item.href || currentPath.startsWith(item.href + '/')
                  ? 'page'
                  : undefined}
              >
                <Icon name={item.icon} size={item.iconSize} />
                {t(item.labelKey)}
              </a>
            </li>
          {/each}
        </ul>
      </nav>
    </aside>

    <main class="shell__main" id="main-content" tabindex="-1">
      {#if localeReady && sessionReady}
        <Breadcrumb />
        {#key currentPath}
          <ErrorBoundary
            source="admin-shell"
            dsn={env.PUBLIC_SENTRY_DSN}
            title={t('error.boundary.title')}
            body={t('error.boundary.body')}
            retryLabel={t('error.boundary.retry')}
          >
            {@render children()}
          </ErrorBoundary>
        {/key}
      {:else}
        <p class="shell__boot" role="status" aria-live="polite">{t('state.loading')}</p>
      {/if}
    </main>
  </div>

  <footer class="shell__footer">
    <p>{t('app.ministry')}</p>
  </footer>
</div>

<CommandPalette bind:open={paletteOpen} />

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

  .shell__actions {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    margin-inline-start: auto;
  }

  .shell__sidebar-toggle,
  .shell__sidebar-reopen {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    inline-size: 2rem;
    block-size: 2rem;
    padding: 0;
    border: 0;
    border-radius: var(--k-radius-sm);
    background: transparent;
    color: var(--k-text-secondary);
    cursor: pointer;
  }

  .shell__sidebar-reveal-zone {
    position: fixed;
    inset-block: 0;
    inset-inline-start: 0;
    z-index: 80;
    inline-size: 0.75rem;
  }

  .shell__sidebar-toggle:hover,
  .shell__sidebar-reopen:hover {
    background: var(--k-surface-sunken);
    color: var(--k-text-primary);
  }

  .shell__sidebar-toggle {
    position: absolute;
    inset-inline-end: var(--k-space-3);
    inset-block-start: var(--k-space-2);
    inline-size: 1.5rem;
    block-size: 1.5rem;
  }

  .shell__body.shell__body--sidebar-hidden {
    grid-template-columns: minmax(0, 1fr);
  }

  .shell__palette-trigger {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-1) var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    background: var(--k-surface-sunken);
    color: var(--k-text-secondary);
    font: inherit;
    font-size: var(--k-text-sm);
    cursor: pointer;
  }

  .shell__palette-trigger kbd {
    font-family: inherit;
    font-size: var(--k-text-2xs);
    padding: 0.1rem 0.35rem;
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-xs, 0.2rem);
    color: var(--k-text-tertiary, var(--k-text-secondary));
  }

  .shell__role {
    font-size: var(--k-text-2xs);
    letter-spacing: var(--k-tracking-wide);
    text-transform: uppercase;
    color: var(--k-text-secondary);
  }

  .shell__login-link {
    font-size: var(--k-text-sm);
    color: var(--k-text-primary);
  }

  .shell__body {
    display: grid;
    grid-template-columns: minmax(0, 15rem) minmax(0, 1fr);
    align-items: start;
  }

  .shell__sidebar {
    position: sticky;
    top: 0;
    padding: var(--k-space-5) var(--k-space-3);
    border-inline-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  .shell__nav-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
  }

  .shell__nav-link {
    display: grid;
    grid-template-columns: 1.5rem 1fr;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-3);
    border-radius: var(--k-radius-sm);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-medium);
  }

  /* Fixed icon column so labels start at the same x regardless of an
     item's icon intrinsic size (e.g. Badges' 1.75rem override). */
  .shell__nav-link :global(.k-icon) {
    justify-self: center;
  }

  .shell__nav-link[aria-current='page'] {
    background: var(--k-surface-sunken);
    color: var(--k-text-primary);
  }

  .shell__main {
    padding: var(--k-space-5) var(--k-space-6) var(--k-space-9);
    min-inline-size: 0;
  }

  .shell__boot {
    padding-block: var(--k-space-6);
    text-align: center;
    color: var(--k-text-secondary);
  }

  @media (max-width: 900px) {
    .shell__header {
      flex-wrap: wrap;
      row-gap: var(--k-space-2);
    }

    .shell__body {
      grid-template-columns: minmax(0, 1fr);
    }

    .shell__sidebar {
      position: relative;
      border-inline-end: none;
      border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    }

    .shell__nav-list {
      flex-direction: row;
      flex-wrap: wrap;
    }
  }
</style>
