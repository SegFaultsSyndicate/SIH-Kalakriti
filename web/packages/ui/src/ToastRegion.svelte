<!--
  packages/ui/src/ToastRegion.svelte

  Mount once, at the app root:

    <ToastRegion />
    ...
    import { showToast } from '@kalakriti/ui';
    showToast({ message: t('...'), variant: 'success' });

  Two live regions, not one: aria-live="polite" for info/success (waits for
  a pause in speech) and role="alert" (assertive) for error, per the brief.
  A single region set to the loudest level would make every success toast
  interrupt whatever the screen reader was already saying.

  Mounted once at every app's root layout -- meaning whatever it imports is
  in the CRITICAL path for every route, not lazy-loaded like a page. The
  three icons below are imported directly from their .svg files rather than
  through <Icon name>, which pulls in the full ~90-icon manifest (see
  packages/icons' own README on this trade-off): that difference alone was
  worth roughly 30KB gzip in the artisan app's initial payload the first
  time this file used <Icon>.

  A bare `.svg` import resolves to a real Svelte component at runtime (the
  plugin's own default), but vite/client's ambient `declare module '*.svg'`
  (a plain asset -> string URL) matches the same bare pattern and wins under
  `skipLibCheck`, so the compiler sees these as `string`. Recast rather than
  fight that with a second, more specific ambient declaration: this codebase
  has zero prior art for a typed direct `.svg` import to build on, and a
  four-line cast here is far less fragile than a wildcard ambient pattern
  whose interaction with `moduleResolution: "bundler"` and package `exports`
  maps was not turning out to resolve the way the plugin's own shipped
  svg.d.ts implies it should.
-->
<script lang="ts">
  import type { Component } from 'svelte';
  import type { SVGAttributes } from 'svelte/elements';
  import { locale } from '@kalakriti/i18n';
  import IconInfoRaw from '@kalakriti/icons/src/info.svg';
  import IconSuccessRaw from '@kalakriti/icons/src/success.svg';
  import IconWarningRaw from '@kalakriti/icons/src/warning.svg';
  import IconCloseRaw from '@kalakriti/icons/src/close.svg';
  import { toastQueue, type ToastVariant } from './toast.svelte';

  type SvgComponent = Component<SVGAttributes<SVGSVGElement>>;
  const asSvgComponent = (value: unknown) => value as SvgComponent;

  const IconInfo = asSvgComponent(IconInfoRaw);
  const IconSuccess = asSvgComponent(IconSuccessRaw);
  const IconWarning = asSvgComponent(IconWarningRaw);
  const IconClose = asSvgComponent(IconCloseRaw);

  const t = $derived(locale.t);

  const polite = $derived(toastQueue.items.filter((item) => item.variant !== 'error'));
  const assertive = $derived(toastQueue.items.filter((item) => item.variant === 'error'));

  const VARIANT_ICON: Record<ToastVariant, SvgComponent> = {
    info: IconInfo,
    success: IconSuccess,
    error: IconWarning,
  };
</script>

{#snippet region(items: typeof toastQueue.items)}
  {#each items as item (item.id)}
    {@const ToastIcon = VARIANT_ICON[item.variant]}
    <div
      class="k-toast k-toast--{item.variant}"
      role="group"
      onpointerenter={() => toastQueue.pause(item.id)}
      onpointerleave={() => toastQueue.resume(item.id)}
      onfocusin={() => toastQueue.pause(item.id)}
      onfocusout={() => toastQueue.resume(item.id)}
    >
      <ToastIcon class="k-icon k-toast__icon" aria-hidden="true" focusable="false" />
      <p class="k-toast__message">{item.message}</p>
      {#if item.action}
        <button type="button" class="k-toast__action" onclick={item.action.onclick}>
          {item.action.label}
        </button>
      {/if}
      <button
        type="button"
        class="k-toast__close"
        onclick={() => toastQueue.dismiss(item.id)}
        aria-label={t('ui.toast.dismiss')}
      >
        <IconClose class="k-icon" aria-hidden="true" focusable="false" />
      </button>
    </div>
  {/each}
{/snippet}

<div class="k-toast-region k-toast-region--polite" aria-live="polite">
  {@render region(polite)}
</div>
<div class="k-toast-region k-toast-region--assertive" role="alert">
  {@render region(assertive)}
</div>
