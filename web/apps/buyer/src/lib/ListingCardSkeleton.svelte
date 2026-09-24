<!--
  apps/buyer/src/lib/ListingCardSkeleton.svelte

  Loading placeholder shaped like ListingCard.svelte (4/5 photo, craft line,
  title line, artisan avatar + name row, price row) instead of one blank
  rectangle the size of the whole card -- see the grids in search/craft/feed
  that render one of these per expected result while loading.
-->
<script lang="ts">
  import { Skeleton } from '@kalakriti/ui';
</script>

<div class="listing-card-skeleton" aria-hidden="true">
  <div class="listing-card-skeleton__media">
    <Skeleton width="100%" height="100%" radius="var(--k-radius-md)" />
  </div>
  <div class="listing-card-skeleton__body">
    <Skeleton shape="text" width="35%" height="0.75rem" />
    <Skeleton shape="text" width="80%" height="1rem" />
    <div class="listing-card-skeleton__artisan">
      <Skeleton width="1.25rem" height="1.25rem" radius="50%" />
      <Skeleton shape="text" width="45%" height="0.8rem" />
    </div>
    <Skeleton shape="text" width="30%" height="1rem" />
  </div>
</div>

<style>
  .listing-card-skeleton {
    display: flex;
    flex-direction: column;
  }

  .listing-card-skeleton__media {
    position: relative;
    aspect-ratio: 4 / 5;
  }

  .listing-card-skeleton__body {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-2);
    padding-block-start: var(--k-space-2);
  }

  .listing-card-skeleton__artisan {
    display: flex;
    align-items: center;
    gap: var(--k-space-2);
  }
</style>
