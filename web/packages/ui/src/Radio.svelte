<!--
  packages/ui/src/Radio.svelte

  One option in a group. Group them with a shared `name` and native radio
  semantics -- arrow-key traversal within the group is then free, from the
  platform, not reimplemented.

    <fieldset>
      <legend>Payment terms</legend>
      <Radio name="terms" value="advance" bind:group={terms}>50% advance</Radio>
      <Radio name="terms" value="delivery" bind:group={terms}>On delivery</Radio>
    </fieldset>
-->
<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    name: string;
    value: string;
    group?: string;
    disabled?: boolean;
    id?: string;
    class?: string;
    children: Snippet;
    [key: string]: unknown;
  }

  let {
    name,
    value,
    group = $bindable(''),
    disabled = false,
    id,
    class: className,
    children,
    ...rest
  }: Props = $props();
</script>

<label class="k-radio {className || ''}" class:k-radio--disabled={disabled}>
  <input type="radio" {name} {value} {id} {disabled} bind:group {...rest} />
  <span class="k-radio__dot" aria-hidden="true"></span>
  <span class="k-radio__label">{@render children()}</span>
</label>
