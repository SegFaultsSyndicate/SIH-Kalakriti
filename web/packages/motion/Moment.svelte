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

  /**
   * @typedef {object} Props
   * @property {'success'|'failure'} [kind]
   * @property {string} [title] Accessible name. A success or failure moment
   *   shown without adjacent text MUST carry one -- it is the only thing
   *   announcing the outcome.
   */

  /** @type {Props & Record<string, unknown>} */
  let { kind = 'success', title, ...rest } = $props();

  const Component = $derived(kind === 'success' ? Success : Failure);
</script>

<Component
  class="k-moment k-moment--{kind}"
  role={title ? 'img' : undefined}
  aria-label={title}
  aria-hidden={title ? undefined : 'true'}
  focusable="false"
  {...rest}
/>
