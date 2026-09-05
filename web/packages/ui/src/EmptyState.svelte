<!--
  packages/ui/src/EmptyState.svelte

    <EmptyState
      illustration="empty-no-listings"
      heading={t('listings.empty.heading')}
      body={t('listings.empty.body')}
    >
      {#snippet action()}
        <Button onclick={startListing}>{t('listings.new')}</Button>
      {/snippet}
    </EmptyState>

  Never a bare "No data": a scene from the asset pack, a heading, a line of
  body copy, and (usually) a way out. `action` is a snippet rather than a
  label+onclick pair so the call site can reach for exactly the button
  variant the moment calls for -- primary to start something, ghost to
  clear a filter -- without EmptyState guessing.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';
  import { Illustration } from '@kalakriti/illustrations';

  interface Props {
    illustration: string;
    heading: string;
    body?: string;
    class?: string;
    action?: Snippet;
  }

  let { illustration, heading, body, class: className, action }: Props = $props();
</script>

<div class="k-empty-state {className || ''}">
  <Illustration name={illustration} size="14rem" class="k-empty-state__illustration" />
  <p class="k-empty-state__heading">{heading}</p>
  {#if body}
    <p class="k-empty-state__body">{body}</p>
  {/if}
  {#if action}
    <div class="k-empty-state__action">
      {@render action()}
    </div>
  {/if}
</div>
