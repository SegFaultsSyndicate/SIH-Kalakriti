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
    /* Stretch the anchor to block-level layout so a full-row trigger
       (accordion toggle etc.) keeps its inline-size: 100%. */
    fill?: boolean;
    trigger: Snippet<[TriggerProps]>;
  }

  let { text, placement = 'top', fill = false, trigger }: Props = $props();

  const uid = $props.id();
  const id = uid + '-tooltip';
  let visible = $state(false);
  /* `placement` is the caller's preference; the chip flips to the other
     side when the preferred side would hang off the viewport (a header
     icon near the top of the page has no room above it, for example).
     Measured against the anchor's live rect, so it reacts to scroll. */
  let resolvedPlacement = $state<'top' | 'bottom'>('top');
  let anchor = $state<HTMLSpanElement>();
  let chip = $state<HTMLSpanElement>();

  const VIEWPORT_MARGIN = 8;

  function roomFor(side: 'top' | 'bottom'): boolean {
    if (!anchor || !chip) return true;
    const bounds = anchor.getBoundingClientRect();
    /* The floating gap under the chip is --k-space-2, which the chip's own
       block padding resolves to; reading it from computed style keeps the
       measurement honest without reimplementing the token. */
    const gap = parseFloat(getComputedStyle(chip).paddingTop) || 8;
    const height = chip.offsetHeight;
    return side === 'top'
      ? bounds.top - gap - height >= VIEWPORT_MARGIN
      : bounds.bottom + gap + height <= window.innerHeight - VIEWPORT_MARGIN;
  }

  /* The chip is portalled to <body> and positioned `fixed` from the anchor's
     live rect, so an ancestor's `overflow: hidden/auto` (a scrolling pill
     row, a rounded card) can never clip it -- CSS alone cannot escape that. */
  let chipStyle = $state('');

  function show(): void {
    resolvedPlacement = roomFor(placement) ? placement : placement === 'top' ? 'bottom' : 'top';
    if (anchor && chip) {
      const r = anchor.getBoundingClientRect();
      const gap = parseFloat(getComputedStyle(chip).paddingTop) || 8;
      const top = resolvedPlacement === 'top' ? r.top - gap : r.bottom + gap;
      const shiftY = resolvedPlacement === 'top' ? '-100%' : '0';
      chipStyle = `position:fixed;inset:auto;top:${top}px;left:${r.left + r.width / 2}px;transform:translate(-50%,${shiftY})`;
    }
    visible = true;
  }
  function hide(): void {
    visible = false;
  }

  /* A fixed chip would stay put while its anchor scrolls away. */
  $effect(() => {
    if (!visible) return;
    addEventListener('scroll', hide, true);
    return () => removeEventListener('scroll', hide, true);
  });

  function portal(node: HTMLElement) {
    document.body.appendChild(node);
    return { destroy: () => node.remove() };
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

<span class="k-tooltip-anchor" class:k-tooltip-anchor--fill={fill} bind:this={anchor}>
  {@render trigger(triggerProps)}
  <span
    class="k-tooltip k-tooltip--{resolvedPlacement}"
    id={id}
    role="tooltip"
    bind:this={chip}
    use:portal
    style={chipStyle}
    class:k-tooltip--visible={visible}
  >
    {text}
  </span>
</span>
