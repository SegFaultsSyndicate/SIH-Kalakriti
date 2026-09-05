<!--
  packages/ui/src/VisuallyHidden.svelte

    <VisuallyHidden>Opens in a new tab</VisuallyHidden>
    <VisuallyHidden as="span">...</VisuallyHidden>

  Content for assistive tech only: present in the accessibility tree, clipped
  from the visual canvas without display:none (which would hide it from that
  tree too). The standard clip-rect technique, not opacity/width tricks that
  a screen magnifier can still show as a stray dot.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    /** Element to render. 'span' by default so it never breaks inline flow. */
    as?: 'span' | 'div';
    children: Snippet;
  }

  let { as = 'span', children }: Props = $props();
</script>

<svelte:element this={as} class="k-visually-hidden">
  {@render children()}
</svelte:element>

<style>
  .k-visually-hidden {
    position: absolute;
    inline-size: 1px;
    block-size: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    clip-path: inset(50%);
    white-space: nowrap;
    margin-inline-end: -1px;
  }
</style>
