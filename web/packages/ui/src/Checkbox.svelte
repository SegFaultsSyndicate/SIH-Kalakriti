<!--
  packages/ui/src/Checkbox.svelte

  Native input, custom box drawn in CSS (k-checkbox in ui.css). The check
  mark is drawn with CSS, not an <Icon>, so the checked state never depends
  on the icon set loading.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    checked?: boolean;
    disabled?: boolean;
    id?: string;
    class?: string;
    children: Snippet;
    [key: string]: unknown;
  }

  let {
    checked = $bindable(false),
    disabled = false,
    id,
    class: className,
    children,
    ...rest
  }: Props = $props();
</script>

<label class="k-checkbox {className || ''}" class:k-checkbox--disabled={disabled}>
  <input type="checkbox" {id} {disabled} bind:checked {...rest} />
  <span class="k-checkbox__box" aria-hidden="true"></span>
  <span class="k-checkbox__label">{@render children()}</span>
</label>
