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
  import { NAV_ITEMS } from '$lib/nav';
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
  const visibleItems = $derived(NAV_ITEMS.filter((item) => !item.role || item.role === role));
  const currentPath = $derived($page.url.pathname);

  let paletteOpen = $state(false);

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
</script>

<svelte:head>
  <title>{t('admin.home.title')}</title>
</svelte:head>

<SkipLink target="main-content" />
<RouteAnnouncer />

<div class="shell">
  <header class="shell__header">
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

  <div class="shell__body">
    <nav class="shell__sidebar" aria-label={t('nav.dashboard')}>
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
              <Icon name={item.icon} />
              {t(item.labelKey)}
            </a>
          </li>
        {/each}
      </ul>
    </nav>

    <main class="shell__main" id="main-content" tabindex="-1">
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
    </main>
  </div>

  <footer class="shell__footer">
    <p>{t('app.ministry')}</p>
  </footer>
</div>

<CommandPalette bind:open={paletteOpen} />

<style>
  .shell__actions {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    margin-inline-start: auto;
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
    grid-template-columns: 15rem 1fr;
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
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    padding: var(--k-space-2) var(--k-space-3);
    border-radius: var(--k-radius-sm);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
    font-weight: var(--k-weight-medium);
  }

  .shell__nav-link[aria-current='page'] {
    background: var(--k-surface-sunken);
    color: var(--k-text-primary);
  }

  .shell__main {
    padding: var(--k-space-5) var(--k-space-6) var(--k-space-9);
    min-inline-size: 0;
  }

  @media (max-width: 900px) {
    .shell__body {
      grid-template-columns: 1fr;
    }

    .shell__sidebar {
      position: static;
      border-inline-end: none;
      border-block-end: var(--k-hairline) solid var(--k-border-hairline);
    }

    .shell__nav-list {
      flex-direction: row;
      flex-wrap: wrap;
    }
  }
</style>
