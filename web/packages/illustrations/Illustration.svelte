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

  /**
   * @typedef {object} Props
   * @property {string} name
   * @property {string} [title] Accessible name. Omit for a decorative scene,
   *   which most empty-state art is -- the surrounding copy carries the
   *   meaning and a duplicated label is noise for a screen-reader user.
   * @property {string} [size] Any CSS length, e.g. "20rem". Sets width only.
   * @property {string} [class]
   */

  /** @type {Props & Record<string, unknown>} */
  let { name, title, size, class: className, ...rest } = $props();

  const entry = $derived(ILLUSTRATIONS.find((i) => i.name === name));
  const Component = $derived(entry ? modules['./' + entry.file] : undefined);

  $effect(() => {
    if (!entry) {
      console.warn(
        `<Illustration name="${name}"> does not match any entry in packages/illustrations/manifest.js`,
      );
    }
  });
</script>

<!--
  aria-label, not a <title> child: the SVG components are generated as
  `<svg {...props}>{@html contents}</svg>` with no slot, so a child <title>
  is dropped and an aria-labelledby pointing at it dangles. See Icon.svelte.
-->
{#if Component}
  <Component
    class="k-illustration {className || ''}"
    role={title ? 'img' : undefined}
    aria-label={title}
    aria-hidden={title ? undefined : 'true'}
    focusable="false"
    {...rest}
    style={size ? `width:${size}` : undefined}
  />
{/if}
