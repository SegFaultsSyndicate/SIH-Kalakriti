<!--
  packages/ui/src/Switch.svelte

  role="switch" on a native checkbox -- the platform already knows how to
  announce this, and re-deriving it from a div would only lose fidelity.
  On/off is never colour alone: the track carries an "On"/"Off" word too.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';
  import { locale } from '@kalakriti/i18n';

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

  const t = $derived(locale.t);
</script>

<label class="k-switch {className || ''}" class:k-switch--disabled={disabled}>
  <input type="checkbox" role="switch" {id} {disabled} bind:checked {...rest} />
  <span class="k-switch__track" aria-hidden="true">
    <span class="k-switch__thumb"></span>
    <span class="k-switch__state">{checked ? t('ui.switch.on') : t('ui.switch.off')}</span>
  </span>
  <span class="k-switch__label">{@render children()}</span>
</label>
