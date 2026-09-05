<!--
  Kalakriti <PipelineProgress /> — Batch 7

    <PipelineProgress stages={[
      { key: 'enhance',    label: 'Enhance',    state: 'complete' },
      { key: 'attributes', label: 'Attributes', state: 'active' },
      { key: 'describe',   label: 'Describe',   state: 'pending' },
      { key: 'translate',  label: 'Translate',  state: 'pending' },
    ]} />

  `state` per stage is 'pending' | 'active' | 'complete' | 'failed', supplied
  by the caller from real backend state (poll, SSE, websocket — whatever the
  ML service reports). This component NEVER advances a stage on its own: no
  setTimeout, no setInterval, no internal clock. A demo or storybook that
  wants to fake movement does so by changing the `stages` prop from outside,
  which is a different concern from this component's own code containing a
  timer.
-->
<script>
  import Enhance from './src/pipeline-enhance.svg';
  import Attributes from './src/pipeline-attributes.svg';
  import Describe from './src/pipeline-describe.svg';
  import Translate from './src/pipeline-translate.svg';

  const ICONS = {
    enhance: Enhance,
    attributes: Attributes,
    describe: Describe,
    translate: Translate,
  };

  /**
   * @typedef {object} Stage
   * @property {string} key
   * @property {string} [label]
   * @property {'pending'|'active'|'complete'|'failed'} state
   */

  /**
   * @typedef {object} Props
   * @property {Stage[]} [stages] Real backend state. Never a timer.
   * @property {string} label Accessible name for the list, from the i18n
   *   layer -- this used to be a hardcoded English string.
   */

  /** @type {Props & Record<string, unknown>} */
  let { stages = [], label, ...rest } = $props();
</script>

<div class="k-pipeline" role="list" aria-label={label} {...rest}>
  {#each stages as stage, i (stage.key)}
    {@const Station = ICONS[stage.key]}
    <div
      class="k-pipeline-station"
      data-state={stage.state}
      role="listitem"
      aria-label="{stage.label || stage.key}: {stage.state}"
    >
      <Station aria-hidden="true" focusable="false" />
    </div>
    {#if i < stages.length - 1}
      <div
        class="k-pipeline-thread"
        data-crossed={stage.state === 'complete'}
        aria-hidden="true"
      ></div>
    {/if}
  {/each}
</div>
