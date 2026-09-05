<!--
  packages/ui/src/Input.svelte

  A bare text input. Usually placed inside FieldGroup's children snippet,
  which supplies `id` and `aria-describedby` -- see FieldGroup.svelte.
-->
<script lang="ts">
  interface Props {
    value?: string;
    type?: 'text' | 'email' | 'tel' | 'search' | 'url' | 'date';
    placeholder?: string;
    disabled?: boolean;
    invalid?: boolean;
    id?: string;
    class?: string;
    [key: string]: unknown;
  }

  let {
    value = $bindable(''),
    type = 'text',
    placeholder,
    disabled = false,
    invalid = false,
    id,
    class: className,
    ...rest
  }: Props = $props();
</script>

<input
  class="k-input {className || ''}"
  class:k-input--invalid={invalid}
  {type}
  {id}
  {placeholder}
  {disabled}
  aria-invalid={invalid || undefined}
  bind:value
  {...rest}
/>
