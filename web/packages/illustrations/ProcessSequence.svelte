<!--
  Kalakriti <ProcessSequence /> — Batch 5

    <ProcessSequence craft="weaving" labels={['Warp', 'Weft', 'Beat', 'Finish']} />

  Renders the four process-step illustrations for a craft landing page in
  order. `craft` matches the file prefix (block-printing -> "blockprint" in
  the filename — see manifest.js). Labels are supplied by the caller (real
  text, not baked into the SVG) since the step icons themselves carry no
  text, only a dot-count step marker.
-->
<script>
  import { ILLUSTRATIONS } from './manifest.js';

  const modules = import.meta.glob('./src/*.svg', { eager: true, import: 'default' });

  /**
   * @typedef {object} Props
   * @property {'blockprint'|'weaving'|'pottery'} craft
   * @property {string[]} [labels] Four labels, in step order. From the i18n
   *   layer -- never hardcoded here.
   */

  /** @type {Props & Record<string, unknown>} */
  let { craft, labels = [], ...rest } = $props();

  const steps = $derived(
    ILLUSTRATIONS.filter((i) => i.group === 'process' && i.name.includes(`-${craft}-`)).sort(
      (a, b) => a.name.localeCompare(b.name),
    ),
  );
</script>

<div class="k-process-strip" {...rest}>
  {#each steps as step, i (step.name)}
    {@const Step = modules['./' + step.file]}
    <div class="k-process-step">
      <Step aria-hidden="true" focusable="false" />
      {#if labels[i]}<div class="k-process-step__label">{labels[i]}</div>{/if}
    </div>
  {/each}
</div>

<style>
  .k-process-step__label {
    margin-top: 0.4rem;
    font-size: 0.75rem;
    text-align: center;
    color: var(--k-text-secondary);
  }
</style>
