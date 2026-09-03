<!--
  Kalakriti <Icon /> — Batch 4

    <Icon name="search" />
    <Icon name="warning" title="Something needs attention" />
    <Icon name="trash" class="k-icon--bold" />

  Importing this component (or icons.js, which it uses) pulls in the whole
  icon set — the name lookup needs every component available to dispatch on.
  When only one or two icons are used on a route, import the .svg directly
  instead (`import Search from '@kalakriti/icons/src/search.svg'`) to keep
  that page's bundle to just those icons; see the package README.

  Decorative by default (aria-hidden, focusable="false"). Passing `title`
  switches to role="img" with the title wired through aria-labelledby, per
  Batch 0's accessibility rule.

  `size` and `strokeWidth` are shorthand for the same icons.css custom
  properties the `k-icon--dense`/`k-icon--bold` classes set — use whichever
  reads better at the call site:

    <Icon name="trash" size="1.5rem" strokeWidth={2} />
-->
<script>
  import { ICON_COMPONENTS } from './icons.js';

  /** @type {import('./icons.d.ts').IconName} */
  export let name;
  export let title = undefined;
  export let size = undefined;       // any CSS length, e.g. "1.5rem" or "24px"
  export let strokeWidth = undefined; // 1.25 (dense) / 1.5 (default) / 2 (bold)
  let className = undefined;
  export { className as class };

  $: titleId = title ? `k-icon-title-${name}-${Math.random().toString(36).slice(2, 8)}` : undefined;
  $: Component = ICON_COMPONENTS[name];
  $: sizeStyle = [
    size ? `width:${size};height:${size}` : '',
    strokeWidth ? `--k-icon-stroke:${strokeWidth}` : '',
  ].filter(Boolean).join(';');

  if (!Component && typeof console !== 'undefined') {
    console.warn(`<Icon name="${name}"> does not match any icon in packages/icons/manifest.js`);
  }
</script>

{#if Component}
  <svelte:component
    this={Component}
    class="k-icon {className || ''}"
    role={title ? 'img' : undefined}
    aria-labelledby={title ? titleId : undefined}
    aria-hidden={title ? undefined : 'true'}
    focusable="false"
    {...$$restProps}
    style={sizeStyle || undefined}
  >
    {#if title}<title id={titleId}>{title}</title>{/if}
  </svelte:component>
{/if}
