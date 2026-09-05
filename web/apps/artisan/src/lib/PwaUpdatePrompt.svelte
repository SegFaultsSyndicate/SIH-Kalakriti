<!--
  apps/artisan/src/lib/PwaUpdatePrompt.svelte

  Service worker registration and the update prompt.

  registerType is 'prompt', so a new worker waits rather than taking over. This
  component is the only thing that can promote it, and it will not do so
  without a tap: the artisan may be halfway through photographing a piece, and
  an auto-reload would discard the capture state that has not reached IndexedDB
  yet.

  `virtual:pwa-register` (not the /svelte variant) because that one hands back
  Svelte 4 stores; this hands back plain callbacks that drop straight into
  $state.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { locale } from '@kalakriti/i18n';

  const t = $derived(locale.t);

  let needRefresh = $state(false);
  let updateServiceWorker: ((reload?: boolean) => Promise<void>) | null = $state(null);

  onMount(async () => {
    // Dynamic import so the registration code is not on the critical path and
    // so a build without the PWA plugin (unit tests) does not fail to resolve.
    // @ts-expect-error virtual:pwa-register is injected by vite-plugin-pwa at build time.
    const { registerSW } = await import('virtual:pwa-register');
    updateServiceWorker = registerSW({
      immediate: true,
      onNeedRefresh() {
        needRefresh = true;
      },
    });
  });

  async function update() {
    needRefresh = false;
    await updateServiceWorker?.(true);
  }
</script>

{#if needRefresh}
  <!--
    role="status" and not "alert": this is not urgent, and an alert would
    interrupt a screen reader mid-sentence during capture.
  -->
  <div class="update" role="status" aria-live="polite">
    <p class="update__title">{t('sw.update.title')}</p>
    <p class="update__body">{t('sw.update.body')}</p>
    <div class="update__actions">
      <button type="button" class="update__button update__button--primary" onclick={update}>
        {t('sw.update.accept')}
      </button>
      <button type="button" class="update__button" onclick={() => (needRefresh = false)}>
        {t('sw.update.dismiss')}
      </button>
    </div>
  </div>
{/if}

<style>
  /*
   * Genuinely floating, so a shadow is honest here -- this is one of the three
   * cases the design system allows it.
   */
  .update {
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

  .update__title {
    font-weight: var(--k-weight-semibold);
  }

  .update__body {
    margin-block-start: var(--k-space-1);
    color: var(--k-text-secondary);
    font-size: var(--k-text-sm);
  }

  .update__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-2);
    margin-block-start: var(--k-space-3);
  }

  .update__button {
    min-block-size: var(--k-touch-min);
    padding-inline: var(--k-space-4);
    background-color: transparent;
    color: var(--k-text-primary);
    border: var(--k-hairline) solid var(--k-border-interactive);
    border-radius: var(--k-radius-sm);
    cursor: pointer;
  }

  .update__button--primary {
    background-color: var(--k-accent-primary-bg);
    color: var(--k-text-on-accent);
    border-color: var(--k-accent-primary-bg);
    font-weight: var(--k-weight-medium);
  }
</style>
