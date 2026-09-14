<!--
  apps/artisan/src/routes/+layout.svelte

  The artisan shell, now carrying Batch 7's onboarding gate and the app's
  first real session/sync wiring:

    - restoreAccessToken() + session.establish() so a reload of an already
      logged-in artisan does not bounce them back to /login.
    - setUnauthorizedHandler(createLoginRedirectHandler(goto)), so a 401 from
      any future API call sends the artisan to /login rather than throwing.
    - the route guard (see $lib/route-guard.ts): language -> welcome/login/
      verify -> register/* -> home, enforced on every navigation.
    - syncEngine, started once here so the home screen's sync indicator and
      $lib/outbox-send's SendFn are live from the first paint.

  Header chrome (the lockup, connectivity pill, accessibility control) and
  the bottom nav are both conditional -- see showChrome and showBottomNav
  below -- because /language must show no chrome at all, and the bottom nav
  must not appear on any onboarding screen.
-->
<script lang="ts">
  import '../app.css';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { locale, hasExplicitLocale } from '@kalakriti/i18n';
  import { network } from '@kalakriti/offline';
  import {
    session,
    restoreAccessToken,
    setUnauthorizedHandler,
    createLoginRedirectHandler,
    setSessionRefreshHandler,
    setAcceptLanguage,
  } from '@kalakriti/api';
  import {
    SkipLink,
    ToastRegion,
    RouteAnnouncer,
    AccessibilityControl,
    LanguageSelector,
    a11y,
  } from '@kalakriti/ui';
  import { ReadScreen } from '@kalakriti/voice';
  import { ErrorBoundary } from '@kalakriti/observability';
  import { env } from '$env/dynamic/public';
  import PwaUpdatePrompt from '$lib/PwaUpdatePrompt.svelte';
  import InstallPrompt from '$lib/InstallPrompt.svelte';
  import BottomNav from '$lib/BottomNav.svelte';
  import { getArtisanId, watchArtisanId } from '$lib/registration';
  import { resolveRedirect, type GuardState } from '$lib/route-guard';
  import { syncEngine } from '$lib/sync';

  interface Props {
    children: import('svelte').Snippet;
  }

  let { children }: Props = $props();

  const t = $derived(locale.t);

  let booted = $state(false);
  let registeredArtisanId = $state<string | undefined>(undefined);

  // Genuine side effects: resolve the startup language, restore the session,
  // subscribe to connectivity/registration/sync. None of these compute a
  // value the template reads directly.
  $effect(() => {
    void locale.init();
  });

  $effect(() => {
    setAcceptLanguage(locale.meta.tag);
  });

  $effect(() => network.start());
  $effect(() => {
    void a11y.init();
  });
  $effect(() => a11y.start());
  $effect(() => syncEngine.start());

  $effect(() => {
    setUnauthorizedHandler(createLoginRedirectHandler((url) => goto(url)));
    setSessionRefreshHandler((token) => session.establish(token));
    return () => {
      setUnauthorizedHandler(undefined);
      setSessionRefreshHandler(undefined);
    };
  });

  $effect(() => {
    let disposeWatch: (() => void) | undefined;
    void (async () => {
      const [token, initialId] = await Promise.all([restoreAccessToken(), getArtisanId()]);
      if (token) session.establish(token);
      registeredArtisanId = initialId;
      booted = true;
      disposeWatch = watchArtisanId((id) => {
        registeredArtisanId = id;
      });
    })();
    return () => disposeWatch?.();
  });

  $effect(() => {
    if (!booted) return;
    const state: GuardState = {
      hasExplicitLocale: hasExplicitLocale(),
      authenticated: session.status === 'authenticated',
      registered: registeredArtisanId !== undefined,
    };
    const redirect = resolveRedirect(state, page.url.pathname);
    if (redirect !== null && redirect !== page.url.pathname) {
      void goto(redirect);
    }
  });

  const showChrome = $derived(booted && page.url.pathname !== '/language');
  const showBottomNav = $derived(
    showChrome && session.status === 'authenticated' && registeredArtisanId !== undefined,
  );
</script>

<svelte:head>
  <title>{t('app.name')}</title>
</svelte:head>

<SkipLink target="main-content" />
<RouteAnnouncer />

<div class="shell">
  {#if showChrome}
    <header class="shell__header">
      <a class="shell__lockup" href="/">
        <!--
          The emblem is a placeholder mark, not the State Emblem of India: that
          is legally protected and cannot ship in a demo build. Swapped for the
          real lockup only in a ministry-hosted deployment.
        -->
        <img class="shell__emblem" src="/favicon.svg" alt="" width="28" height="28" />
        <span class="shell__wordmark">{t('app.name')}</span>
      </a>

      <p
        class="shell__status net"
        class:net--online={network.online}
        class:net--offline={!network.online}
      >
        <span class="net__mark" aria-hidden="true"></span>
        <span>{network.online ? t('network.online') : t('network.offline')}</span>
      </p>

      <LanguageSelector />
      
      <AccessibilityControl statementHref="/accessibility" />
    </header>
  {/if}

  <main class="shell__main" id="main-content" tabindex="-1">
    {#if booted}
      <ErrorBoundary
        source="artisan-shell"
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

  {#if showBottomNav}
    <BottomNav />
  {/if}
</div>

<PwaUpdatePrompt />
<InstallPrompt />
<ToastRegion />
{#if showChrome}
  <div class:has-bottom-nav={showBottomNav}>
    <ReadScreen />
  </div>
{/if}

<style>
  .shell {
    display: flex;
    flex-direction: column;
    min-block-size: 100dvh;
  }

  .shell__header {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    padding: var(--k-space-3) var(--k-space-4);
    border-block-end: var(--k-hairline) solid var(--k-border-hairline);
  }

  .shell__lockup {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
    color: var(--k-text-primary);
    text-decoration: none;
  }

  .shell__wordmark {
    font-weight: var(--k-weight-semibold);
  }

  .shell__main {
    flex: 1;
    max-inline-size: var(--k-container-artisan);
    margin-inline: auto;
    inline-size: 100%;
    padding-inline: var(--k-gutter);
  }

  .shell__boot {
    padding-block: var(--k-space-6);
    text-align: center;
    color: var(--k-text-secondary);
  }

  :global(.has-bottom-nav .k-read-screen) {
    inset-block-end: calc(var(--k-space-4) + 4.5rem + env(safe-area-inset-bottom, 0px));
  }

  :global(.has-bottom-nav .k-read-screen__unavailable) {
    inset-block-end: calc(var(--k-space-4) + 4.5rem + var(--k-touch-min) + var(--k-space-2) + env(safe-area-inset-bottom, 0px));
  }

  @media (max-width: 32rem) {
    .shell__header {
      padding: var(--k-space-2) var(--k-space-3);
      gap: var(--k-space-2);
    }

    .shell__status span:not(.net__mark) {
      display: none;
    }

    .shell__main {
      padding-inline: var(--k-space-3);
    }
  }
</style>
