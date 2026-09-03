<!--
  Kalakriti <Illustration /> — Batch 5

    <Illustration name="empty-no-listings" />
    <Illustration name="onboard-paid-directly" title="A thread runs directly from buyer to artisan" />

  Every illustration is decorative by default; pass `title` only when the
  illustration is standing in for real content (rare — usually the
  surrounding UI already carries the text an empty state or onboarding
  slide needs).

  Pulls in the full ~30-illustration set, same trade-off as Icon.svelte —
  import a single `./src/<name>.svg` directly when only one illustration is
  used on a route.

  `size` sets width (height follows from the illustration's own aspect
  ratio via the viewBox, so only one dimension is needed). No `strokeWidth`
  prop — illustrations are fixed compositions, not a weight-variant system
  like the icon set.
-->
<script>
  import { ILLUSTRATIONS } from './manifest.js';

  const modules = import.meta.glob('./src/*.svg', { eager: true, import: 'default' });

  export let name;
  export let title = undefined;
  export let size = undefined; // any CSS length, e.g. "20rem" — sets width only
  let className = undefined;
  export { className as class };

  $: entry = ILLUSTRATIONS.find((i) => i.name === name);
  $: Component = entry ? modules['./' + entry.file] : undefined;
  $: titleId = title ? `k-ill-title-${name}` : undefined;

  if (!entry && typeof console !== 'undefined') {
    console.warn(`<Illustration name="${name}"> does not match any entry in packages/illustrations/manifest.js`);
  }
</script>

{#if Component}
  <svelte:component
    this={Component}
    class="k-illustration {className || ''}"
    role={title ? 'img' : undefined}
    aria-labelledby={title ? titleId : undefined}
    aria-hidden={title ? undefined : 'true'}
    focusable="false"
    {...$$restProps}
    style={size ? `width:${size}` : undefined}
  >
    {#if title}<title id={titleId}>{title}</title>{/if}
  </svelte:component>
{/if}
