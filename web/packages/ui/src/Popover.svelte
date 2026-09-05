<!--
  packages/ui/src/Popover.svelte

    <Popover bind:open={menuOpen}>
      {#snippet trigger(props)}
        <Button icon="more-vertical" label="More actions" {...props} />
      {/snippet}
      {#snippet children()}
        <MenuItem onclick={...}>Edit</MenuItem>
      {/snippet}
    </Popover>

  Positioned with plain CSS (absolute, anchored to the wrapper) rather than
  a floating-ui-style collision engine -- this system has no menus that open
  near a screen edge in a way that needs one, and `align="end"` covers the
  one case (a trailing icon button) that would otherwise clip. Closes on
  Escape or an outside click/pointerdown, and returns focus to whatever was
  focused before it opened, same as Dialog.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';

  interface TriggerProps {
    'aria-expanded': boolean;
    'aria-haspopup': 'dialog';
    onclick: () => void;
  }

  interface Props {
    open?: boolean;
    align?: 'start' | 'end';
    trigger: Snippet<[TriggerProps]>;
    children: Snippet;
  }

  let { open = $bindable(false), align = 'start', trigger, children }: Props = $props();

  let wrapper: HTMLDivElement;
  let panel = $state<HTMLDivElement>();
  let previousActive: HTMLElement | null = null;

  function toggle(): void {
    if (open) close();
    else openPopover();
  }

  function openPopover(): void {
    previousActive = document.activeElement as HTMLElement | null;
    open = true;
  }

  function close(): void {
    open = false;
    previousActive?.focus();
  }

  function onDocumentPointerDown(event: PointerEvent): void {
    if (open && wrapper && !wrapper.contains(event.target as Node)) close();
  }

  function onkeydown(event: KeyboardEvent): void {
    if (event.key === 'Escape') {
      event.preventDefault();
      close();
    }
  }

  $effect(() => {
    if (!open) return;
    document.addEventListener('pointerdown', onDocumentPointerDown);
    panel?.focus();
    return () => document.removeEventListener('pointerdown', onDocumentPointerDown);
  });

  const triggerProps: TriggerProps = {
    get 'aria-expanded'() {
      return open;
    },
    'aria-haspopup': 'dialog',
    onclick: toggle,
  };
</script>

<div class="k-popover-wrapper" bind:this={wrapper}>
  {@render trigger(triggerProps)}
  {#if open}
    <div
      class="k-popover k-popover--{align}"
      role="dialog"
      bind:this={panel}
      tabindex="-1"
      {onkeydown}
    >
      {@render children()}
    </div>
  {/if}
</div>
