<!--
  packages/ui/src/Label.svelte

  A standalone label, for the rare case a field is composed by hand instead
  of through FieldGroup (which renders this itself). Association is by
  `for`, always -- a label with no `for` is decoration, not a label.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';
  import { locale } from '@kalakriti/i18n';

  interface Props {
    for: string;
    /** The field is not required. Every other field is assumed required. */
    optional?: boolean;
    class?: string;
    children: Snippet;
  }

  let { for: htmlFor, optional = false, class: className, children }: Props = $props();

  const t = $derived(locale.t);
</script>

<label class="k-label {className || ''}" for={htmlFor}>
  {@render children()}
  {#if optional}
    <span class="k-label__optional">({t('ui.field.optional')})</span>
  {/if}
</label>
