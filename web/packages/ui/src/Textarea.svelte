<!--
  packages/ui/src/Textarea.svelte
-->
<script lang="ts">
  interface Props {
    value?: string;
    rows?: number;
    placeholder?: string;
    disabled?: boolean;
    invalid?: boolean;
    id?: string;
    class?: string;
    [key: string]: unknown;
  }

  let {
    value = $bindable(),
    rows = 4,
    placeholder,
    disabled = false,
    invalid = false,
    id,
    class: className,
    ...rest
  }: Props = $props();
</script>

<textarea
  class="k-input k-textarea {className || ''}"
  class:k-input--invalid={invalid}
  {id}
  {rows}
  {placeholder}
  {disabled}
  aria-invalid={invalid || undefined}
  bind:value
  {...rest}
></textarea>
