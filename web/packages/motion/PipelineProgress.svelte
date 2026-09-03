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

  const ICONS = { enhance: Enhance, attributes: Attributes, describe: Describe, translate: Translate };

  /** @type {{key: string, label?: string, state: 'pending'|'active'|'complete'|'failed'}[]} */
  export let stages = [];
</script>

<div class="k-pipeline" role="list" aria-label="Processing pipeline" {...$$restProps}>
  {#each stages as stage, i (stage.key)}
    <div class="k-pipeline-station" data-state={stage.state} role="listitem" aria-label="{stage.label || stage.key}: {stage.state}">
      <svelte:component this={ICONS[stage.key]} aria-hidden="true" focusable="false" />
    </div>
    {#if i < stages.length - 1}
      <div class="k-pipeline-thread" data-crossed={stage.state === 'complete'} aria-hidden="true"></div>
    {/if}
  {/each}
</div>
