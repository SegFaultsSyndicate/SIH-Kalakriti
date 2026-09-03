<!--
  Kalakriti <Moment /> — Batch 7

    <Moment kind="success" />
    <Moment kind="failure" />

  A brief inline confirmation for a completed action (form submit, item
  saved). "success" reuses sync-synced.svg (the completed-stitch mark) —
  same visual, two names, because a header sync dot and an inline success
  moment are the same idea at two sizes, and drawing two near-identical
  stitch marks would just be two files to keep in sync with each other.
-->
<script>
  import Success from './src/sync-synced.svg';
  import Failure from './src/failure-thread.svg';

  /** @type {'success'|'failure'} */
  export let kind = 'success';
  export let title = undefined;

  $: Component = kind === 'success' ? Success : Failure;
</script>

<svelte:component
  this={Component}
  class="k-moment k-moment--{kind}"
  role={title ? 'img' : undefined}
  aria-label={title}
  aria-hidden={title ? undefined : 'true'}
  focusable="false"
  {...$$restProps}
/>
