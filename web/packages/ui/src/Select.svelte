<!--
  packages/ui/src/Select.svelte

  A native <select>, not a custom listbox. A real select gets keyboard
  support, type-ahead, and the platform picker on Android for free -- a
  custom one would have to rebuild all three to reach the same bar.
-->
<script lang="ts">
  import { Icon } from '@kalakriti/icons';

  interface Option {
    value: string;
    label: string;
  }

  interface Props {
    value?: string;
    options: Option[];
    disabled?: boolean;
    invalid?: boolean;
    id?: string;
    class?: string;
    [key: string]: unknown;
  }

  let {
    value = $bindable(''),
    options,
    disabled = false,
    invalid = false,
    id,
    class: className,
    ...rest
  }: Props = $props();
</script>

<div class="k-select {className || ''}" class:k-input--invalid={invalid}>
  <select class="k-input k-select__control" {id} {disabled} aria-invalid={invalid || undefined} bind:value {...rest}>
    {#each options as option (option.value)}
      <option value={option.value}>{option.label}</option>
    {/each}
  </select>
  <Icon name="chevron-down" class="k-select__chevron" />
</div>
