<!--
  packages/ui/src/CraftTerm.svelte

    <p>Woven using the <CraftTerm term={{ term: 'Ikat', canonicalName: 'Ikat',
      region: 'Odisha', giStatus: 'GI registered' }} /> technique.</p>

  A term flagged "do not translate" in a description: `term.term` is rendered
  verbatim inline, with a subtle underline marking it as a defined term, and
  a popover giving the canonical name, region and GI status. The term text
  itself never passes through `t()` or any other transform -- that's the
  whole point of "do not translate".
-->
<script lang="ts">
  import { locale, type DntTerm } from '@kalakriti/i18n';
  import Popover from './Popover.svelte';

  interface Props {
    term: DntTerm;
  }

  let { term }: Props = $props();

  const t = $derived(locale.t);
</script>

<Popover align="start">
  {#snippet trigger(props)}
    <button type="button" class="k-craft-term" {...props}>{term.term}</button>
  {/snippet}
  {#snippet children()}
    <div class="k-craft-term-panel">
      <p class="k-craft-term-panel__name">{term.canonicalName}</p>
      {#if term.region}
        <p><span class="k-craft-term-panel__label">{t('craftTerm.region')}:</span> {term.region}</p>
      {/if}
      {#if term.giStatus}
        <p>
          <span class="k-craft-term-panel__label">{t('craftTerm.giStatus')}:</span>
          {term.giStatus}
        </p>
      {/if}
    </div>
  {/snippet}
</Popover>

<style>
  .k-craft-term {
    font: inherit;
    color: inherit;
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    text-decoration: underline;
    text-decoration-style: dotted;
    text-decoration-color: var(--k-border-interactive);
    text-underline-offset: 0.15em;
  }

  .k-craft-term-panel {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    padding: var(--k-space-3);
    max-inline-size: 18rem;
    font-size: var(--k-text-sm);
  }

  .k-craft-term-panel__name {
    font-weight: var(--k-weight-semibold);
  }

  .k-craft-term-panel__label {
    color: var(--k-text-secondary);
  }
</style>
