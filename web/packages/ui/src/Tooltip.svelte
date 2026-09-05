<!--
  packages/ui/src/Tooltip.svelte

    <Tooltip text="GI: Geographical Indication, a legal mark of origin">
      {#snippet trigger(props)}
        <button {...props}>GI <Icon name="info" /></button>
      {/snippet}
    </Tooltip>

  A tooltip is supplementary, never the only place a fact lives -- if the
  trigger's own visible text does not already say the important part, this
  is the wrong component (see EmptyState/SectionHeader for content that must
  stand alone). Shows on hover AND focus, dismisses on Escape or blur, and
  wires aria-describedby onto whatever the caller renders as the trigger
  via the spread props rather than this component owning the trigger markup
  -- a tooltip attaches to a button, a link, an icon, anything, so it cannot
  render its own.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';

  interface TriggerProps {
    'aria-describedby': string;
    onmouseenter: () => void;
    onmouseleave: () => void;
    onfocus: () => void;
    onblur: () => void;
    onkeydown: (event: KeyboardEvent) => void;
  }

  interface Props {
    text: string;
    placement?: 'top' | 'bottom';
    trigger: Snippet<[TriggerProps]>;
  }

  let { text, placement = 'top', trigger }: Props = $props();

  const uid = $props.id();
  const id = uid + '-tooltip';
  let visible = $state(false);

  function show(): void {
    visible = true;
  }
  function hide(): void {
    visible = false;
  }
  function onkeydown(event: KeyboardEvent): void {
    if (event.key === 'Escape') hide();
  }

  const triggerProps: TriggerProps = {
    'aria-describedby': id,
    onmouseenter: show,
    onmouseleave: hide,
    onfocus: show,
    onblur: hide,
    onkeydown,
  };
</script>

<span class="k-tooltip-anchor">
  {@render trigger(triggerProps)}
  <span class="k-tooltip k-tooltip--{placement}" id={id} role="tooltip" class:k-tooltip--visible={visible}>
    {text}
  </span>
</span>
