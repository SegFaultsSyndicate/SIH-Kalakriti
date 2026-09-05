<!--
  apps/artisan/src/lib/InstallPrompt.svelte

  The `beforeinstallprompt` event fires whenever Chrome decides the page is
  installable -- often on first load, long before the artisan has any reason
  to trust the app enough to want it on their home screen. This component
  always captures the event (so `prompt()` stays available to call later),
  but never SHOWS the banner until hasPublishedOnce() says there is a real
  listing out there -- "add it after they've published once", not a timer.

  iOS Safari never fires beforeinstallprompt at all (no browser API for it);
  this component simply never appears there, which is correct -- iOS's own
  install path is Share -> Add to Home Screen, documented for the artisan in
  web/DEMO.md rather than replicated here as a fake button.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { locale } from '@kalakriti/i18n';
  import { hasPublishedOnce } from './listings';

  const t = $derived(locale.t);

  let deferredPrompt: Event & { prompt(): Promise<void> } = $state(
    null as unknown as Event & { prompt(): Promise<void> },
  );
  let visible = $state(false);
  let dismissed = $state(false);

  onMount(() => {
    const onBeforeInstall = (event: Event) => {
      event.preventDefault();
      deferredPrompt = event as typeof deferredPrompt;
      void checkEligible();
    };
    window.addEventListener('beforeinstallprompt', onBeforeInstall);
    void checkEligible();
    return () => window.removeEventListener('beforeinstallprompt', onBeforeInstall);
  });

  async function checkEligible() {
    if (!deferredPrompt || dismissed) return;
    visible = await hasPublishedOnce();
  }

  async function install() {
    visible = false;
    await deferredPrompt?.prompt();
    deferredPrompt = null as unknown as typeof deferredPrompt;
  }

  function dismiss() {
    visible = false;
    dismissed = true;
  }
</script>

{#if visible}
  <div class="install" role="status" aria-live="polite">
    <p class="install__title">{t('pwa.install.title')}</p>
    <p class="install__body">{t('pwa.install.body')}</p>
    <div class="install__actions">
      <button type="button" class="install__button install__button--primary" onclick={install}>
        {t('pwa.install.accept')}
      </button>
      <button type="button" class="install__button" onclick={dismiss}>
        {t('pwa.install.dismiss')}
      </button>
    </div>
  </div>
{/if}

<style>
  .install {
    position: fixed;
    inset-block-end: var(--k-space-4);
    inset-inline: var(--k-space-4);
    z-index: var(--k-z-toast);
    padding: var(--k-space-4);
    background-color: var(--k-surface-raised);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-inline-start: var(--k-rule-heavy) solid var(--k-accent-primary-bg);
    border-radius: var(--k-radius-md);
    box-shadow: var(--k-elevation-menu);
  }

  .install__title {
    font-weight: var(--k-weight-semibold);
  }

  .install__body {
    margin-block-start: var(--k-space-1);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .install__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-3);
  }

  .install__button {
    min-block-size: var(--k-touch-min);
    padding-inline: var(--k-space-4);
    background-color: transparent;
    color: var(--k-text-primary);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    cursor: pointer;
  }

  .install__button--primary {
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    border-color: var(--k-accent-primary-bg);
    font-weight: var(--k-weight-medium);
  }
</style>
