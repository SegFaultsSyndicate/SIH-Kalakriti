<!--
  packages/ui/src/FieldGroup.svelte

    <FieldGroup label="Craft name" description="As printed on your GI certificate" error={errors.craftName}>
      {#snippet children({ id, describedBy })}
        <Input {id} aria-describedby={describedBy} bind:value={craftName} />
      {/snippet}
    </FieldGroup>

  Computes the id and the aria-describedby chain once, in one place, so no
  field wires its own -- the two ways that goes wrong (a stale id after a
  copy-paste, a describedby that omits the error because it was added
  later) are eliminated by construction rather than by review.

  `$props.id()` is Svelte's own per-instance unique id, the same primitive
  React's useId or Solid's createUniqueId are -- reached for here rather
  than a hand-rolled counter or crypto.randomUUID() for exactly the "native
  platform feature covers it" reason those exist.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';
  import Label from './Label.svelte';

  interface Props {
    label: string;
    description?: string;
    error?: string;
    optional?: boolean;
    class?: string;
    children: Snippet<[{ id: string; describedBy: string | undefined }]>;
  }

  let { label, description, error, optional = false, class: className, children }: Props =
    $props();

  const uid = $props.id();
  const fieldId = `${uid}-field`;
  const descId = `${uid}-desc`;
  const errorId = `${uid}-error`;

  const describedBy = $derived(
    [description ? descId : null, error ? errorId : null].filter(Boolean).join(' ') || undefined,
  );
</script>

<div class="k-field-group {className || ''}">
  <Label for={fieldId} {optional}>{label}</Label>
  {@render children({ id: fieldId, describedBy })}
  {#if description}
    <p class="k-field-group__description" id={descId}>{description}</p>
  {/if}
  {#if error}
    <p class="k-field-group__error" id={errorId} role="alert">{error}</p>
  {/if}
</div>
