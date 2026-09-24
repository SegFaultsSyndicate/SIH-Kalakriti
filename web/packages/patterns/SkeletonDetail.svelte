<!--
  Kalakriti <SkeletonDetail /> — single-record page loading placeholder.

  Mirrors a detail/hero page's actual shape (a media block, then a heading,
  a subtitle, a few body lines, and an action row) rather than one blank
  rectangle the size of the whole viewport.

    <SkeletonDetail />
    <SkeletonDetail mediaHeight="24rem" lines={4} />
    <SkeletonDetail media={false} lines={2} actions={false} />
-->
<script>
  import Skeleton from './Skeleton.svelte';

  /**
   * @typedef {object} Props
   * @property {boolean} [media]
   * @property {string} [mediaHeight]
   * @property {'square'|'circle'} [mediaShape]
   * @property {number} [lines]
   * @property {boolean} [actions]
   */

  /** @type {Props & Record<string, unknown>} */
  let {
    media = true,
    mediaHeight = '20rem',
    mediaShape = 'square',
    lines = 3,
    actions = true,
    class: className = '',
    ...rest
  } = $props();

  const lineArray = $derived(Array.from({ length: lines }));
</script>

<div class="k-skeleton-detail {className}" aria-hidden="true" {...rest}>
  {#if media}
    <Skeleton
      width={mediaShape === 'circle' ? mediaHeight : '100%'}
      height={mediaHeight}
      radius={mediaShape === 'circle' ? '50%' : 'var(--k-radius-lg)'}
    />
  {/if}
  <div class="k-skeleton-detail__body">
    <Skeleton shape="text" width="70%" height="1.5rem" />
    <Skeleton shape="text" width="40%" height="1rem" />
    {#each lineArray as _, i (i)}
      <Skeleton shape="text" width={i === lineArray.length - 1 ? '55%' : '100%'} height="0.9rem" />
    {/each}
  </div>
  {#if actions}
    <div class="k-skeleton-detail__actions">
      <Skeleton width="8rem" height="2.5rem" radius="var(--k-radius-md)" />
      <Skeleton width="8rem" height="2.5rem" radius="var(--k-radius-md)" />
    </div>
  {/if}
</div>

<style>
  .k-skeleton-detail {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-4);
  }

  .k-skeleton-detail__body {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
  }

  .k-skeleton-detail__actions {
    display: flex;
    gap: var(--k-space-3);
  }
</style>
