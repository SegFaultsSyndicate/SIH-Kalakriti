<!--
  Kalakriti <SkeletonRow /> — list-row loading placeholder.

  Mirrors the shape every "list of cards" page in this system actually
  renders (optional thumbnail/avatar + a stack of text lines + an optional
  trailing action), instead of one blank rectangle the size of the whole
  row. Used for listings/orders/notifications/earnings/crafts-style rows.

    <SkeletonRow />                                   2 lines, no thumbnail
    <SkeletonRow thumbnail />                          + 3.5rem square media
    <SkeletonRow thumbnail thumbnailShape="circle" />   avatar-style row
    <SkeletonRow lines={[{ width: '70%' }, { width: '40%' }, { width: '30%' }]} />
    <SkeletonRow trailing trailingWidth="6rem" />       + trailing button/chip
-->
<script>
  import Card from './Card.svelte';
  import Skeleton from './Skeleton.svelte';

  /**
   * @typedef {object} Props
   * @property {boolean} [thumbnail]
   * @property {'square'|'circle'} [thumbnailShape]
   * @property {string} [thumbnailSize]
   * @property {{ width?: string, height?: string }[]} [lines]
   * @property {boolean} [trailing]
   * @property {string} [trailingWidth]
   * @property {string} [trailingHeight]
   */

  /** @type {Props & Record<string, unknown>} */
  let {
    thumbnail = false,
    thumbnailShape = 'square',
    thumbnailSize = '3.5rem',
    lines = [{ width: '60%' }, { width: '40%' }],
    trailing = false,
    trailingWidth = '4rem',
    trailingHeight = '2rem',
    class: className = '',
    ...rest
  } = $props();
</script>

<Card variant="hairline" element="div" class="k-skeleton-row {className}" aria-hidden="true" {...rest}>
  {#if thumbnail}
    <Skeleton
      width={thumbnailSize}
      height={thumbnailSize}
      radius={thumbnailShape === 'circle' ? '50%' : 'var(--k-radius-md)'}
    />
  {/if}
  <div class="k-skeleton-row__body">
    {#each lines as line, i (i)}
      <Skeleton shape="text" width={line.width ?? '50%'} height={line.height ?? (i === 0 ? '1rem' : '0.8rem')} />
    {/each}
  </div>
  {#if trailing}
    <Skeleton width={trailingWidth} height={trailingHeight} radius="var(--k-radius-md)" />
  {/if}
</Card>

<style>
  :global(.k-skeleton-row) {
    display: flex;
    align-items: center;
    gap: var(--k-space-3);
    padding: var(--k-space-3);
  }

  .k-skeleton-row__body {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    min-inline-size: 0;
  }
</style>
