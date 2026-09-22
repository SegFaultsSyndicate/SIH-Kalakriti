<!--
  packages/ui/src/Stepper.svelte

    <Stepper label="Listing progress" steps={[t('capture.step.photo'), t('capture.step.details'), t('capture.step.price')]} current={1} />

  A numbered progress trail for the capture flow and the bulk-order wizard.
  `current` is 0-indexed. Completed steps show a check rather than their
  number -- meaning is never colour alone, so "done" has to look different,
  not just green. Pass `stepHref` to let the user jump back: completed steps
  (and only those -- never forward) become links.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon } from '@kalakriti/icons';

  interface Props {
    /** Names the whole trail for a screen reader, e.g. "Listing progress". */
    label: string;
    steps: string[];
    /** 0-indexed. */
    current: number;
    /** Link target for a completed step; omit for a read-only trail. */
    stepHref?: (index: number) => string;
    class?: string;
  }

  let { label, steps, current, stepHref, class: className }: Props = $props();

  const t = $derived(locale.t);
</script>

{#snippet body(step: string, index: number)}
  <span class="k-stepper__marker" aria-hidden="true">
    {#if index < current}
      <Icon name="check" />
    {:else}
      {index + 1}
    {/if}
  </span>
  <span class="k-stepper__label">{step}</span>
  <span class="k-visually-hidden">
    {t('ui.stepper.stepLabel', { current: index + 1, total: steps.length })}
  </span>
{/snippet}

<ol class="k-stepper {className || ''}" aria-label={label}>
  {#each steps as step, index (index)}
    <li
      class="k-stepper__step"
      class:k-stepper__step--done={index < current}
      class:k-stepper__step--current={index === current}
      aria-current={index === current ? 'step' : undefined}
    >
      {#if stepHref && index < current}
        <a class="k-stepper__link" href={stepHref(index)}>{@render body(step, index)}</a>
      {:else}
        {@render body(step, index)}
      {/if}
    </li>
  {/each}
</ol>
