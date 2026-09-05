<!--
  packages/ui/src/RouteAnnouncer.svelte

    Mounted once, in the root +layout.svelte, alongside SkipLink:
    <RouteAnnouncer />

  SvelteKit swaps the page in place on client-side navigation; a screen
  reader has no DOM-mutation event that tells it "this is now a different
  page" the way a full document load does. This mounts one aria-live region
  and, after each navigation, announces the new <svelte:head><title> (already
  set by every route -- reusing it means this component doesn't need to know
  the page data shape) and moves focus to the main heading.

  The first `afterNavigate` firing is the initial load, not a navigation: the
  browser already announced that page via the document title at load time,
  so announcing it again here would be a redundant, confusing echo before the
  user has done anything.
-->
<script lang="ts">
  import { afterNavigate } from '$app/navigation';
  import { focusMainHeading } from './focus';

  let message = $state('');
  let seenFirst = false;

  afterNavigate(() => {
    if (!seenFirst) {
      seenFirst = true;
      return;
    }
    // Re-set even if the title string repeats, so an identical title between
    // two routes still gets announced -- clear first, then set on the next
    // tick, or a screen reader may treat the unchanged text as no change.
    message = '';
    queueMicrotask(() => {
      message = typeof document !== 'undefined' ? document.title : '';
      focusMainHeading();
    });
  });
</script>

<div class="k-route-announcer" role="status" aria-live="polite" aria-atomic="true">
  {message}
</div>

<style>
  .k-route-announcer {
    position: absolute;
    inline-size: 1px;
    block-size: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    clip-path: inset(50%);
    white-space: nowrap;
  }
</style>
